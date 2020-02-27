package spec

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

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
	Scan(r io.Reader) error
	Save(r io.Reader) error
}

type DefaultRegister struct {
	es store.Store

	// Represent a project.
	spec     string
	filename string
}

func (r DefaultRegister) Scan(ir io.Reader) error {
	// Scan keys in the spec and save them into underlying store.
	kvs, err := scan(ir, true)
	if err != nil {
		return err
	}
	for _, kv := range kvs {
		if err := r.es.Set(store.Key{
			Namespace: DefaultKVNs,
			Kind:      kv.Kind,
			Name:      kv.Name,
		}, kv.Value); err != nil {
			return fmt.Errorf("set key failed: %T", err)
		}
	}
	return nil
}

func (r DefaultRegister) Save(ir io.Reader) error {
	return saveSpecElement(r.es, r.spec, r.filename, ir)
}

func saveSpecElement(es store.Store, spec, fn string, ir io.Reader) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}
	if err := es.Set(store.Key{
		Namespace: spec,
		Kind:      specFileKind,
		Name:      fn,
	}, string(bs)); err != nil {
		return fmt.Errorf("save spec itself failed: %v", err)
	}
	return nil
}

func getSpecElement(es store.Store, spec, fn string) (string, error) {
	val, err := es.Get(store.Key{
		Namespace: spec,
		Kind:      specFileKind,
		Name:      fn,
	})
	if err != nil {
		return "", err
	}
	return val.(string), nil
}

func saveSpecMeta(es store.Store, hdr Header) error {
	bs, err := json.Marshal(&hdr)
	if err != nil {
		return err
	}
	if err := es.Set(store.Key{
		Namespace: hdr.Spec,
		Kind:      specMetaKind,
		Name:      specMetaKeyName,
	}, string(bs)); err != nil {
		return fmt.Errorf("save spec meata failed: %T", err)
	}

	return nil
}

func getSpecMeta(es store.Store, spec string) (Header, error) {
	var hdr Header
	val, err := es.Get(store.Key{
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
