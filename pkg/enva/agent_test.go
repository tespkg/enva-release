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
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "tailFilename"}).Return("agent_test.go", nil).AnyTimes()

	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "poet"}).Return("poet", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "title"}).Return("title", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "at"}).Return("", kvs.ErrNotFound).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvKind, Name: "at"}, "atAT").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvfKind, Name: "length"}).Return("", kvs.ErrNotFound).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvfKind, Name: "length"}, "content of /tmp/path/to/length/file").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "_did"}).Return("did", nil).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvKind, Name: "cRoSs"}, "cross").Return(nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvfKind, Name: "an"}).Return("an", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "Albatross"}).Return("${env://.nestedAlbatross}", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "nestedAlbatross"}).Return("nested Albatross", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvKind, Name: "crossbow"}).Return("crossbow", nil).AnyTimes()
	se.Get(kvs.Key{Kind: kvs.EnvfKind, Name: "ALBATROSS"}).Return("ALBATROSS", nil).AnyTimes()
	se.Set(kvs.Key{Kind: kvs.EnvfKind, Name: "everywhere"}, "content of /tmp/path/to/everywhere/file").Return(nil).AnyTimes()

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

	a, err := NewAgent(s, []string{"tail", "-n", "5", "-f", "${env:// .tailFilename }"}, []string{envFilename}, false, DefaultRetry, pt)
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
