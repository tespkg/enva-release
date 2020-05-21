package kvs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"tespkg.in/kit/log"
)

const (
	EnvKind  = "env"
	EnvfKind = "envf"

	// Optional env ONLY used for checking if the value is required or not when rendering
	// The underlying kind is always env for env & envo cases.
	EnvoKind = "envo"

	briefMaxLen = 50

	actionDefault   = "default"
	actionOverwrite = "overwrite"

	defaultEmptyStub = `''`
	nonePlaceHolder  = "nonePlaceHolder"
)

var (
	ErrNotFound = errors.New("not found")
)

var (
	leftDlims  = []string{"${env://", "${envo://", "${envf://"}
	rightDlims = []string{"}", "}", "}"}
)

var (
	envKeyRegex = regexp.MustCompile(`\${env([of])?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *(\| *(default|overwrite) ([~!@#$%^&*()\-_+={}\[\]:";'<>,.?/|\\a-zA-Z0-9]*))? *}`)
)

type KVStore interface {
	Get(key Key) (string, error)
	Set(key Key, val string) error
}

type Key struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type Action struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (av Action) String() string {
	return av.Type + " to" + briefOf(av.Value)
}

func (k Key) String() string {
	return k.Kind + "/" + k.Name
}

type KeyVal struct {
	Key
	Value string `json:"value"`

	actionType string
}

type EnvKeyVal struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (kv KeyVal) String() string {
	value := kv.Value
	if value == nonePlaceHolder {
		value = ""
	}
	return kv.Kind + "/" + kv.Name + "=" + value
}

type KeyVals []KeyVal

type tempFunc func(dir, pattern string) (f *os.File, err error)
type readFileFunc func(filename string) ([]byte, error)

func Render(s KVStore, ir io.Reader, iw io.Writer) error {
	return render(s, ir, iw, &kvState{}, ioutil.TempFile, ioutil.ReadFile)
}

// Scan given reader exact all keys & default values
func Scan(r io.Reader) (KeyVals, error) {
	return scan(r, ioutil.ReadFile)
}

func scan(r io.Reader, readFileFunc readFileFunc) (KeyVals, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := string(bs)

	uniqueKeys := make(map[string]struct{})
	var kvs KeyVals
	keyRes := envKeyRegex.FindAllStringSubmatch(doc, -1)
	for _, keyMatch := range keyRes {
		k, av := keyFromMatchItem(keyMatch)
		if k.Kind == EnvfKind && av.Value != "" && av.Value != nonePlaceHolder {
			bs, err := readFileFunc(av.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid envf: %v with %v, err: %v", k.Name, av, err)
			}
			av.Value = string(bs)
		}
		if _, ok := uniqueKeys[k.String()]; ok {
			continue
		}
		uniqueKeys[k.String()] = struct{}{}

		kvs = append(kvs, KeyVal{
			Key:        k,
			Value:      strings.TrimSpace(av.Value),
			actionType: av.Type,
		})
	}

	return kvs, nil
}

func render(s KVStore, ir io.Reader, iw io.Writer, kvS *kvState, tmpFunc tempFunc, rdFileFunc readFileFunc) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}

	kvs, err := scan(bytes.NewBuffer(bs), rdFileFunc)
	if err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, kv := range kvs {
		val, err := valueOf(s, kv.Key, Action{Type: kv.actionType, Value: kv.Value}, kvS, tmpFunc, rdFileFunc)
		if err != nil {
			return err
		}
		switch kv.Kind {
		case EnvKind:
			if val == "" {
				return fmt.Errorf("got empty value on required env key: %v", kv.Key)
			}
			vars[kv.Name] = val
		case EnvoKind:
			vars[kv.Name] = val
		case EnvfKind:
			if val == "" {
				return fmt.Errorf("got empty value on required envf key: %v", kv.Key)
			}
			// Create a tmp file save the val as it's content, and set the file name to the key
			f, err := tmpFunc("", kv.Kind+"-*")
			if err != nil {
				return err
			}
			_, err = io.WriteString(f, val)
			if err != nil {
				f.Close()
				return err
			}
			f.Close()
			vars[kv.Name] = f.Name()
		default:
			return fmt.Errorf("unexpected env key kind: %v", kv.Kind)
		}
	}

	// Remove default value before perform render
	doc := envKeyRegex.ReplaceAllStringFunc(string(bs), func(s string) string {
		res := envKeyRegex.FindAllStringSubmatch(s, -1)
		if len(res) == 0 {
			return s
		}
		return fmt.Sprintf("${env%s:// .%s }", res[0][1], res[0][2])
	})

	// Render env & envf vars
	for i := range leftDlims {
		buf := &bytes.Buffer{}
		tpl := template.New("")
		tpl, err = tpl.Delims(leftDlims[i], rightDlims[i]).Parse(doc)
		if err != nil {
			return err
		}
		if err := tpl.Execute(buf, vars); err != nil {
			return err
		}
		doc = buf.String()
	}
	if _, err := io.WriteString(iw, doc); err != nil {
		return err
	}

	return nil
}

