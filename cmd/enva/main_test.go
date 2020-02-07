package main

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyInspectFiles(t *testing.T) {
	cases := []struct {
		fn          string
		expectedErr bool
	}{
		{fn: "/a/b/c", expectedErr: false},
		{fn: "/a/b/c*", expectedErr: true},
		{fn: "/a/b?/c*", expectedErr: true},
		{fn: "./a/b?/c*", expectedErr: false},
		{fn: "../a/b?/c*", expectedErr: true},
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, _, err := verifyInspectFiles([]string{c.fn})
			if c.expectedErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
