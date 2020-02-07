package store

import "github.com/pkg/errors"

var (
	ErrNotFound             = errors.New("not found")
	ErrUnsupportedValueType = errors.New("unsupported value type")
)

// Key represents the key to identify a resource in the store.
type Key struct {
	Kind      string
	Namespace string
	Name      string
}

type Store interface {
	Set(key Key, val interface{}) error
	Get(key Key) (interface{}, error)
	Close() error
}
