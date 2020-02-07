package main

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexKey(t *testing.T) {
	a := regexp.MustCompile(`^\{\{ \.Consul_([a-z].*)_([a-z].*) \}\}`)
	parts := a.FindStringSubmatch(`{{ .Consul_aaa_bbb }}`)
	require.Equal(t, []string{`{{ .Consul_aaa_bbb }}`, "aaa", "bbb"}, parts)
}

func TestNilSlice(t *testing.T) {
	var a []*int
	a = append(a, nil, nil)
	t.Log(a)
}
