package spec

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"

	"tespkg.in/envs/pkg/store"
)

const (
	defaultKVNs = "kvs"
	specKind    = "spec"
)

// Register save the application spec itself and keys in it to underlying storage.
type Register interface {
	Scan(r io.Reader) error
}

type PlainRegister struct {
	es store.Store

	// Represent a project.
	spec     string
	filename string
}

func (p PlainRegister) Scan(ir io.ReadSeeker) error {
	// Scan keys in the spec and save them into underlying store.
	kvs, err := scan(ir, true)
	if err != nil {
		return err
	}
	for _, kv := range kvs {
		if err := p.es.Set(store.Key{
			Namespace: defaultKVNs,
			Kind:      kv.Kind,
			Name:      kv.Name,
		}, kv.Value); err != nil {
			return fmt.Errorf("set key failed: %T", err)
		}
	}

	// Save plain spec to underlying store.
	_, err = ir.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	return saveSpecEle(p.es, p.spec, p.filename, ir)
}

type ZipRegister struct {
	es store.Store

	// ZIP reader
	zr zip.Reader
	// Represent a project.
	spec string
}

func (z ZipRegister) Scan(r io.Reader) error {
	panic("implement me")
}

func saveSpecEle(es store.Store, spec, fn string, ir io.Reader) error {
	bs, err := ioutil.ReadAll(ir)
	if err != nil {
		return err
	}
	if err := es.Set(store.Key{
		Namespace: spec,
		Kind:      specKind,
		Name:      fn,
	}, string(bs)); err != nil {
		return fmt.Errorf("save spec itself failed: %T", err)
	}
	return nil
}
