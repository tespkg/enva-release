package enva

import (
	"context"
	"os"
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
		terminate <- newExitStatus(success, nil)
	}()

	extStatus := runProc([]string{"tail", "-f", "agent_test.go"}, os.Environ(), terminate)
	require.Equal(t, success, extStatus.code)
}

func TestAgentRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	s := kvs.NewMockKVStore(mockCtrl)

	se := s.EXPECT()
	se.Get(kvs.Key{Kind: "env", Name: "tailFilename"}).Return("agent_test.go", nil).AnyTimes()

	a, err := NewAgent(s, []string{"tail", "-n", "5", "-f", "${env:// .tailFilename }"}, []string{}, DefaultRetry, DefaultPatchTable())
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
}
