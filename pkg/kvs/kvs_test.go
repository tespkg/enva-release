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
	kvs, err := scan(bytes.NewBufferString(doc), func(filename string) (i []byte, err error) {
		return []byte("content of " + filename), nil
	})
	require.Nil(t, err)

	expected := KeyVals{
		{
			Key: Key{
				Kind: EnvKind,
				Name: "poet",
			},
		},
		{
			Key: Key{
				Kind: EnvoKind,
				Name: "title",
			},
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "at",
			},
			Value: "atAT",
		},
		{
			Key: Key{
				Kind: EnvfKind,
				Name: "length",
			},
			Value: "content of /tmp/path/to/length/file",
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "_did",
			},
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "cRoSs",
			},
		},
		{
			Key: Key{
				Kind: EnvfKind,
				Name: "an",
			},
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "Albatross",
			},
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "crossbow",
			},
		},
		{
			Key: Key{
				Kind: EnvfKind,
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
	se.Get(Key{Kind: EnvKind, Name: "poet"}).Return("poet", nil).AnyTimes()
	se.Get(Key{Kind: EnvoKind, Name: "title"}).Return("title", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "at"}).Return("", ErrNotFound).AnyTimes()
	se.Set(Key{Kind: EnvKind, Name: "at"}, "atAT").Return(nil).AnyTimes()
	se.Get(Key{Kind: EnvfKind, Name: "length"}).Return("", ErrNotFound).AnyTimes()
	se.Set(Key{Kind: EnvfKind, Name: "length"}, "content of /tmp/path/to/length/file").Return(nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "_did"}).Return("did", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "cRoSs"}).Return("cross", nil).AnyTimes()
	se.Get(Key{Kind: EnvfKind, Name: "an"}).Return("an", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "Albatross"}).Return("${env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "nestedAlbatross"}).Return("nested Albatross", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "crossbow"}).Return("crossbow", nil).AnyTimes()
	se.Get(Key{Kind: EnvfKind, Name: "ALBATROSS"}).Return("ALBATROSS", nil).AnyTimes()

	idx := 0
	buf := &bytes.Buffer{}
	err := render(s, bytes.NewBufferString(doc), buf, &kvState{}, func(dir, pattern string) (f *os.File, err error) {
		idx++
		return os.Create(fmt.Sprintf("%s/tmp-%d.out", os.TempDir(), idx))
	}, func(filename string) (i []byte, err error) {
		return []byte("content of " + filename), nil
	})
	require.Nil(t, err)

	expected := fmt.Sprintf(`
poet: "{poet}"
title: "title"
stanza:
  - "atAT"
  - "%s/tmp-1.out"
  - "did"
  - cross
  - %s/tmp-2.out
  - nested Albatross

mariner:
  with: "crossbow"
  shot: "%s/tmp-3.out"

water:
  water:
    where: "everywhere"
    nor: "any drop to drink"
`, os.TempDir(), os.TempDir(), os.TempDir())

	// Just for showcase, put a \n in front of the expected when initiating, remove it here.
	expected = strings.TrimPrefix(expected, "\n")
	require.Equal(t, expected, buf.String())

	// Check envf with default value
	bs, err := ioutil.ReadFile(fmt.Sprintf("%s/tmp-1.out", os.TempDir()))
	require.Nil(t, err)
	require.Equal(t, "content of /tmp/path/to/length/file", string(bs))
}

func TestRegex(t *testing.T) {
	cases := []string{
		`Hi, this is ${env:// .config | default value/of/config }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${env:// .config| default value/of/config }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${env:// .config|default value/of/config }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${env://.config|default value/of/config }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${env://.config|default value/of/config}, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${env:// .config }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${envf:// .config | default /usr/local/config/config-dev.yaml }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${envf:// .config| default /usr/local/config/config.yaml }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${envf:// .config|default /usr/local/config/config.yaml }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${envf://.config|default /usr/local/config/config.yaml }, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${envf://.config|default /usr/local/config/config.yaml}, I'm speaking to ${env:// .clientID | default alice }'`,
		`Hi, this is ${envf:// .config }, I'm speaking to ${env:// .clientID | default alice }'`,
	}

	for idx, c := range cases {
		res := envKeyRegex.FindAllStringSubmatch(c, -1)
		for _, i := range res {
			fmt.Println(idx, len(i), i)
		}
	}
	for idx, c := range cases {
		newStr := envKeyRegex.ReplaceAllStringFunc(c, func(s string) string {
			res := envKeyRegex.FindAllStringSubmatch(s, -1)
			if len(res) == 0 {
				return s
			}
			return fmt.Sprintf("${env%s:// .%s }", res[0][1], res[0][2])
		})
		fmt.Println(idx, " new ", newStr)
	}
}
