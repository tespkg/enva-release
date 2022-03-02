package ssparser

import (
	"os"
	"strconv"
	"strings"
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
			ss:             "aa'abc'aa",
			expectedTokens: []*token{{typ: tokenLiteral, value: "aa'abc'aa"}},
		},
		{
			ss:             `"aa'abc'aa"`,
			expectedTokens: []*token{{typ: tokenLiteral, value: `"aa'abc'aa"`}},
		},
		{
			ss:             `'{"a": {"b": "c}}"'`,
			expectedTokens: []*token{{typ: tokenString, value: `'{"a": {"b": "c}}"'`}},
		},
		{
			ss:             `'{"a": {"b": "c}}"`,
			expectedTokens: []*token{{typ: tokenString, value: `'{"a": {"b": "c}}"'`}},
			invalid:        true,
		},
		{
			ss:      `{"a": {"b": "c"}}"`,
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
		{
			ss:       `'{"a": {"b": "c"}}"'`,
			expected: `{"a": {"b": "c"}}"`,
		},
		{
			ss:       `'{"a": {"b": "${TEST}"}}"'`,
			expected: `{"a": {"b": "${TEST}"}}"`,
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

func TestSplitN(t *testing.T) {
	kv := "ssoHTTPAddr=sso-be.dev-meeraspace-sso:5556"
	ii := strings.SplitN(kv, "=", 2)
	println(len(ii))
	for i, v := range ii {
		println(i, v)
	}
	v, err := Parse(ii[1])
	require.Nil(t, err)
	require.Equal(t, v, ii[1])
}
