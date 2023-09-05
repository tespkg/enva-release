package kvs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"

	"tespkg.in/kit/log"
)

const (
	EnvKind  = "env"
	EnvfKind = "envf"

	// EnvkKind is used for secret that needs to be stored & transferred in encrypted fashion
	EnvkKind = "envk"

	// EnvoKind Optional env ONLY used for checking if the value is required or not when rendering
	// The underlying kind is always env for env & envo cases.
	EnvoKind = "envo"

	briefMaxLen = 50

	actionDefault   = "default"
	actionOverwrite = "overwrite"
	actionPrefix    = "prefix"
	actionInline    = "inline"

	empty = `''`
	none  = "nonePlaceHolder"
)

var (
	EnvfKindTmpDir = ""
)

var (
	ErrNotFound = errors.New("not found")
)

var (
	leftDlims  = []string{"${env://", "${envo://", "${envf://", "${envk://"}
	rightDlims = []string{"}", "}", "}", "}"}
)

var (
	envKeyRegex = regexp.MustCompile(`\${env([ofk])?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *(\| *(default|overwrite|prefix|inline) *([~!@#$%^&*()\-_+={}\[\]:";'<>,.?/|\\a-zA-Z0-9]*))? *}`)
)

type KVStore interface {
	Get(key Key, isPrefix bool) (string, error)
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
	return av.Type + " to " + briefOf(av.Value)
}

func (k Key) String() string {
	return k.Kind + "/" + k.Name
}

type KeyVal struct {
	Key
	Value string `json:"value"`
}

func (kv KeyVal) String() string {
	value := kv.Value
	if value == none {
		value = ""
	}
	return kv.Kind + "/" + kv.Name + "=" + value
}

type KeyVals []KeyVal

func (kvs KeyVals) MarshalJSON() ([]byte, error) {
	out := bytes.NewBuffer([]byte{})
	var tmp interface{}
	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Name < kvs[j].Name
	})
	out.WriteString(`{`)
	for k, kv := range kvs {
		out.WriteString(`"`)
		out.WriteString(kv.Name)
		out.WriteString(`":`)
		if err := json.Unmarshal([]byte(kv.Value), &tmp); err != nil {
			marshaled, err := json.Marshal(kv.Value)
			if err != nil {
				return nil, err
			}
			out.Write(marshaled)
		} else {
			out.WriteString(kv.Value)
		}
		if k != len(kvs)-1 {
			out.WriteString(`,`)
		}
	}
	out.WriteString(`}`)
	return []byte(out.String()), nil
}

type RawKeyVal struct {
	KeyVal
	Action
}

type RawKeyVals []RawKeyVal

type tempFunc func(dir, pattern string) (f *os.File, err error)
type readFileFunc func(filename string) ([]byte, error)

func Render(s KVStore, ir io.Reader, iw io.Writer) error {
	cred, err := NewCreds()
	if err != nil {
		return err
	}
	rd := &rendering{
		s:            s,
		kvS:          &kvState{},
		cred:         cred,
		tmpFunc:      ioutil.TempFile,
		readFileFunc: ioutil.ReadFile,
	}
	return rd.render(ir, iw)
}

type rendering struct {
	s            KVStore
	kvS          *kvState
	cred         *Creds
	tmpFunc      tempFunc
	readFileFunc readFileFunc
}

func (rd *rendering) scan(r io.Reader) (RawKeyVals, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := string(bs)

	uniqueKeys := make(map[string]struct{})
	var rkvs RawKeyVals
	keyRes := envKeyRegex.FindAllStringSubmatch(doc, -1)
	for _, keyMatch := range keyRes {
		k, action := keyFromMatchItem(keyMatch)
		defaultValue := strings.TrimSpace(action.Value)
		if k.Kind == EnvfKind && action.Value != "" && action.Value != none {
			// read the default file that represented by action.Value
			b, err := rd.readFileFunc(action.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid envf: %v with %v, err: %v", k.Name, action, err)
			}
			defaultValue = string(b)
		}
		if _, ok := uniqueKeys[k.String()]; ok {
			continue
		}
		uniqueKeys[k.String()] = struct{}{}

		keyVal := KeyVal{
			Key:   k,
			Value: defaultValue,
		}
		rkvs = append(rkvs, RawKeyVal{
			KeyVal: keyVal,
			Action: action,
		})
	}

	return rkvs, nil
}

