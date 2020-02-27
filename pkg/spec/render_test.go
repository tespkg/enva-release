package spec

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/store"
)

var doc = `
# {envfn: chapter01}
poet: "{{env://.poet}}"
title: "{env:// .title }"
stanza:
  - "{envo:// .at}"
  - "{envof://.length }"
  - "{env://._did}"
  - {env://.cRoSs}
  - {envf:// .an }
  - {env:// .Albatross }

mariner:
  with: "{envo://.crossbow}"
  shot: "{envof://.ALBATROSS}"

water:
  water:
    where: "everywhere"
    nor: "any drop to drink"
`

func TestScan(t *testing.T) {
	kvs, err := scan(bytes.NewBufferString(doc), true)
	require.Nil(t, err)
	expected := KeyVals{
		{
			Kind:  envfKind,
			Name:  "chapter01",
			Value: doc,
		},
		{
			Kind: envKind,
			Name: "poet",
		},
		{
			Kind: envKind,
			Name: "title",
		},
		{
			Kind: envoKind,
			Name: "at",
		},
		{
			Kind: envofKind,
			Name: "length",
		},
		{
			Kind: envKind,
			Name: "_did",
		},
		{
			Kind: envKind,
			Name: "cRoSs",
		},
		{
			Kind: envfKind,
			Name: "an",
		},
		{
			Kind: envKind,
			Name: "Albatross",
		},
		{
			Kind: envoKind,
			Name: "crossbow",
		},
		{
			Kind: envofKind,
			Name: "ALBATROSS",
		},
	}

	require.Equal(t, expected, kvs)
}

func TestRender(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	s := store.NewMockStore(mockCtrl)

	se := s.EXPECT()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envfKind, Name: "chapter01"}).Return(doc, nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envKind, Name: "poet"}).Return("poet", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envKind, Name: "title"}).Return("title", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envoKind, Name: "at"}).Return("at", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envofKind, Name: "length"}).Return("length", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envKind, Name: "_did"}).Return("did", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envKind, Name: "cRoSs"}).Return("cross", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envfKind, Name: "an"}).Return("an", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envKind, Name: "Albatross"}).Return("{env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envKind, Name: "nestedAlbatross"}).Return("nested Albatross", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envoKind, Name: "crossbow"}).Return("", nil).AnyTimes()
	se.Get(store.Key{Namespace: DefaultKVNs, Kind: envofKind, Name: "ALBATROSS"}).Return("", nil).AnyTimes()

	idx := 0
	buf := &bytes.Buffer{}
	err := render(s, bytes.NewBufferString(doc), buf, &kvState{}, func(dir, pattern string) (f *os.File, err error) {
		idx++
		return os.Create(fmt.Sprintf("%s/tmp-%d.out", os.TempDir(), idx))
	})
	require.Nil(t, err)

	expected := fmt.Sprintf(`
# {envfn: chapter01}
poet: "{poet}"
title: "title"
stanza:
  - "at"
  - "%s/tmp-1.out"
  - "did"
  - cross
  - %s/tmp-2.out
  - nested Albatross

mariner:
  with: ""
  shot: "%s/tmp-3.out"

water:
  water:
    where: "everywhere"
    nor: "any drop to drink"
`, os.TempDir(), os.TempDir(), os.TempDir())

	require.Equal(t, expected, buf.String())
}
