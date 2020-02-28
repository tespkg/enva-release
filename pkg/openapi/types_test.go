package openapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type D1 struct {
	d int
}
type D2 struct {
	d int
}
type D3 struct {
	E int
	D4
}
type D4 struct {
	F int
}

type S0 struct {
	A int `json:"a"`
	B int `json:"b,omitempty"`
	C int
	D1
	D2
	D3
}

type S0s []S0

func TestTyp2Fields(t *testing.T) {
	v := reflect.ValueOf(S0s{})
	fields := typeFields(v)
	expected := Fields{
		{Name: "a", Required: true},
		{Name: "b", Required: false},
		{Name: "C", Required: true},
		{Name: "E", Required: true},
		{Name: "F", Required: true},
	}
	require.Equal(t, len(expected), len(fields))
	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i].Name, fields[i].Name)
		require.Equal(t, expected[i].Required, fields[i].Required)
	}
}
