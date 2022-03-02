package ssparser

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenizer(t *testing.T) {
	var cases = []struct {
		ss             string
		expectedTokens []*token
		invalid        bool
	}{
		{
			ss:             "$TEST",
			expectedTokens: []*token{{typ: tokenVariable, value: "$TEST"}},
		},
		{
			ss:             "abc${TEST}abc",
			expectedTokens: []*token{{typ: tokenLiteral, value: "abc"}, {typ: tokenVariable, value: "${TEST}"}, {typ: tokenLiteral, value: "abc"}},
		},
		{
			ss:      "abc${TESTabc",
			invalid: true,
		},
		{
			ss:      "aa'abc'aa",
			invalid: true,
		},
		{
			ss:      `aa"abc"aa`,
			invalid: true,
		},
	}

	ssTokenizer := shellStringTokenizer()
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tokens, err := ssTokenizer.tokenize(c.ss)
			if c.invalid {
				println(err.Error())
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			require.Equal(t, c.expectedTokens, tokens)
		})
	}
}

func TestParser(t *testing.T) {
	os.Setenv("TEST", "foo")
	var cases = []struct {
		ss       string
		expected string
	}{
		{
			ss:       "$TEST",
			expected: "foo",
		},
		{
			ss:       "${TEST}",
			expected: "foo",
		},
		{
			ss:       "abc_${TEST}_abc_$TEST",
			expected: "abc_foo_abc_foo",
		},
		{
			ss:       "abc_${TEST}_abc_${TEST}",
			expected: "abc_foo_abc_foo",
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := Parse(c.ss)
			require.Nil(t, err)
			require.Equal(t, c.expected, got)
		})
	}
}
