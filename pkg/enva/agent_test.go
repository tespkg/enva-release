package enva

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/kvs"
)

func TestTermRunProc(t *testing.T) {
	terminate := make(chan exitStatus)

	timer := time.NewTicker(time.Second * 2)
	go func() {
		<-timer.C
		terminate <- newExitStatus(finished, nil)
	}()

	extStatus := runProc([]string{"tail", "-f", "agent_test.go"}, os.Environ(), false, terminate)
	require.Equal(t, finished, extStatus.code)
}

func docOfChapter01(t *testing.T) string {
	bs, err := ioutil.ReadFile("../../testdata/chapter01.yaml")
	require.Nil(t, err)
	return string(bs)
}

func TestAgentRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	s := kvs.NewMockKVStore(mockCtrl)

	se := s.EXPECT()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "tailFilename"}, false).Return("agent_test.go", nil).AnyTimes()

	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "poet"}, false).Return("poet", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "title"}, false).Return("title", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "at"}, false).Return("", kvs.ErrNotFound).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvKind, Name: "at"}, "atAT").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvfKind, Name: "length"}, false).Return("", kvs.ErrNotFound).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvfKind, Name: "length"}, "content of /tmp/path/to/length/file").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "_did"}, false).Return("did", nil).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvKind, Name: "cRoSs"}, "cross").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvfKind, Name: "an"}, false).Return("an", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "Albatross"}, false).Return("${env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "nestedAlbatross"}, false).Return("nested Albatross", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "crossbow"}, false).Return("crossbow", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvfKind, Name: "ALBATROSS"}, false).Return("ALBATROSS", nil).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvfKind, Name: "everywhere"}, "content of /tmp/path/to/everywhere/file").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "prefixKey"}, true).Return(`{"prefixKey/key1":"val1","prefixKey/key2":"val2"}`, nil)
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "prefixKey1"}, true).Return(`{"prefixKey1/key1":"val1","prefixKey1/key2":"val2"}`, nil)

	lengthFile := "/tmp/path/to/length/file"
	err := os.MkdirAll(filepath.Dir(lengthFile), 0755)
	require.Nil(t, err)
	err = ioutil.WriteFile(lengthFile, []byte("content of "+lengthFile), 0755)
	require.Nil(t, err)

	everywhereFile := "/tmp/path/to/everywhere/file"
	err = os.MkdirAll(filepath.Dir(everywhereFile), 0755)
	require.Nil(t, err)
	err = ioutil.WriteFile(everywhereFile, []byte("content of "+everywhereFile), 0755)
	require.Nil(t, err)

	envFilename, err := filepath.Abs("../../testdata/chapter01.yaml")
	require.Nil(t, err)

	renderedEnvFile := ""
	pt := PatchTable{
		create: func(name string) (*os.File, error) {
			if !strings.HasPrefix(name, "/tmp") {
				f, err := ioutil.TempFile("", "enva")
				if err != nil {
					return nil, err
				}
				renderedEnvFile = f.Name()
				return f, nil
			}
			return os.Create(name)
		},
		tplDir: func() string {
			return "/tmp"
		},
	}

	a, err := NewAgent(s, []string{"tail", "-n", "5", "-f", "${env:// .tailFilename }"}, []EnvFile{{Filename: envFilename}}, false, DefaultRetry, pt)
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancelTimer := time.NewTicker(time.Second * 5)
	go func() {
		<-cancelTimer.C
		cancel()
	}()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.Watch(ctx)
	}()

	wg.Wait()
	fmt.Println(renderedEnvFile)
}

func TestIsVarEqual(t *testing.T) {
	cases := []struct {
		name       string
		a          string
		b          string
		prepareFun func()
		expected   bool
	}{
		{
			name:     "1",
			a:        "foo",
			b:        "bar",
			expected: false,
		},
		{
			name:     "2",
			a:        "foo",
			b:        "foo",
			expected: true,
		},
		{
			name: "3",
			a:    "foo",
			b:    "/tmp/envf-123.out",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
			},
			expected: false,
		},
		{
			name: "4",
			a:    "/tmp/envf-123.out",
			b:    "/tmp/envf-345.out",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("123"), 0744)
			},
			expected: true,
		},
		{
			name: "5",
			a:    "/tmp/envf-123.out",
			b:    "/tmp/envf-345.out",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("345"), 0744)
			},
			expected: false,
		},
		{
			name: "6",
			a:    "/tmp/envf-123.out /tmp/envf-345.out",
			b:    "/tmp/envf-456.out /tmp/envf-789.out",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("345"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-456.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-789.out", []byte("345"), 0744)
			},
			expected: true,
		},
		{
			name: "7",
			a:    "/tmp/envf-123.out /tmp/envf-345.out foo",
			b:    "/tmp/envf-456.out /tmp/envf-789.out foo",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("345"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-456.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-789.out", []byte("345"), 0744)
			},
			expected: true,
		},
		{
			name: "8",
			a:    "/tmp/envf-123.out /tmp/envf-345.out foo",
			b:    "/tmp/envf-456.out /tmp/envf-789.out bar",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("345"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-456.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-789.out", []byte("345"), 0744)
			},
			expected: false,
		},
		{
			name: "9",
			a:    "/tmp/envf-123.out /tmp/envf-345.out",
			b:    "/tmp/envf-456.out /tmp/envf-789.out",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("345"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-456.out", []byte("456"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-789.out", []byte("789"), 0744)
			},
			expected: false,
		},
		{
			name: "10",
			a:    "/tmp/envf-123.out /tmp/envf-345.out",
			b:    "/tmp/envf-456.out",
			prepareFun: func() {
				_ = ioutil.WriteFile("/tmp/envf-123.out", []byte("123"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-345.out", []byte("345"), 0744)
				_ = ioutil.WriteFile("/tmp/envf-456.out", []byte("123"), 0744)
			},
			expected: false,
		},
	}
	for _, c := range cases {
		if c.prepareFun != nil {
			c.prepareFun()
		}
		got := isEnvVarEqual(c.a, c.b)
		if got != c.expected {
			fmt.Println(c.a, c.b)
		}
		require.Equal(t, c.expected, got, c.name)
	}
}
