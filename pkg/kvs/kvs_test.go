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
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvoKind,
				Name: "title",
			},
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "at",
			},
			Value:      "atAT",
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvfKind,
				Name: "length",
			},
			Value:      "content of /tmp/path/to/length/file",
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "_did",
			},
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "cRoSs",
			},
			Value:      "cross",
			actionType: actionOverwrite,
		},
		{
			Key: Key{
				Kind: EnvfKind,
				Name: "an",
			},
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "Albatross",
			},
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "crossbow",
			},
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvfKind,
				Name: "ALBATROSS",
			},
			Value:      nonePlaceHolder,
			actionType: actionDefault,
		},
		{
			Key: Key{
				Kind: EnvfKind,
				Name: "everywhere",
			},
			Value:      "content of /tmp/path/to/everywhere/file",
			actionType: actionOverwrite,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "prefixKey",
			},
			Value:      nonePlaceHolder,
			actionType: actionPrefix,
		},
		{
			Key: Key{
				Kind: EnvKind,
				Name: "prefixKey1",
			},
			Value:      nonePlaceHolder,
			actionType: actionPrefix,
		},
	}

	require.Equal(t, expected, kvs)
}

func TestRender(t *testing.T) {
	doc := docOfChapter01(t)

	mockCtrl := gomock.NewController(t)
	s := NewMockKVStore(mockCtrl)

	se := s.EXPECT()
	se.Get(Key{Kind: EnvKind, Name: "poet"}, false).Return("poet", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "title"}, false).Return("title", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "at"}, false).Return("", ErrNotFound).AnyTimes()
	se.Set(Key{Kind: EnvKind, Name: "at"}, "atAT").Return(nil).AnyTimes()
	se.Get(Key{Kind: EnvfKind, Name: "length"}, false).Return("", ErrNotFound).AnyTimes()
	se.Set(Key{Kind: EnvfKind, Name: "length"}, "content of /tmp/path/to/length/file").Return(nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "_did"}, false).Return("did", nil).AnyTimes()
	se.Set(Key{Kind: EnvKind, Name: "cRoSs"}, "cross").Return(nil).AnyTimes()
	se.Get(Key{Kind: EnvfKind, Name: "an"}, false).Return("an", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "Albatross"}, false).Return("${env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "nestedAlbatross"}, false).Return("nested Albatross", nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "crossbow"}, false).Return("crossbow", nil).AnyTimes()
	se.Get(Key{Kind: EnvfKind, Name: "ALBATROSS"}, false).Return("ALBATROSS", nil).AnyTimes()
	se.Set(Key{Kind: EnvfKind, Name: "everywhere"}, "content of /tmp/path/to/everywhere/file").Return(nil).AnyTimes()
	se.Get(Key{Kind: EnvKind, Name: "prefixKey"}, true).Return(`{"key1":"val1","key2":"val2"}`, nil)
	se.Get(Key{Kind: EnvKind, Name: "prefixKey1"}, true).Return(`{"key1":"val1","key2":"val2"}`, nil)

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
    where: "%s/tmp-4.out"
    nor: "any drop to drink"

prefix:
  - '{"key1":"val1","key2":"val2"}'
  - "{"key1":"val1","key2":"val2"}"
`, os.TempDir(), os.TempDir(), os.TempDir(), os.TempDir())

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
		`Hi, this is ${env:// .emptyDefault | default '' }, I'm speaking to ${env:// .empty | overwrite '' }'`,
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
		`Hi, this is ${envf:// .config }, I'm speaking to ${env:// .clientID | default ~!@#$%^&*()_+-={}[]|\:";'<>?,./'" }'`,
		`Hi, this is ${envf:// .config }, I'm speaking to ${env:// .clientID | prefix }'`,
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

func TestKeyVals_MarshalJSON(t *testing.T) {
	cases := []struct {
		in       KeyVals
		expected string
	}{
		{
			in: KeyVals{{
				Key:   Key{Name: `field1`},
				Value: `string`,
			}},
			expected: `{"field1":"string"}`,
		},
		{
			in: KeyVals{{
				Key:   Key{Name: `field1`},
				Value: `"string"`,
			}},
			expected: `{"field1":"string"}`,
		},
		{
			in: KeyVals{
				{
					Key:   Key{Name: `field2`},
					Value: `["a"]`,
				},
				{
					Key:   Key{Name: `field1`},
					Value: `string`,
				},
			},
			expected: `{"field1":"string","field2":["a"]}`,
		},
		{
			in: KeyVals{{
				Key:   Key{Name: `field1`},
				Value: `true`,
			}},
			expected: `{"field1":true}`,
		},
		{
			in: KeyVals{{
				Key:   Key{Name: `field1`},
				Value: `"true"`,
			}},
			expected: `{"field1":"true"}`,
		},
	}

	for _, v := range cases {
		out, err := v.in.MarshalJSON()
		require.Nil(t, err, err)
		require.Equal(t, v.expected, string(out), string(out))
	}
}
