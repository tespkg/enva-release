package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"meera.tech/envs/pkg/store"
)

const (
	s = `
version: "3"
foo: {%% .ENV_test_foo %%}
bar: %% .ENV_dev_bar %%
`
)

func TestRegexInspectFile(t *testing.T) {
	a := regexp.MustCompile(`\%\% \.(ENV)_([a-z].*)_([a-z].*) \%\%`)
	res := a.FindAllStringSubmatch(s, -1)

	expected := [][]string{
		{"%% .ENV_test_foo %%", "ENV", "test", "foo"},
		{"%% .ENV_dev_bar %%", "ENV", "dev", "bar"},
	}

	require.Equal(t, expected, res)
}

func TestRegexArgs(t *testing.T) {
	a := regexp.MustCompile(`env://[a-z]*`)
	parts := a.FindAll([]byte(`enva --env-store-dsn http://localhost:8500 \
/usr/local/example-svc --oidc env://sso --ac env://ac --dsn postgres://postgres:password@env://postgres/example?sslmode=disable`), -1)
	expected := []string{
		"env://sso",
		"env://ac",
		"env://postgres",
	}

	var res []string
	for _, part := range parts {
		res = append(res, string(part))
	}
	require.Equal(t, expected, res)
}

func TestRegexReplaceArgs(t *testing.T) {
	a := regexp.MustCompile(`env://[a-z]*`)
	newArgs := a.ReplaceAllFunc(
		[]byte(`enva --env-store-dsn http://localhost:8500 \
/usr/local/example-svc --oidc env://sso --ac env://ac --dsn postgres://postgres:password@env://postgres/example?sslmode=disable`),
		func(bytes []byte) []byte {
			t.Log(string(bytes))
			return bytes
		})

	t.Log(string(newArgs))
}

func TestNilSlice(t *testing.T) {
	var a []*int
	a = append(a, nil, nil)
	t.Log(a)
}

func TestPopulatedVars(t *testing.T) {
	rd := bytes.NewBufferString(s)
	vars := make(map[string]string)

	mockCtrl := gomock.NewController(t)
	s := store.NewMockStore(mockCtrl)

	se := s.EXPECT()
	se.Get(store.Key{Name: "test/foo"}).Return("foo val", nil).AnyTimes()
	se.Get(store.Key{Name: "dev/bar"}).Return("bar val", nil).AnyTimes()

	var err error
	vars, err = fetchVars(s, rd, vars)
	require.Nil(t, err)
	require.Equal(t, map[string]string{
		"ENV_dev_bar":  "bar val",
		"ENV_test_foo": "foo val",
	}, vars)
}

func TestPopulateVars(t *testing.T) {
	wd, _ := os.Getwd()
	relFiles := []string{
		"*.yaml",
		"*.html",
	}
	var absFiles []string
	absFn, _ := filepath.Abs(filepath.Join(wd, "testdata/a/bc/i.yaml"))
	absFiles = append(absFiles, absFn)
	absFn, _ = filepath.Abs(filepath.Join(wd, "testdata/e/j.yaml"))
	absFiles = append(absFiles, absFn)

	mockCtrl := gomock.NewController(t)
	s := store.NewMockStore(mockCtrl)
	se := s.EXPECT()
	se.Get(gomock.Any()).Return("foo", nil).AnyTimes()

	p := patchTable{
		create: func(name string) (file *os.File, err error) {
			name, err = filepath.Rel(wd, name)
			if err != nil {
				return nil, err
			}
			name = "/tmp/genfiles/" + strings.TrimPrefix(name, "testdata")
			if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
				return nil, fmt.Errorf("mkdirall failed: %v", err)
			}
			return os.Create(name)
		},
		tplDir: func() string {
			return ""
		},
	}

	defer func() {
		_ = os.RemoveAll("testdata/genfiles")
	}()

	a, err := newAgent(
		s,
		[]string{
			"/usr/local/example-svc",
			"--oidc env://sso",
			"--ac env://ac",
			"--dsn postgres://postgres:password@env://postgres/example?sslmode=disable",
		},
		absFiles,
		relFiles,
		p,
	)
	require.Nil(t, err)
	require.Equal(t, values(map[string]string{
		"ENV_Swagger_CallbackURL": "foo",
		"ENV_Swagger_JSONURI":     "foo",
		"ENV_dev_bar":             "foo",
		"ENV_test_foo":            "foo",
		"ac":                      "foo",
		"postgres":                "foo",
		"sso":                     "foo",
	}), a.vars)

	finalisedArgs, err := a.populateProcEnvVars(a.vars)
	require.Nil(t, err)
	require.Equal(t, "/usr/local/example-svc --oidc foo --ac foo --dsn postgres://postgres:password@foo/example?sslmode=disable", strings.Join(finalisedArgs, " "))
}

func TestTermRunProc(t *testing.T) {
	terminate := make(chan error)

	timer := time.NewTicker(time.Second * 2)
	go func() {
		<-timer.C
		terminate <- errors.New("test term")
	}()

	err := runProc([]string{"tail", "-f", "agent_test.go"}, terminate)
	require.Equal(t, errors.New("test term"), err)
}

func TestKillRunProc(t *testing.T) {
	terminate := make(chan error)

	timer := time.NewTicker(time.Second * 2)
	go func() {
		<-timer.C
		terminate <- errors.New("test kill")
	}()

	err := runProc([]string{"tail", "-f", "agent_test.go"}, terminate)
	require.Equal(t, errors.New("test kill"), err)
}
