package file

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/store"
)

func newMemoryStore(t *testing.T) *ms {
	s, err := NewStore("file://../../../testdata/file.yaml")
	require.Nil(t, err)
	return s.(*ms)
}

func TestStoreRecovery(t *testing.T) {
	s := newMemoryStore(t)
	for i := 1; i <= 5; i++ {
		val, err := s.Get(store.Key{
			Kind:      kvs.EnvKind,
			Namespace: store.DefaultKVNs,
			Name:      fmt.Sprintf("KEY%d", i),
		})
		require.Nil(t, err)
		require.Equal(t, val, fmt.Sprintf("VAL%d", i))
	}
}

func TestSetGet(t *testing.T) {
	es := newMemoryStore(t)
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
	es := newMemoryStore(t)
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
	es := newMemoryStore(t)
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
	es := newMemoryStore(t)
	key := store.Key{
		Kind:      "test",
		Namespace: "test",
		Name:      "hello",
	}
	_, err := es.Get(key)
	require.True(t, errors.Is(err, store.ErrNotFound))
}

func setKeyValues(t *testing.T, s *ms) {
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
	es := newMemoryStore(t)
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
	es := newMemoryStore(t)
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
	es := newMemoryStore(t)
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
	sortedKeyVals(expected)
	sortedKeyVals(kvals)
	require.Equal(t, expected, kvals)
	require.Equal(t, expected, kvals)
}

func setEnvKeys(t *testing.T, s *ms) {
	var kvals = store.KeyVals{
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s1",
			},
			Value: "s1",
		},
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s2",
			},
			Value: "s2",
		},
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s3",
			},
			Value: "s3",
		},
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
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

func TestPersist(t *testing.T) {
	file := filepath.Join(os.TempDir(), "temp.yaml")
	s, err := NewStore("file://" + file)
	defer os.Remove(file)

	require.Nil(t, err)
	ms1 := s.(*ms)
	setEnvKeys(t, ms1)

	err = ms1.Close()
	require.Nil(t, err)

	data, err := ioutil.ReadFile(file)
	require.Nil(t, err)
	require.NotNil(t, data)

	file2 := filepath.Join(os.TempDir(), "temp.yaml")
	s2, err := NewStore("file://" + file2)
	require.Nil(t, err)
	ms2 := s2.(*ms)
	err = ms2.yaml2Data(file)
	require.Nil(t, err)

	kvals, err := ms2.GetNsKindValues(store.DefaultKVNs, kvs.EnvKind)
	require.Nil(t, err)
	expected := store.KeyVals{
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s1",
			},
			Value: "s1",
		},
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s2",
			},
			Value: "s2",
		},
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s3",
			},
			Value: "s3",
		},
		{
			Key: store.Key{
				Namespace: store.DefaultKVNs,
				Kind:      kvs.EnvKind,
				Name:      "s4",
			},
			Value: "s4",
		},
	}
	sortedKeyVals(expected)
	sortedKeyVals(kvals)
	require.Equal(t, expected, kvals)

	data2, err := ms2.data2Yaml()
	require.Nil(t, err)

	require.Equal(t, len(data2), len(data))
}

func TestFilePath(t *testing.T) {
	dsn := "file:///path/to/file?ns=kvs"
	path, vals, err := getFilePath(dsn)
	require.Nil(t, err)
	require.Equal(t, "/path/to/file", path)
	require.Equal(t, vals.Get("ns"), "kvs")

	dsn = "file://../../../testdata/file.yaml?ns=kvs"
	path, vals, err = getFilePath(dsn)
	require.Nil(t, err)
	abs, _ := filepath.Abs("../../../testdata/file.yaml")
	require.Equal(t, abs, path)
	require.Equal(t, vals.Get("ns"), "kvs")
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

func setKeyValuesForPrefix(t *testing.T, s *ms) {
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
	cs := newMemoryStore(t)
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
