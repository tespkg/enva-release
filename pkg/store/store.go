package store

import "errors"

const (
	DefaultKVNs = "kvs"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrUnsupportedValueType = errors.New("unsupported value type")
)

// Key represents the key to identify a resource in the store.
type Key struct {
	Namespace string
	Kind      string
	Name      string
}

// KeyVal represents the key & value pair in the store.
type KeyVal struct {
	Key
	Value interface{}
}

// KeyVals set of KeyVal
type KeyVals []KeyVal

type Store interface {
	Set(key Key, val interface{}) error
	Get(key Key) (interface{}, error)
	GetNsValues(namespace string) (KeyVals, error)
	GetKindValues(kind string) (KeyVals, error)
	GetNsKindValues(namespace, kind string) (KeyVals, error)
	Delete(key Key) error
	Close() error
}
