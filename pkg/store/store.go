package store

import "errors"

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
	GetNsValues(namespace string) ([]Key, []interface{}, error)
	GetKindValues(kind string) ([]Key, []interface{}, error)
	GetNsKindValues(namespace, kind string) ([]Key, []interface{}, error)
	Close() error
}
