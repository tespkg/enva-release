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
	envKind   = "env"
	envfKind  = "envf"
	envoKind  = "envo"
	envofKind = "envof"
)

var (
	leftDlims  = []string{"${env://", "${envf://", "${envo://", "${envof://"}
	rightDlims = []string{"}", "}", "}", "}"}
)

var (
	envKeyRegex      = regexp.MustCompile(`\${env(f|o|of)?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *}`)
	envFilenameRegex = regexp.MustCompile(`\${envfn: *([-_a-zA-Z0-9]*) *}`)
)

type KVStore interface {
	Get(key Key) (string, error)
}

type Key struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type KeyVal struct {
	Key
	Value string `json:"value"`
}

type KeyVals []KeyVal

type tempFunc func(dir, pattern string) (f *os.File, err error)

func Render(s KVStore, ir io.Reader, iw io.Writer) error {
	return render(s, ir, iw, &kvState{}, ioutil.TempFile)
}

func Scan(r io.Reader, scanFilename bool) (KeyVals, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := string(bs)

	var kvs KeyVals
	var isFn bool

	if scanFilename {
		// Check if doc is an envf or not
		fnRes := envFilenameRegex.FindAllStringSubmatch(doc, -1)
		if len(fnRes) > 1 {
			return nil, fmt.Errorf("expected only one {envfn: *}, got %v", len(fnRes))
		}

		// Found a filename annotation
		if len(fnRes) == 1 {
			isFn = true
			fn := fnRes[0][1]
			kvs = append(kvs, KeyVal{
				Key: Key{
					Kind: envfKind,
					Name: fn,
				},
				Value: doc,
			})
		}
	}

	// Scan doc, exact all keys
	keyRes := envKeyRegex.FindAllStringSubmatch(doc, -1)
	for _, keyMatch := range keyRes {
		k := keyFromMatchItem(keyMatch)
		kvs = append(kvs, KeyVal{
			Key:   k,
			Value: "",
		})
		if isFn && (k.Kind == envfKind || k.Kind == envofKind) && k.Name == kvs[0].Name {
			return nil, errors.New("found cycle reference in envf file")
		}
	}

	return kvs, nil
}

func render(s KVStore, ir io.Reader, iw io.Writer, kvS *kvState, tmpFunc tempFunc) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}

	kvs, err := Scan(bytes.NewBuffer(bs), false)
	if err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, kv := range kvs {
		val, err := valueOf(s, kv.Kind, kv.Name, kvS, tmpFunc)
		if err != nil {
			return err
		}
		switch kv.Kind {
		case envKind, envoKind:
			if kv.Kind == envKind && val == "" {
				return fmt.Errorf("got empty value on required env key: %v", kv.Name)
			}
			vars[kv.Name] = val
		case envfKind, envofKind:
			if kv.Kind == envfKind && val == "" {
				return fmt.Errorf("got empty value on required envf key: %v", kv.Name)
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

	doc := string(bs)
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

func valueOf(s KVStore, kind, key string, kvS *kvState, tmpFunc tempFunc) (string, error) {
	value, err := s.Get(Key{
		Kind: kind,
		Name: key,
	})
	if err != nil {
		return "", fmt.Errorf("get value of %v/%v failed: %v", kind, key, err)
	}

	kvS.set(key)

	keyRes := envKeyRegex.FindAllStringSubmatch(value, -1)
	if len(keyRes) == 0 {
		return value, nil
	}

	for _, keyMatch := range keyRes {
		kv := keyFromMatchItem(keyMatch)
		if kvS.exists(kv.Name) {
			return "", fmt.Errorf("cycle key usage found on %v", kv.Name)
		}
	}

	i := bytes.NewBufferString(value)
	out := &bytes.Buffer{}
	if err := render(s, i, out, kvS, tmpFunc); err != nil {
		return "", fmt.Errorf("render nested key: %v failed: %v", value, err)
	}

	return out.String(), nil
}

func keyFromMatchItem(math []string) Key {
	kind, key := envKind+math[1], math[2]
	return Key{
		Kind: kind,
		Name: key,
	}
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
