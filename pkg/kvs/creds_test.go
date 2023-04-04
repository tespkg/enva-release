package kvs

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreds(t *testing.T) {

	cases := []struct {
		key string
		str string
	}{
		{"123", "hello world"},
		{"345", "can ai change the world?"},
		{"sksjfjsla", "if ai can change the world is unknown yet, however it can definitely improve the productivity"},
		{"sksjfjsla", ""},
	}

	for i, c := range cases {
		name := strconv.Itoa(i)
		t.Run(name, func(t *testing.T) {
			creds, err := newCreds(c.key)
			require.Nil(t, err)
			got, err := creds.Encrypt(c.str)
			require.Nil(t, err)
			got1, err := creds.Encrypt(c.str)
			require.NotEqual(t, got, got1)
			out, err := creds.Decrypt(got)
			require.Nil(t, err)
			require.Equal(t, c.str, out)
		})
	}
}
