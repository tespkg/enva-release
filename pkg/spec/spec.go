package spec

import (
	"tespkg.in/envs/pkg/store"
)

type Spec struct {
	Name      string
	Filenames []string
	Documents []string
}

type Specs []Spec

type Handler struct {
	s store.Store
}

func NewHandler(s store.Store) Handler {
	return Handler{s: s}
}

func (h Handler) GetSpecs() (Specs, error) {
	panic("implement me")
}

func (h Handler) GetSpec(name string) (Spec, error) {
	panic("implement me")
}

func (h Handler) Register(spec Spec, typ string, overwrite bool) error {
	panic("implement me")
}
