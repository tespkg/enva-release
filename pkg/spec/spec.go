package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"tespkg.in/envs/pkg/store"
)

type Header struct {
	Spec      string   `json:"spec"`
	Filenames []string `json:"filenames"`
}

type Spec struct {
	Header
	Documents []string `json:"documents"`
}

type Headers []Header
type Specs []Spec

type Handler struct {
	store.Store
}

func NewHandler(s store.Store) Handler {
	return Handler{Store: s}
}

func (h Handler) GetSpecHeaders() (Headers, error) {
	kvals, err := h.GetKindValues(specMetaKind)
	if err != nil {
		return nil, err
	}
	hdrs := Headers{}
	for _, kval := range kvals {
		var hdr Header
		if err := json.Unmarshal([]byte(kval.Value.(string)), &hdr); err != nil {
			return nil, err
		}
		hdrs = append(hdrs, hdr)
	}

	return hdrs, nil
}

func (h Handler) GetSpec(name string) (Spec, error) {
	hdr, err := getSpecMeta(h.Store, name)
	if err != nil {
		return Spec{}, err
	}
	var docs []string
	for _, fn := range hdr.Filenames {
		doc, err := getSpecElement(h.Store, hdr.Spec, fn)
		if err != nil {
			return Spec{}, err
		}
		docs = append(docs, doc)
	}

	return Spec{
		Header:    hdr,
		Documents: docs,
	}, nil
}

func (h Handler) Register(specName string, prune bool, filenames []string, irs ...io.ReadSeeker) error {
	if specName == "" {
		return errors.New("empty spec name")
	}
	if !prune {
		return errors.New("unsupported prune operation yet")
	}

	if len(filenames) != len(irs) {
		return errors.New("invalid filenames & readers")
	}

	kvals, err := h.GetNsKindValues(specName, specFileKind)
	if err != nil {
		return err
	}
	for _, kval := range kvals {
		if err := h.Delete(kval.Key); err != nil {
			return fmt.Errorf("delete key failed: %v", err)
		}
	}

	for i := range filenames {
		fn := strings.TrimPrefix(filenames[i], "/")
		ir := irs[i]
		register := DefaultRegister{spec: specName, Store: h.Store, filename: fn}

		if err := register.Scan(ir); err != nil {
			return fmt.Errorf("scan file: %v failed: %v", fn, err)
		}

		// Save plain spec to underlying store.
		_, err = ir.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}

		if err := register.Save(ir); err != nil {
			return fmt.Errorf("save spec failed: %v", err)
		}
	}

	hdr := Header{
		Spec:      specName,
		Filenames: filenames,
	}

	return saveSpecMeta(h.Store, hdr)
}
