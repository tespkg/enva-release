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
	kvs, err := scan("envs", bytes.NewBufferString(doc), true)
	require.Nil(t, err)
	expected := []kv{
		{
			spec: "envs",
			kind: envfKind,
			key:  "chapter01",
			val:  doc,
		},
		{
			spec: "envs",
			kind: envKind,
			key:  "poet",
		},
		{
			spec: "envs",
			kind: envKind,
			key:  "title",
		},
		{
			spec: "envs",
			kind: envoKind,
			key:  "at",
		},
		{
			spec: "envs",
			kind: envofKind,
			key:  "length",
		},
		{
			spec: "envs",
			kind: envKind,
			key:  "_did",
		},
		{
			spec: "envs",
			kind: envKind,
			key:  "cRoSs",
		},
		{
			spec: "envs",
			kind: envfKind,
			key:  "an",
		},
		{
			spec: "envs",
			kind: envKind,
			key:  "Albatross",
		},
		{
			spec: "envs",
			kind: envoKind,
			key:  "crossbow",
		},
		{
			spec: "envs",
			kind: envofKind,
			key:  "ALBATROSS",
		},
	}

	require.Equal(t, expected, kvs)
}

func TestRender(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	s := store.NewMockStore(mockCtrl)

	se := s.EXPECT()
	se.Get(store.Key{Namespace: "envs", Kind: envfKind, Name: "chapter01"}).Return(doc, nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envKind, Name: "poet"}).Return("poet", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envKind, Name: "title"}).Return("title", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envoKind, Name: "at"}).Return("at", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envofKind, Name: "length"}).Return("length", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envKind, Name: "_did"}).Return("did", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envKind, Name: "cRoSs"}).Return("cross", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envfKind, Name: "an"}).Return("an", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envKind, Name: "Albatross"}).Return("{env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envKind, Name: "nestedAlbatross"}).Return("nested Albatross", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envoKind, Name: "crossbow"}).Return("", nil).AnyTimes()
	se.Get(store.Key{Namespace: "envs", Kind: envofKind, Name: "ALBATROSS"}).Return("", nil).AnyTimes()

	idx := 0
	buf := &bytes.Buffer{}
	err := render(s, "envs", bytes.NewBufferString(doc), buf, &kvState{}, func(dir, pattern string) (f *os.File, err error) {
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
