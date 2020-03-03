package main

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
	terminate, abort := make(chan struct{}), make(chan struct{})

	timer := time.NewTicker(time.Second * 2)
	go func() {
		<-timer.C
		terminate <- struct{}{}
	}()

	err := runProc([]string{"tail", "-f", "agent_test.go"}, os.Environ(), terminate, abort)
	require.Nil(t, err)
}

func TestAgentRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	s := kvs.NewMockKVStore(mockCtrl)

	se := s.EXPECT()
	se.Get(kvs.Key{Kind: "env", Name: "tailFilename"}).Return("agent_test.go", nil).AnyTimes()

	a, err := newAgent(s, []string{"tail", "-n", "5", "-f", "${env:// .tailFilename }"}, []string{}, defaultRetry, defaultPatchTable())
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
		a.run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.watch(ctx)
	}()

	wg.Wait()
}
