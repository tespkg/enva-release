package spec

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"

	"tespkg.in/envs/pkg/store"
)

var (
	envKeyRegex      = regexp.MustCompile(`{env(f|o|of)?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *}`)
	envFilenameRegex = regexp.MustCompile(`{envfn: *([-_a-zA-Z0-9]*) *}`)
)

type kvPair struct {
	kind string
	spec string
	key  string
	val  string
}

type tempFunc func(dir, pattern string) (f *os.File, err error)

func Render(es store.Store, spec string, ir io.Reader, iw io.Writer) error {
	return render(es, spec, ir, iw, &kvState{}, ioutil.TempFile)
}

func scan(spec string, r io.Reader, scanFilename bool) ([]kvPair, error) {
	if spec == "" {
		return nil, errors.New("got empty spec")
	}

	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := string(bs)

	var kvs []kvPair
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
			kvs = append(kvs, kvPair{
				kind: "envf",
				spec: spec,
				key:  fn,
				val:  doc,
			})
		}
	}

	// Scan doc, exact all keys
	keyRes := envKeyRegex.FindAllStringSubmatch(doc, -1)
	for _, keyMatch := range keyRes {
		kv := kvFromMatchItem(spec, keyMatch)
		kvs = append(kvs, kv)
		if isFn && (kv.kind == "envf" || kv.kind == "envof") && kv.key == kvs[0].key {
			return nil, errors.New("found cycle reference in envf file")
		}
	}

	return kvs, nil
}

func render(es store.Store, spec string, ir io.Reader, iw io.Writer, kvS *kvState, tmpFunc tempFunc) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}

	kvs, err := scan(spec, bytes.NewBuffer(bs), false)
	if err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, kv := range kvs {
		val, err := valueOf(es, spec, kv.kind, kv.key, kvS, tmpFunc)
		if err != nil {
			return err
		}
		switch kv.kind {
		case "env", "envo":
			if kv.key == "env" && val == "" {
				return fmt.Errorf("got empty value on required env key: %v", kv.key)
			}
			vars[kv.key] = val
		case "envf", "envof":
			if kv.key == "envf" && val == "" {
				return fmt.Errorf("got empty value on required envf key: %v", kv.key)
			}
			// Create a tmp file save the val as it's content, and set the file name to the key
			f, err := tmpFunc("", "envf*")
			if err != nil {
				return err
			}
			_, err = io.WriteString(f, val)
			if err != nil {
				f.Close()
				return err
			}
			f.Close()
			vars[kv.key] = f.Name()
		default:
			return fmt.Errorf("unexpected env key kind: %v", kv.kind)
		}
	}

	leftDlims := []string{"{env://", "{envf://", "{envo://", "{envof://"}
	rightDlims := []string{"}", "}", "}", "}"}

	doc := string(bs)
	for i := range leftDlims {
		buf := &bytes.Buffer{}
		tpl := template.New("i")
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

func valueOf(es store.Store, spec, kind, key string, kvS *kvState, tmpFunc tempFunc) (string, error) {
	val, err := es.Get(store.Key{
		Kind:      kind,
		Namespace: spec,
		Name:      key,
	})
	if err != nil {
		return "", err
	}
	value := val.(string)

	kvS.set(spec, key)

	keyRes := envKeyRegex.FindAllStringSubmatch(value, -1)
	if len(keyRes) == 0 {
		return value, nil
	}

	for _, keyMatch := range keyRes {
		kv := kvFromMatchItem(spec, keyMatch)
		if kvS.exists(kv.spec, kv.key) {
			return "", fmt.Errorf("cycle key usage found on %v", kv.key)
		}
	}

	i := bytes.NewBufferString(value)
	out := &bytes.Buffer{}
	if err := render(es, spec, i, out, kvS, tmpFunc); err != nil {
		return "", fmt.Errorf("render nested key: %v failed: %v", value, err)
	}

	return out.String(), nil
}

func kvFromMatchItem(spec string, math []string) kvPair {
	kind, key := "env"+math[1], math[2]
	return kvPair{
		kind: kind,
		spec: spec,
		key:  key,
	}
}

type kvState struct {
	sync.Mutex

	m map[string]struct{}
}

func (kvs *kvState) set(spec, key string) {
	stateKey := strings.Join([]string{spec, key}, "/")

	kvs.Lock()
	defer kvs.Unlock()
	if kvs.m == nil {
		kvs.m = make(map[string]struct{})
	}
	kvs.m[stateKey] = struct{}{}
}

func (kvs *kvState) exists(spec, key string) bool {
	stateKey := strings.Join([]string{spec, key}, "/")

	kvs.Lock()
	defer kvs.Unlock()
	_, ok := kvs.m[stateKey]
	return ok
}
