package kvs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func docOfChapter01(t *testing.T) string {
	bs, err := ioutil.ReadFile("../../testdata/chapter01.yaml")
	require.Nil(t, err)
	return string(bs)
}

func TestScan(t *testing.T) {
	doc := docOfChapter01(t)
	kvs, err := Scan(bytes.NewBufferString(doc), true)
	require.Nil(t, err)

	expected := KeyVals{
		{
			Key: Key{
				Kind: envfKind,
				Name: "chapter01",
			},
			Value: doc,
		},
		{
			Key: Key{
				Kind: envKind,
				Name: "poet",
			},
		},
		{
			Key: Key{
				Kind: envKind,
				Name: "title",
			},
		},
		{
			Key: Key{
				Kind: envoKind,
				Name: "at",
			},
		},
		{
			Key: Key{
				Kind: envofKind,
				Name: "length",
			},
		},
		{
			Key: Key{
				Kind: envKind,
				Name: "_did",
			},
		},
		{
			Key: Key{
				Kind: envKind,
				Name: "cRoSs",
			},
		},
		{
			Key: Key{
				Kind: envfKind,
				Name: "an",
			},
		},
		{
			Key: Key{
				Kind: envKind,
				Name: "Albatross",
			},
		},
		{
			Key: Key{
				Kind: envoKind,
				Name: "crossbow",
			},
		},
		{
			Key: Key{
				Kind: envofKind,
				Name: "ALBATROSS",
			},
		},
	}

	require.Equal(t, expected, kvs)
}

func TestRender(t *testing.T) {
	doc := docOfChapter01(t)

	mockCtrl := gomock.NewController(t)
	s := NewMockKVStore(mockCtrl)

	se := s.EXPECT()
	se.Get(Key{Kind: envfKind, Name: "chapter01"}).Return(doc, nil).AnyTimes()
	se.Get(Key{Kind: envKind, Name: "poet"}).Return("poet", nil).AnyTimes()
	se.Get(Key{Kind: envKind, Name: "title"}).Return("title", nil).AnyTimes()
	se.Get(Key{Kind: envoKind, Name: "at"}).Return("at", nil).AnyTimes()
	se.Get(Key{Kind: envofKind, Name: "length"}).Return("length", nil).AnyTimes()
	se.Get(Key{Kind: envKind, Name: "_did"}).Return("did", nil).AnyTimes()
	se.Get(Key{Kind: envKind, Name: "cRoSs"}).Return("cross", nil).AnyTimes()
	se.Get(Key{Kind: envfKind, Name: "an"}).Return("an", nil).AnyTimes()
	se.Get(Key{Kind: envKind, Name: "Albatross"}).Return("${env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(Key{Kind: envKind, Name: "nestedAlbatross"}).Return("nested Albatross", nil).AnyTimes()
	se.Get(Key{Kind: envoKind, Name: "crossbow"}).Return("", nil).AnyTimes()
	se.Get(Key{Kind: envofKind, Name: "ALBATROSS"}).Return("", nil).AnyTimes()

	idx := 0
	buf := &bytes.Buffer{}
	err := render(s, bytes.NewBufferString(doc), buf, &kvState{}, func(dir, pattern string) (f *os.File, err error) {
		idx++
		return os.Create(fmt.Sprintf("%s/tmp-%d.out", os.TempDir(), idx))
	})
	require.Nil(t, err)

	expected := fmt.Sprintf(`
# ${envfn: chapter01}
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

	// Just for showcase, put a \n in front of the expected when initiating, remove it here.
	expected = strings.TrimPrefix(expected, "\n")
	require.Equal(t, expected, buf.String())
}
