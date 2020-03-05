package kvs

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sync"
)

const (
	EnvKind  = "env"
	EnvfKind = "envf"
)

var (
	ErrNotFound = errors.New("not found")
)

var (
	leftDlims  = []string{"${env://", "${envf://"}
	rightDlims = []string{"}", "}", "}", "}"}
)

var (
	envKeyRegex = regexp.MustCompile(`\${env(f)?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *(\| *default ([\-./_a-zA-Z0-9]*))? *}`)
)

type KVStore interface {
	Get(key Key) (string, error)
	Set(key Key, val string) error
}

type Key struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (k Key) String() string {
	return k.Kind + "/" + k.Name
}

type KeyVal struct {
	Key
	Value string `json:"value"`
}

func (kv KeyVal) String() string {
	return kv.Kind + "/" + kv.Name + "=" + kv.Value
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

	var kvs KeyVals
	keyRes := envKeyRegex.FindAllStringSubmatch(doc, -1)
	for _, keyMatch := range keyRes {
		k, value := keyFromMatchItem(keyMatch)
		if k.Kind == EnvfKind && value != "" {
			bs, err := readFileFunc(value)
			if err != nil {
				return nil, fmt.Errorf("invalid envf: %v with default: %v, err: %v", k.Name, value, err)
			}
			value = string(bs)
		}
		kvs = append(kvs, KeyVal{
			Key:   k,
			Value: value,
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
		val, err := valueOf(s, kv.Key, kv.Value, kvS, tmpFunc, rdFileFunc)
		if err != nil {
			return err
		}
		switch kv.Kind {
		case EnvKind:
			if val == "" {
				return fmt.Errorf("got empty value on required env key: %v", kv.Key)
			}
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

func valueOf(s KVStore, key Key, defValue string, kvS *kvState, tmpFunc tempFunc, rdFileFunc readFileFunc) (string, error) {
	value, err := s.Get(key)
	if err != nil && !errors.As(err, &ErrNotFound) {
		return "", fmt.Errorf("get valueOf %v failed: %v", key, err)
	}
	if errors.As(err, &ErrNotFound) {
		if defValue == "" {
			return "", fmt.Errorf("get valueOf %v failed: %v", key, err)
		}
		if err := s.Set(key, defValue); err != nil {
			return "", fmt.Errorf("set default valueOf %v failed: %v", key, err)
		}
		value = defValue
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
		return "", fmt.Errorf("render nested key %v failed: %v", value, err)
	}

	return out.String(), nil
}

func keyFromMatchItem(match []string) (Key, string) {
	kind, key, defaultValue := EnvKind+match[1], match[2], match[4]
	return Key{Kind: kind, Name: key}, defaultValue
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
