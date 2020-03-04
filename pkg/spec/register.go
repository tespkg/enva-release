package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/store"
)

const (
	DefaultKVNs     = "kvs"
	specFileKind    = "spec-file"
	specMetaKind    = "spec-meta"
	specMetaKeyName = "meta"
)

// Register save the application spec itself and keys in it to underlying storage.
type Register interface {
	Scan(spec, filename string, r io.Reader) error
	Save(spec, filename string, r io.Reader) error
}

type DefaultRegister struct {
	store.Store
}

// Scan keys in the spec and save them into underlying store.
func (r DefaultRegister) Scan(spec, filename string, ir io.Reader) error {
	keyVals, err := kvs.Scan(ir)
	if err != nil {
		return err
	}
	for _, kv := range keyVals {
		// TODO: find a way to do check & set automatically
		key := store.Key{
			Namespace: DefaultKVNs,
			Kind:      kv.Kind,
			Name:      kv.Name,
		}
		_, err := r.Get(key)
		if err != nil && !errors.As(err, &store.ErrNotFound) {
			return fmt.Errorf("check key %v failed: %T", kv.Key, err)
		}
		if !errors.As(err, &store.ErrNotFound) && key.Kind == kvs.EnvKind {
			// Ignore the existed env key.
			continue
		}
		if err := r.Set(key, kv.Value); err != nil {
			return fmt.Errorf("set key %v failed: %T", kv.Key, err)
		}
	}
	return nil
}

func (r DefaultRegister) Save(spec, filename string, ir io.Reader) error {
	return saveSpecElement(r.Store, spec, filename, ir)
}

func saveSpecElement(s store.Store, spec, fn string, ir io.Reader) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}
	if err := s.Set(store.Key{
		Namespace: spec,
		Kind:      specFileKind,
		Name:      fn,
	}, string(bs)); err != nil {
		return fmt.Errorf("save spec itself failed: %v", err)
	}
	return nil
}

func getSpecElement(s store.Store, spec, fn string) (string, error) {
	val, err := s.Get(store.Key{
		Namespace: spec,
		Kind:      specFileKind,
		Name:      fn,
	})
	if err != nil {
		return "", err
	}
	return val.(string), nil
}

func saveSpecMeta(s store.Store, hdr Header) error {
	bs, err := json.Marshal(&hdr)
	if err != nil {
		return err
	}
	if err := s.Set(store.Key{
		Namespace: hdr.Spec,
		Kind:      specMetaKind,
		Name:      specMetaKeyName,
	}, string(bs)); err != nil {
		return fmt.Errorf("save spec meata failed: %T", err)
	}

	return nil
}

func getSpecMeta(s store.Store, spec string) (Header, error) {
	var hdr Header
	val, err := s.Get(store.Key{
		Namespace: spec,
		Kind:      specMetaKind,
		Name:      specMetaKeyName,
	})
	if err != nil {
		return hdr, err
	}
	if err := json.Unmarshal([]byte(val.(string)), &hdr); err != nil {
		return hdr, err
	}
	return hdr, nil
}
