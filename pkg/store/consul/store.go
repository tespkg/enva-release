package consul

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/hashicorp/consul/api"
	"tespkg.in/envs/pkg/store"
)

const (
	namespacePrefix = "ns-"
	kindPrefix      = "kind-"
)

type cs struct {
	prefix string

	client *api.Client
}

func (c *cs) Set(key store.Key, val interface{}) error {
	if err := validateKey(key); err != nil {
		return err
	}
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
	if pair == nil {
		return nil, store.ErrNotFound
	}

	return fromKVPair(pair)
}

func (c *cs) GetNsValues(namespace string) (store.KeyVals, error) {
	if namespace == "" {
		return nil, errors.New("empty namespace, not allowed")
	}

	return c.list(strings.Join([]string{c.prefix, addPrefix(namespacePrefix, namespace)}, "/"))
}

func (c *cs) GetNsKindValues(namespace, kind string) (store.KeyVals, error) {
	if namespace == "" {
		return nil, errors.New("empty namespace, not allowed")
	}
	if kind == "" {
		return nil, errors.New("empty kind, not allowed")
	}

	return c.list(strings.Join([]string{c.prefix, addPrefix(namespacePrefix, namespace), addPrefix(kindPrefix, kind)}, "/"))
}

func (c *cs) list(prefix string) (store.KeyVals, error) {
	pairs, _, err := c.client.KV().List(prefix, nil)
	if err != nil {
		return nil, err
	}
	var kvals store.KeyVals
	for _, p := range pairs {
		key, err := extractKey(c.prefix, p.Key)
		if err != nil {
			return nil, err
		}
		val, err := fromKVPair(p)
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

func (c *cs) ListByPrefix(prefix store.Key) (store.KeyVals, error) {
	return c.list(parseKey(c.prefix, prefix))
}

func (c *cs) GetKindValues(kind string) (store.KeyVals, error) {
	if kind == "" {
		return nil, errors.New("empty kind, not allowed")
	}

	pairs, _, err := c.client.KV().List(c.prefix, nil)
	if err != nil {
		return nil, err
	}
	var kvals store.KeyVals
	for _, p := range pairs {
		key, err := extractKey(c.prefix, p.Key)
		if err != nil {
			return nil, err
		}
		if key.Kind != kind {
			continue
		}

		val, err := fromKVPair(p)
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

func (c *cs) Delete(key store.Key) error {
	_, err := c.client.KV().Delete(parseKey(c.prefix, key), nil)
	return err
}

func (c *cs) Close() error { return nil }

func toKVPair(key string, val interface{}) (*api.KVPair, error) {
	if val == nil {
		val = ""
	}
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
		return nil, fmt.Errorf("%w %0x %s", store.ErrUnsupportedValueType, p.Flags, p.Key)
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
