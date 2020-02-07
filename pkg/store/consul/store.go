package consul

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/hashicorp/consul/api"
	"meera.tech/envs/pkg/store"
)

type cs struct {
	prefix string

	client *api.Client
}

func (c *cs) Set(key store.Key, val interface{}) error {
	kv := c.client.KV()

	p, err := toKVPair(parseKey(c.prefix, key), val)
	if err != nil {
		return err
	}

	if _, err := kv.Put(p, nil); err != nil {
		return err
	}

	return nil
}

func (c *cs) Get(key store.Key) (interface{}, error) {
	kv := c.client.KV()
	pair, _, err := kv.Get(parseKey(c.prefix, key), nil)
	if err != nil {
		return nil, err
	}

	return fromKVPair(pair)
}

func (c *cs) Close() error { return nil }

func toKVPair(key string, val interface{}) (*api.KVPair, error) {
	var bVal []byte
	var flag uint64
	switch reflect.TypeOf(val).Kind() {
	case reflect.String:
		flag = 0x1
		bVal = []byte(val.(string))
	default:
		return nil, fmt.Errorf("%w %T", store.ErrUnsupportedValueType, val)
	}

	return &api.KVPair{
		Key:   key,
		Flags: flag,
		Value: bVal,
	}, nil
}

func fromKVPair(p *api.KVPair) (interface{}, error) {
	var val interface{}
	switch p.Flags {
	case 0x01:
		val = string(p.Value)
	default:
		return nil, fmt.Errorf("%w %0x", store.ErrUnsupportedValueType, p.Flags)
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
	// E.g for dsn: http://localhost:8500/prefix/for/key
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	client, err := api.NewClient(&api.Config{
		Address: u.Host,
		Scheme:  u.Scheme,
	})
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimPrefix(u.Path, "/")
	return &cs{
		client: client,
		prefix: prefix,
	}, nil
}
