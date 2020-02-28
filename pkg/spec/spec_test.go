package spec

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/store"
	"tespkg.in/envs/pkg/store/consul"
)

func mustOpenFile(fn string) *os.File {
	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}
	return f
}

func getConsulDsn(t *testing.T) string {
	consulDsn := os.Getenv("CONSUL_DSN")
	if consulDsn == "" {
		t.Skip("skipping consul test case, since CONSUL_DSN env not found")
	}
	return consulDsn
}

func newStore(t *testing.T) store.Store {
	dsn := getConsulDsn(t)
	s, err := consul.NewStore(dsn)
	require.Nil(t, err)
	return s
}

func TestRegister(t *testing.T) {
	wd, _ := os.Getwd()
	filenames := []string{
		filepath.Join(wd, "../../testdata/app.sh"),
		filepath.Join(wd, "../../testdata/chapter01.yaml"),
	}
	var fds []*os.File
	var rds []io.ReadSeeker
	for _, fn := range filenames {
		fd := mustOpenFile(fn)
		fds = append(fds, fd)
		rds = append(rds, fd)
	}

	defer func() {
		for _, fd := range fds {
			fd.Close()
		}
	}()

	h := Handler{Store: newStore(t)}
	specName := "app"
	err := h.Register(specName, true, filenames, rds...)
	require.Nil(t, err)

	hdrs, err := h.GetSpecHeaders()
	require.Nil(t, err)
	require.True(t, len(hdrs) >= 1)

	spec, err := h.GetSpec(specName)
	require.Nil(t, err)

	// Verify documents contents
	expectedDocs := make(map[string]string)
	for i := range filenames {
		bs, err := ioutil.ReadFile(filenames[i])
		require.Nil(t, err)
		expectedDocs[filenames[i]] = string(bs)
	}
	docs := make(map[string]string)
	for i := range spec.Filenames {
		docs[spec.Filenames[i]] = spec.Documents[i]
	}
	require.Equal(t, expectedDocs, docs)
}
