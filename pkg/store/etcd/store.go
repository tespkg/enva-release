package etcd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"tespkg.in/envs/pkg/store"
)

const (
	defaultDialTimeout = 2 * time.Second
	defaultGetTimeout  = 5 * time.Second
	etcdSchema         = "etcd"

	namespacePrefix = "ns-"
	kindPrefix      = "kind-"
)

type es struct {
	prefix string

	db *clientv3.Client
}

func (s *es) Set(key store.Key, val interface{}) error {
	if err := validateKey(key); err != nil {
		return err
	}
	bVal, err := encodeVal(val)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultGetTimeout)
	defer cancel()

	_, err = s.db.Put(ctx, parseKey(s.prefix, key), string(bVal))
	if err != nil {
		return err
	}
	return nil
}

func (s *es) Get(key store.Key) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultGetTimeout)
	defer cancel()

	r, err := s.db.Get(ctx, parseKey(s.prefix, key))
	if err != nil {
		return nil, err
	}
	if r.Count == 0 {
		return nil, store.ErrNotFound
	}
	return decodeVal(r.Kvs[0].Value)
}

func (s *es) GetNsValues(namespace string) (store.KeyVals, error) {
	if namespace == "" {
		return nil, errors.New("empty namespace, not allowed")
	}

	return s.list(strings.Join([]string{s.prefix, addPrefix(namespacePrefix, namespace)}, "/"))
}

func (s *es) GetNsKindValues(namespace, kind string) (store.KeyVals, error) {
	if namespace == "" {
		return nil, errors.New("empty namespace, not allowed")
	}
	if kind == "" {
		return nil, errors.New("empty kind, not allowed")
	}

	return s.list(strings.Join([]string{s.prefix, addPrefix(namespacePrefix, namespace), addPrefix(kindPrefix, kind)}, "/"))
}

func (s *es) list(prefix string) (store.KeyVals, error) {
	r, err := s.db.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var kvals store.KeyVals
	for _, i := range r.Kvs {
		key, err := extractKey(s.prefix, string(i.Key))
		if err != nil {
			return nil, err
		}
		val, err := decodeVal(i.Value)
		if err != nil {
			return nil, err
		}
		kvals = append(kvals, store.KeyVal{
			Key:   key,
			Value: val,
		})
	}

	return kvals, nil
}

func (s *es) GetKindValues(kind string) (store.KeyVals, error) {
	r, err := s.db.Get(context.Background(), s.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var kvals store.KeyVals
	for _, i := range r.Kvs {
		key, err := extractKey(s.prefix, string(i.Key))
		if err != nil {
			return nil, err
		}
		if key.Kind != kind {
			continue
		}

		val, err := decodeVal(i.Value)
		if err != nil {
			return nil, err
		}
		kvals = append(kvals, store.KeyVal{
			Key:   key,
			Value: val,
		})
	}

	return kvals, nil
}

func (s *es) Delete(key store.Key) error {
	_, err := s.db.Delete(context.Background(), parseKey(s.prefix, key), nil)
	return err
}

func (s *es) Close() error {
	return s.db.Close()
}

// Add a byte in front of the val to represent the val type for the serialization purpose.
func encodeVal(val interface{}) ([]byte, error) {
	var bVal []byte
	var flag byte
	switch reflect.TypeOf(val).Kind() {
	case reflect.String:
		flag = 0x1
		bVal = []byte(val.(string))
	default:
		return nil, fmt.Errorf("%w %T", store.ErrUnsupportedValueType, val)
	}

	res := []byte{flag}
	res = append(res, bVal...)

	return res, nil
}

func decodeVal(bVal []byte) (interface{}, error) {
	if len(bVal) == 0 {
		return nil, fmt.Errorf("invalid val")
	}

	var val interface{}
	flag := bVal[0]
	switch flag {
	case 0x01:
		if len(bVal) == 1 {
			val = ""
		} else {
			val = string(bVal[1:])
		}
	default:
		return nil, fmt.Errorf("%w %0x", store.ErrUnsupportedValueType, flag)
	}

	return val, nil
}

func validateKey(key store.Key) error {
	if key.Namespace == "" {
		return errors.New("empty namespace not allowed")
	}
	if key.Kind == "" {
		return errors.New("empty kind not allowed")
	}
	if key.Name == "" {
		return errors.New("empty name not allowed")
	}
	return nil
}

func parseKey(prefix string, key store.Key) string {
	var keys []string
	if prefix != "" {
		keys = append(keys, prefix)
	}
	if key.Namespace != "" {
		keys = append(keys, addPrefix(namespacePrefix, key.Namespace))
	}
	if key.Kind != "" {
		keys = append(keys, addPrefix(kindPrefix, key.Kind))
	}
	if key.Name != "" {
		keys = append(keys, key.Name)
	}
	return strings.Join(keys, "/")
}

func extractKey(prefix, key string) (store.Key, error) {
	newKey := strings.TrimPrefix(strings.TrimPrefix(key, prefix), "/")
	parts := strings.SplitN(newKey, "/", 3)
	if len(parts) != 3 {
		return store.Key{}, fmt.Errorf("got invalid key: %v", key)
	}
	return store.Key{
		Namespace: strings.TrimPrefix(parts[0], namespacePrefix),
		Kind:      strings.TrimPrefix(parts[1], kindPrefix),
		Name:      parts[2],
	}, nil
}

func addPrefix(prefix, s string) string {
	return prefix + s
}

func NewStore(dsn string) (store.Store, error) {
	// E.g for dsn: etcd://localhost:2379/prefix/for/key
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	if u.Scheme != etcdSchema {
		return nil, fmt.Errorf("invalid schema, excepted: %v, got: %v", etcdSchema, u.Scheme)
	}
	if u.Path == "" {
		return nil, errors.New("invalid path, got empty value")
	}

	password, _ := u.User.Password()
	cfg := clientv3.Config{
		Endpoints:   []string{u.Host},
		DialTimeout: defaultDialTimeout,
		Username:    u.User.Username(),
		Password:    password,
	}

	db, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimPrefix(u.Path, "/")
	return &es{
		prefix: prefix,
		db:     db,
	}, nil
}
