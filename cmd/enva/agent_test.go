package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"tespkg.in/envs/pkg/kvs"
)

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

func TestAgentRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	s := kvs.NewMockKVStore(mockCtrl)

	se := s.EXPECT()
	se.Get(kvs.Key{Kind: "env", Name: "tailFilename"}).Return("agent_test.go", nil).AnyTimes()

	a, err := newAgent(s, []string{"tail", "-f", "${env:// .tailFilename }"})
	require.Nil(t, err)

	timer := time.NewTicker(time.Second * 2)
	go func() {
		<-timer.C
		a.terminateCh <- errors.New("test term")
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancelTimer := time.NewTicker(time.Second * 3)
	go func() {
		<-cancelTimer.C
		cancel()
	}()

	a.run(ctx)
}