func valueOf(s KVStore, key Key, av Action, kvS *kvState, tmpFunc tempFunc, rdFileFunc readFileFunc) (string, error) {
	rawKey := key
	if key.Kind == EnvoKind {
		// Optional env ONLY used for checking if the value is required or not when rendering
		// The underlying kind is always env for env & envo cases.
		key.Kind = EnvKind
	}

	var value string
	var err error

	if av.Type == actionOverwrite {
		if av.Value == nonePlaceHolder {
			return "", fmt.Errorf("overwrite with none is not allowed")
		}
		value, err = set(s, key, av.Value)
		if err != nil {
			return "", fmt.Errorf("overwrite %v with %v failed: %w", key, briefOf(value), err)
		}
		log.Infof("Overwrite key %v with value %v, length: %v", rawKey, briefOf(value), len(value))
	} else {
		value, err = s.Get(key)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("get valueOf %v failed: %w", key, err)
		}
		if errors.Is(err, ErrNotFound) {
			if rawKey.Kind == EnvoKind {
				return "", nil
			}
			value, err = set(s, key, av.Value)
			if err != nil {
				return "", fmt.Errorf("set %v with default: %v failed: %w", key, briefOf(value), err)
			}
			if av.Value != nonePlaceHolder {
				log.Infof("Set key %v with default value %v, length: %v", rawKey, briefOf(value), len(value))
			}
		}
	}

	kvS.set(key.Name)

	keyRes := envKeyRegex.FindAllStringSubmatch(value, -1)
	if len(keyRes) == 0 {
		return value, nil
	}

	for _, keyMatch := range keyRes {
		kv, _ := keyFromMatchItem(keyMatch)
		if kvS.exists(kv.Name) {
			return "", fmt.Errorf("cycle key usage found on %v", kv.Name)
		}
	}

	i := bytes.NewBufferString(value)
	out := &bytes.Buffer{}
	if err := render(s, i, out, kvS, tmpFunc, rdFileFunc); err != nil {
		return "", fmt.Errorf(`render nested key "%v" failed: %w`, briefOf(value), err)
	}

	return out.String(), nil
}

func set(s KVStore, key Key, value string) (string, error) {
	if value == nonePlaceHolder {
		return "", nil
	}
	if value == defaultEmptyStub {
		value = ""
	}
	if err := s.Set(key, value); err != nil {
		return "", fmt.Errorf("set %v with value %v failed: %w", key, briefOf(value), err)
	}
	return value, nil
}

func briefOf(value string) string {
	if len(value) > briefMaxLen {
		return value[:briefMaxLen] + "..."
	}
	return value
}

func keyFromMatchItem(match []string) (Key, Action) {
	kind, key, actionType, actionValue := EnvKind+match[1], match[2], match[4], match[5]
	if match[3] == "" || match[5] == "" {
		actionValue = nonePlaceHolder
	}
	if actionType == "" {
		actionType = actionDefault
	}
	if actionType != actionDefault && actionType != actionOverwrite {
		actionType = actionDefault
	}
	return Key{Kind: kind, Name: key}, Action{Type: actionType, Value: actionValue}
}

type kvState struct {
	sync.Mutex

	m map[string]struct{}
}

func (kvs *kvState) set(key string) {
	stateKey := key

	kvs.Lock()
	defer kvs.Unlock()
	if kvs.m == nil {
		kvs.m = make(map[string]struct{})
	}
	kvs.m[stateKey] = struct{}{}
}

func (kvs *kvState) exists(key string) bool {
	stateKey := key

	kvs.Lock()
	defer kvs.Unlock()
	_, ok := kvs.m[stateKey]
	return ok
}
