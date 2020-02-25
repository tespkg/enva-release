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
)

type es struct {
	prefix string

	db *clientv3.Client
}

func (s *es) GetNsValues(namespace string) (store.KeyVals, error) {
	panic("implement me")
}

func (s *es) GetKindValues(kind string) (store.KeyVals, error) {
	panic("implement me")
}

func (s *es) GetNsKindValues(namespace, kind string) (store.KeyVals, error) {
	panic("implement me")
}

func (s *es) Set(key store.Key, val interface{}) error {
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

func parseKey(prefix string, key store.Key) string {
	var keys []string
	if prefix != "" {
		keys = append(keys, prefix)
	}
	if key.Namespace != "" {
		keys = append(keys, key.Namespace)
	}
	if key.Kind != "" {
		keys = append(keys, key.Kind)
	}
	if key.Name != "" {
		keys = append(keys, key.Name)
	}
	return strings.Join(keys, "/")
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
