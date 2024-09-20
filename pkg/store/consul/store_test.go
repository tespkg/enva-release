package consul

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/store"
)

func getConsulDsn(t *testing.T) string {
	consulDsn := os.Getenv("CONSUL_DSN")
	if consulDsn == "" {
		t.Skip("skipping consul test case, since CONSUL_DSN env not found")
	}
	return consulDsn
}

func TestNewStore(t *testing.T) {
	dsn := getConsulDsn(t)
	u, err := url.Parse(dsn)
	require.Nil(t, err)

	s, err := NewStore(dsn)
	require.Nil(t, err)
	ss := s.(*cs)
	require.Equal(t, strings.TrimPrefix(u.Path, "/"), ss.prefix)
}

func newConsulStore(t *testing.T) *cs {
	dsn := getConsulDsn(t)
	s, err := NewStore(dsn)
	require.Nil(t, err)
	return s.(*cs)
}

func TestSetGet(t *testing.T) {
	cs := newConsulStore(t)
	key := store.Key{
		Kind:      "animal",
		Namespace: "test",
		Name:      "foo",
	}
	value := "bar"

	err := cs.Set(key, value)
	require.Nil(t, err)

	val, err := cs.Get(key)
	require.Nil(t, err)
	require.Equal(t, value, val.(string))
}

func TestSetOverwrite(t *testing.T) {
	cs := newConsulStore(t)
	key := store.Key{
		Kind:      "animal",
		Namespace: "test",
		Name:      "foo",
	}
	vals := []string{"jone", "doe"}
	for _, val := range vals {
		err := cs.Set(key, val)
		require.Nil(t, err)
	}

	val, err := cs.Get(key)
	require.Nil(t, err)
	require.Equal(t, vals[1], val.(string))
}

func setKeyValues(t *testing.T, s *cs) {
	var kvals = store.KeyVals{
		{
			Key: store.Key{
				Namespace: "multi-dev",
				Kind:      "multi-animal",
				Name:      "s1",
			},
			Value: "s1",
		},
		{
			Key: store.Key{
				Namespace: "multi-dev",
				Kind:      "multi-nuts",
				Name:      "s2",
			},
			Value: "s2",
		},
		{
			Key: store.Key{
				Namespace: "multi-test",
				Kind:      "multi-animal",
				Name:      "s3",
			},
			Value: "s3",
		},
		{
			Key: store.Key{
				Namespace: "multi-test",
				Kind:      "multi-nuts",
				Name:      "s4",
			},
			Value: "s4",
		},
	}

	for _, kval := range kvals {
		err := s.Set(kval.Key, kval.Value)
		require.Nil(t, err)
	}
}

func TestGetNsValues(t *testing.T) {
	cs := newConsulStore(t)
	setKeyValues(t, cs)

	kvals, err := cs.GetNsValues("multi-dev")
	require.Nil(t, err)
	expected := store.KeyVals{
		{
			Key: store.Key{
				Namespace: "multi-dev",
				Kind:      "multi-animal",
				Name:      "s1",
			},
			Value: "s1",
		},
		{
			Key: store.Key{
				Namespace: "multi-dev",
				Kind:      "multi-nuts",
				Name:      "s2",
			},
			Value: "s2",
		},
	}
	require.Equal(t, expected, kvals)
}

func TestGetNsKindValues(t *testing.T) {
	cs := newConsulStore(t)
	setKeyValues(t, cs)

	kvals, err := cs.GetNsKindValues("multi-dev", "multi-animal")
	require.Nil(t, err)
	expected := store.KeyVals{
		{
			Key: store.Key{
				Namespace: "multi-dev",
				Kind:      "multi-animal",
				Name:      "s1",
			},
			Value: "s1",
		},
	}
	require.Equal(t, expected, kvals)
}

func TestGetKindValues(t *testing.T) {
	cs := newConsulStore(t)
	setKeyValues(t, cs)

	kvals, err := cs.GetKindValues("multi-animal")
	require.Nil(t, err)
	expected := store.KeyVals{
		{
			Key: store.Key{
				Namespace: "multi-dev",
				Kind:      "multi-animal",
				Name:      "s1",
			},
			Value: "s1",
		},
		{
			Key: store.Key{
				Namespace: "multi-test",
				Kind:      "multi-animal",
				Name:      "s3",
			},
			Value: "s3",
		},
	}
	require.Equal(t, expected, kvals)
}

func TestSetUnsupportedType(t *testing.T) {
	cs := newConsulStore(t)
	key := store.Key{
		Kind:      "",
		Namespace: "test",
		Name:      "foo",
	}
	value := []string{"bar"}

	err := cs.Set(key, value)
	require.NotNil(t, err)
	require.True(t, errors.Is(err, store.ErrUnsupportedValueType))
}

func TestGetUnsupportedType(t *testing.T) {
	p := &api.KVPair{
		Key:   "test",
		Flags: 0x02,
		Value: nil,
	}
	_, err := fromKVPair(p)
	require.NotNil(t, err)
	require.True(t, errors.Is(err, store.ErrUnsupportedValueType))
}

func TestExactKey(t *testing.T) {
	rawKey := "ns/kind/a/b/c"
	key, err := extractKey("", rawKey)
	require.Nil(t, err)
	require.Equal(t, "ns", key.Namespace)
	require.Equal(t, "kind", key.Kind)
	require.Equal(t, "a/b/c", key.Name)
}

func setKeyValuesForPrefix(t *testing.T, s *cs) {
	var kvals = store.KeyVals{
		{
			Key: store.Key{
				Namespace: "prefix-dev",
				Kind:      "prefix-animal",
				Name:      "s/s1",
			},
			Value: "s1",
		},
		{
			Key: store.Key{
				Namespace: "prefix-dev",
				Kind:      "prefix-nuts",
				Name:      "s/s2",
			},
			Value: "s2",
		},
		{
			Key: store.Key{
				Namespace: "prefix-dev",
				Kind:      "prefix-animal",
				Name:      "s/s3",
			},
			Value: "s3",
		},
		{
			Key: store.Key{
				Namespace: "prefix-dev",
				Kind:      "prefix-nuts",
				Name:      "ss4",
			},
			Value: "s4",
		},
	}

	for _, kval := range kvals {
		err := s.Set(kval.Key, kval.Value)
		require.Nil(t, err)
	}
}
func TestListByPrefix(t *testing.T) {
	cs := newConsulStore(t)
	setKeyValuesForPrefix(t, cs)
	kvs, err := cs.ListByPrefix(store.Key{
		Namespace: "prefix-dev",
		Kind:      "prefix-nuts",
		Name:      "s",
	})
	require.Nil(t, err, err)
	require.Equal(t, 2, len(kvs))
	vals := []interface{}{}
	for _, v := range kvs {
		vals = append(vals, v.Value)
	}
	require.ElementsMatch(t, []interface{}{`s2`, `s4`}, vals)
}
