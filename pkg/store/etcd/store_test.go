package etcd

import (
	"errors"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/store"
)

func getEtcdDsn(t *testing.T) string {
	etcdDsn := os.Getenv("ETCD_DSN")
	if etcdDsn == "" {
		t.Skip("skipping etcd test case, since ETCD_DSN env not found")
	}
	return etcdDsn
}

func TestNewEtcdStore(t *testing.T) {
	dsn := getEtcdDsn(t)
	u, err := url.Parse(dsn)
	require.Nil(t, err)

	s, err := NewStore(dsn)
	require.Nil(t, err)
	ss := s.(*es)
	require.Equal(t, strings.TrimPrefix(u.Path, "/"), ss.prefix)
}

func newEtcdStore(t *testing.T) *es {
	dsn := getEtcdDsn(t)
	s, err := NewStore(dsn)
	require.Nil(t, err, err)
	return s.(*es)
}

func TestSetGet(t *testing.T) {
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "test",
		Namespace: "test",
		Name:      "foo",
	}
	value := "bar"

	err := es.Set(key, value)
	require.Nil(t, err)

	val, err := es.Get(key)
	require.Nil(t, err)
	require.Equal(t, value, val.(string))
}

func TestSetOverwrite(t *testing.T) {
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "test",
		Namespace: "test",
		Name:      "foo",
	}
	vals := []string{"jone", "doe"}
	for _, val := range vals {
		err := es.Set(key, val)
		require.Nil(t, err)
	}

	val, err := es.Get(key)
	require.Nil(t, err)
	require.Equal(t, vals[1], val.(string))
}

func TestSet(t *testing.T) {
	cases := []struct {
		val         interface{}
		expectedErr error
	}{
		{"", nil},
		{"a", nil},
		{"1", nil},
	}
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "test",
		Namespace: "test",
		Name:      "foo",
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := es.Set(key, c.val)
			require.Equal(t, c.expectedErr, err)
		})
	}
}

func TestGetNotFound(t *testing.T) {
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "test",
		Namespace: "test",
		Name:      "hello",
	}
	_, err := es.Get(key)
	require.True(t, errors.Is(err, store.ErrNotFound))
}

func setKeyValues(t *testing.T, s *es) {
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
	es := newEtcdStore(t)
	setKeyValues(t, es)

	kvals, err := es.GetNsValues("multi-dev")
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
	sortedKeyVals(expected)
	sortedKeyVals(kvals)
	require.Equal(t, expected, kvals)
}

func TestGetNsKindValues(t *testing.T) {
	es := newEtcdStore(t)
	setKeyValues(t, es)

	kvals, err := es.GetNsKindValues("multi-dev", "multi-animal")
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
	es := newEtcdStore(t)
	setKeyValues(t, es)

	kvals, err := es.GetKindValues("multi-animal")
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
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "dev",
		Namespace: "test",
		Name:      "foo",
	}
	value := []string{"bar"}

	err := es.Set(key, value)
	require.NotNil(t, err)
	require.True(t, errors.Is(err, store.ErrUnsupportedValueType))
}

func TestGetUnsupportedType(t *testing.T) {
	value := "bar"
	bVal := []byte{0x2}
	bVal = append(bVal, []byte(value)...)
	_, err := decodeVal(bVal)
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

func sortedKeyVals(kvs store.KeyVals) {
	sort.Slice(kvs, func(i, j int) bool {
		if kvs[i].Namespace == kvs[j].Namespace {
			if kvs[i].Kind == kvs[j].Kind {
				return kvs[i].Name < kvs[j].Name
			}
			return kvs[i].Kind < kvs[j].Kind
		}
		return kvs[i].Namespace < kvs[j].Namespace
	})
}

func setKeyValuesForPrefix(t *testing.T, s *es) {
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
	cs := newEtcdStore(t)
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
