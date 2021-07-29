package file

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	goyaml "gopkg.in/yaml.v2"
	"tespkg.in/envs/pkg/store"
)

const (
	envKind = "env"
)

var (
	defaultNs = store.DefaultKVNs
)

// ms is a file store for envs running stand alone
type ms struct {
	data map[string]map[string]map[string]interface{}
	sync.RWMutex
	filepath string
}

func (s *ms) Set(key store.Key, val interface{}) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.data[key.Namespace]; !ok {
		s.data[key.Namespace] = make(map[string]map[string]interface{})
	}
	if _, ok := s.data[key.Namespace][key.Kind]; !ok {
		s.data[key.Namespace][key.Kind] = make(map[string]interface{})
	}
	s.data[key.Namespace][key.Kind][key.Name] = val

	return nil
}

func (s *ms) Get(key store.Key) (interface{}, error) {
	s.RLock()
	defer s.RUnlock()

	if v, ok := s.data[key.Namespace]; ok {
		if vv, ok := v[key.Kind]; ok {
			if vvv, ok := vv[key.Name]; ok {
				return vvv, nil
			}
		}
	}

	return nil, store.ErrNotFound
}

func (s *ms) GetNsValues(namespace string) (store.KeyVals, error) {
	s.RLock()
	defer s.RUnlock()

	nsData, ok := s.data[namespace]
	if !ok {
		return nil, store.ErrNotFound
	}
	var kvs store.KeyVals
	for kind, kindData := range nsData {
		for name, valData := range kindData {
			kvs = append(kvs, store.KeyVal{
				Key: store.Key{
					Namespace: namespace,
					Kind:      kind,
					Name:      name,
				},
				Value: valData,
			})
		}
	}

	return kvs, nil
}

func (s *ms) GetKindValues(kind string) (store.KeyVals, error) {
	s.RLock()
	defer s.RUnlock()

	var kvs store.KeyVals
	for namespace, nsData := range s.data {
		for k, kindData := range nsData {
			if k != kind {
				continue
			}
			for name, valData := range kindData {
				kvs = append(kvs, store.KeyVal{
					Key: store.Key{
						Namespace: namespace,
						Kind:      kind,
						Name:      name,
					},
					Value: valData,
				})
			}
		}
	}

	return kvs, nil
}

func (s *ms) GetNsKindValues(namespace, kind string) (store.KeyVals, error) {
	s.RLock()
	defer s.RUnlock()

	nsData, ok := s.data[namespace]
	if !ok {
		return nil, store.ErrNotFound
	}
	kindData, ok := nsData[kind]
	if !ok {
		return nil, store.ErrNotFound
	}
	var kvs store.KeyVals
	for name, val := range kindData {
		kvs = append(kvs, store.KeyVal{
			Key: store.Key{
				Namespace: namespace,
				Kind:      kind,
				Name:      name,
			},
			Value: val,
		})
	}
	return kvs, nil
}

func (s *ms) ListByPrefix(prefix store.Key) (store.KeyVals, error) {
	s.RLock()
	defer s.RUnlock()

	nsData, ok := s.data[prefix.Namespace]
	if !ok {
		return nil, store.ErrNotFound
	}
	kindData, ok := nsData[prefix.Kind]
	if !ok {
		return nil, store.ErrNotFound
	}
	var kvs store.KeyVals
	for name, val := range kindData {
		if !strings.HasPrefix(name, prefix.Name) {
			continue
		}
		kvs = append(kvs, store.KeyVal{
			Key: store.Key{
				Namespace: prefix.Namespace,
				Kind:      prefix.Kind,
				Name:      name,
			},
			Value: val,
		})
	}
	return kvs, nil
}

func (s *ms) Delete(key store.Key) error {
	s.Lock()
	defer s.Unlock()

	nsData, ok := s.data[key.Namespace]
	if !ok {
		return store.ErrNotFound
	}
	kindData, ok := nsData[key.Kind]
	if !ok {
		return store.ErrNotFound
	}
	delete(kindData, key.Name)
	return nil
}

func (s *ms) Close() error {
	data, err := s.data2Yaml()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.filepath, data, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (s *ms) data2Yaml() ([]byte, error) {
	storeKVals, _ := s.GetKindValues(envKind)
	kvsKVals := storeKVs2kvs(storeKVals)
	return yaml.Marshal(kvsKVals)
}

func (s *ms) yaml2Data(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	storeKVals, err := readKvs(bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	for _, kval := range storeKVals {
		if err := s.Set(kval.Key, kval.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *ms) init(dsn string) error {
	s.data = make(map[string]map[string]map[string]interface{})
	path, vals, err := getFilePath(dsn)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(path), 0666)
	_, err = os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		err = s.yaml2Data(path)
		if err != nil {
			return err
		}
	}
	s.filepath = path
	if ns := vals.Get("ns"); ns != "" {
		defaultNs = ns
	}
	return nil
}

// NewStore get a new file store
func NewStore(dsn string) (store.Store, error) {
	s := &ms{}
	if err := s.init(dsn); err != nil {
		return nil, err
	}
	return s, nil
}

func getFilePath(dsn string) (string, url.Values, error) {
	dsn = strings.TrimPrefix(dsn, "file://")
	paths := strings.SplitN(dsn, "?", 2)
	var (
		path  string
		query url.Values
		err   error
	)
	if len(paths) >= 1 {
		path, err = filepath.Abs(paths[0])
		if err != nil {
			return "", nil, err
		}
	}
	if len(paths) >= 2 {
		query, err = url.ParseQuery(paths[1])
		if err != nil {
			return "", nil, err
		}
	}
	return path, query, nil
}

func storeKVs2kvs(kvals store.KeyVals) store.KeyVals {
	specKVals := store.KeyVals{}
	for _, kval := range kvals {
		specKVals = append(specKVals, storeKV2kv(kval))
	}
	return specKVals
}

func storeKV2kv(kval store.KeyVal) store.KeyVal {
	return store.KeyVal{
		Key: store.Key{
			Kind: kval.Kind,
			Name: kval.Name,
		},
		Value: kval.Value.(string),
	}
}

func kvs2storeKVs(kvals store.KeyVals) store.KeyVals {
	storeKVals := store.KeyVals{}
	for _, kval := range kvals {
		storeKVals = append(storeKVals, kv2storeKV(kval))
	}
	return storeKVals
}

func kv2storeKV(kval store.KeyVal) store.KeyVal {
	return store.KeyVal{
		Key: store.Key{
			Namespace: defaultNs,
			Kind:      kval.Kind,
			Name:      kval.Name,
		},
		Value: kval.Value,
	}
}

func readKvs(r io.Reader) (store.KeyVals, error) {
	dec := goyaml.NewDecoder(r)
	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := goyaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}

	kvs := store.KeyVals{}
	for _, out := range res {
		vals := store.KeyVals{}
		if err := yaml.Unmarshal(out, &vals); err != nil {
			return nil, err
		}
		kvs = append(kvs, vals...)
	}
	return kvs2storeKVs(kvs), nil
}