func (rd *rendering) render(ir io.Reader, iw io.Writer) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}

	rkvs, err := rd.scan(bytes.NewBuffer(bs))
	if err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, rkv := range rkvs {
		val, err := rd.valueof(rkv)
		if err != nil {
			return err
		}
		switch rkv.Kind {
		case EnvKind:
			if val == "" {
				return fmt.Errorf("got empty value on required env key: %v", rkv.Key)
			}
			vars[rkv.Name] = val
		case EnvoKind:
			vars[rkv.Name] = val
		case EnvkKind:
			if val == "" {
				return fmt.Errorf("got empty value on required envk key: %v", rkv.Key)
			}
			out, err := rd.cred.Decrypt(val)
			if err != nil {
				return fmt.Errorf("decrypt secret %v failed: %v", rkv.Name, err)
			}
			vars[rkv.Name] = out
		case EnvfKind:
			if val == "" {
				return fmt.Errorf("got empty value on required envf key: %v", rkv.Key)
			}
			if rkv.Action.Type == actionInline {
				vars[rkv.Name] = val
				continue
			}
			// Create a tmp file save the val as it's content, and set the file name to the key
			pattern := tmpPattern(rkv.Action.Value)
			f, err := rd.tmpFunc(EnvfKindTmpDir, pattern)
			if err != nil {
				return err
			}
			_, err = io.WriteString(f, val)
			if err != nil {
				f.Close()
				return err
			}
			f.Close()
			vars[rkv.Name] = f.Name()
		default:
			return fmt.Errorf("unexpected env key kind: %v", rkv.Kind)
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

func (rd *rendering) valueof(rkv RawKeyVal) (string, error) {
	rk := rkv.Key
	key := rkv.Key
	action := rkv.Action
	if key.Kind == EnvoKind {
		// Optional env ONLY used for checking if the value is required or not when rendering
		// The underlying kind is always env for env & envo cases.
		key.Kind = EnvKind
	}

	var value string
	var err error
	switch action.Type {
	case actionOverwrite:
		val := rkv.KeyVal.Value
		if val == none {
			return "", fmt.Errorf("overwrite with none is not allowed")
		}

		// overwrite with given value
		value, err = rd.set(key, val)
		if err != nil {
			return "", fmt.Errorf("overwrite failed: %w", err)
		}
		log.Debugf("overwrite key %v with value %v, length: %v", rk, briefOf(value), len(value))
	case actionPrefix:
		value, err = rd.get(key, true)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return "", fmt.Errorf("get valueof %v by prefix failed: %w", rk, err)
			}
			return "", nil
		}
	case actionInline:
		if rk.Kind != EnvfKind {
			return "", fmt.Errorf("expect inline on envf")
		}
		switch rkv.Action.Value {
		case none:
			// act like overwrite when action value is not none
			val := rkv.KeyVal.Value
			value, err = rd.set(key, val)
			if err != nil {
				return "", fmt.Errorf("inline overwrie failed: %w", err)
			}
		default:
			value, err = rd.get(key, false)
			if err != nil {
				return "", fmt.Errorf("get valueof %v failed: %w", rk, err)
			}
		}
	default:
		value, err = rd.get(key, false)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("get valueof %v failed: %w", rk, err)
		}
		if errors.Is(err, ErrNotFound) {
			val := rkv.KeyVal.Value
			if rk.Kind == EnvoKind && val == none {
				return "", nil // envo(optional key) have no default value, return directly.
			}

			// set default value
			value, err = rd.set(key, val)
			if err != nil {
				return "", fmt.Errorf("set %v with default: %v failed: %w", rk, briefOf(value), err)
			}
			if val != none {
				log.Debugf("Set key %v with default value %v, length: %v", rk, briefOf(value), len(value))
			}
		}
	}

	rd.kvS.set(key.Name)

	keyRes := envKeyRegex.FindAllStringSubmatch(value, -1)
	if len(keyRes) == 0 {
		return value, nil
	}

	for _, keyMatch := range keyRes {
		kv, _ := keyFromMatchItem(keyMatch)
		if rd.kvS.exists(kv.Name) {
			return "", fmt.Errorf("cycle key usage found on %v", kv.Name)
		}
	}

	i := bytes.NewBufferString(value)
	out := &bytes.Buffer{}
	if err := rd.render(i, out); err != nil {
		return "", fmt.Errorf(`render nested key "%v" failed: %w`, briefOf(value), err)
	}

	return out.String(), nil
}

func (rd *rendering) set(key Key, value string) (string, error) {
	if value == none {
		return "", nil
	}
	if value == empty {
		value = ""
	}
	if err := rd.s.Set(key, value); err != nil {
		return "", fmt.Errorf("set %v with value %v failed: %w", key, briefOf(value), err)
	}
	return value, nil
}

func (rd *rendering) get(key Key, isPrefix bool) (string, error) {
	return rd.s.Get(key, isPrefix)
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

func briefOf(value string) string {
	if len(value) > briefMaxLen {
		return value[:briefMaxLen] + "..."
	}
	return value
}

func keyFromMatchItem(match []string) (Key, Action) {
	kind, key, actionType, actionValue := EnvKind+match[1], match[2], match[4], match[5]
	if match[3] == "" || match[5] == "" {
		actionValue = none
	}
	if actionType == "" {
		actionType = actionDefault
	}
	if actionType != actionDefault && actionType != actionOverwrite && actionType != actionPrefix && actionType != actionInline {
		actionType = actionDefault
	}
	return Key{Kind: kind, Name: key}, Action{Type: actionType, Value: actionValue}
}

func tmpPattern(hint string) string {
	ext := ".out"
	prefix := "envf-*"
	if hint == "" || hint == none {
		return prefix + ext
	}
	fn := filepath.Base(hint)
	i := len(fn) - 1
	for ; i >= 0 && !os.IsPathSeparator(fn[i]); i-- {
		if fn[i] == '.' {
			break
		}
	}
	if i < 0 {
		return prefix + "__" + fn + ext
	}
	if len(fn[i:]) > 1 {
		ext = fn[i:]
	}
	return prefix + "__" + fn[:i] + ext
}
