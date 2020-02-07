package etcd

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"meera.tech/envs/pkg/store"
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

	s, err := newStore(dsn)
	require.Nil(t, err)
	ss := s.(*es)
	require.Equal(t, strings.TrimPrefix(u.Path, "/"), ss.prefix)
}

func newEtcdStore(t *testing.T) *es {
	dsn := getEtcdDsn(t)
	s, err := newStore(dsn)
	require.Nil(t, err)
	return s.(*es)
}

func TestSetGet(t *testing.T) {
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "",
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
		Kind:      "",
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
		Kind:      "",
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
		Kind:      "",
		Namespace: "test",
		Name:      "hello",
	}
	_, err := es.Get(key)
	require.True(t, errors.As(err, &store.ErrNotFound))
}

func TestSetUnsupportedType(t *testing.T) {
	es := newEtcdStore(t)
	key := store.Key{
		Kind:      "",
		Namespace: "test",
		Name:      "foo",
	}
	value := []string{"bar"}

	err := es.Set(key, value)
	require.NotNil(t, err)
	require.True(t, errors.As(err, &store.ErrUnsupportedValueType))
}

func TestGetUnsupportedType(t *testing.T) {
	value := "bar"
	bVal := []byte{0x2}
	bVal = append(bVal, []byte(value)...)
	_, err := decodeVal(bVal)
	require.NotNil(t, err)
	require.True(t, errors.As(err, &store.ErrUnsupportedValueType))
}
