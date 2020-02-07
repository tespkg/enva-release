package consul

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"meera.tech/envs/pkg/store"
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
		Kind:      "",
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
		Kind:      "",
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
	require.True(t, errors.As(err, &store.ErrUnsupportedValueType))
}

func TestGetUnsupportedType(t *testing.T) {
	p := &api.KVPair{
		Key:   "test",
		Flags: 0x02,
		Value: nil,
	}
	_, err := fromKVPair(p)
	require.NotNil(t, err)
	require.True(t, errors.As(err, &store.ErrUnsupportedValueType))
}
