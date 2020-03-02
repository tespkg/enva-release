package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/kit/log"
)

const (
	gracefullyTerminateTimeout = time.Second * 10
)

type agent struct {
	kvs  kvs.KVStore
	args []string

	// Channel for process exit notifications
	statusCh    chan error
	terminateCh chan error
}

func newAgent(kvs kvs.KVStore, args []string) (*agent, error) {
	a := &agent{
		kvs:         kvs,
		args:        args,
		statusCh:    make(chan error),
		terminateCh: make(chan error),
	}

	return a, nil
}

func (a *agent) run(ctx context.Context) {
	log.Infoa("Starting ", a.args)

	err := a.reconcile()
	if err != nil {
		log.Errora(err)
		return
	}

	for {
		select {
		case status := <-a.statusCh:
			log.Infof("Restarting %v, caused by %v", a.args[0], status)
			err = a.reconcile()
			if err != nil {
				log.Errorf("Restart %v failed: %v", a.args[0], err)
				return
			}
		case <-ctx.Done():
			a.terminate(nil)
			log.Infoa("enva has successfully terminated")
			return
		}
	}
}

// TODO: watch env store for key changes...
func (a *agent) watch(ctx context.Context) {
	_ = ctx
}

func (a *agent) reconcile() error {
	rawArgs := strings.Join(a.args, " ")
	out := bytes.Buffer{}
	err := kvs.Render(a.kvs, bytes.NewBufferString(rawArgs), &out)
	if err != nil {
		return err
	}
	finalisedArgs := strings.Split(out.String(), " ")
	go a.runWait(finalisedArgs, a.terminateCh)

	return nil
}

func (a *agent) terminate(err error) {
	a.terminateCh <- err
	<-a.statusCh
}

func (a *agent) runWait(nameArgs []string, terminate <-chan error) {
	err := runProc(nameArgs, terminate)
	a.statusCh <- err
}

func runProc(nameArgs []string, terminate <-chan error) error {
	if len(nameArgs) == 0 {
		return errors.New("require at least proc name")
	}
	log.Infoa("Running ", nameArgs)

	name := nameArgs[0]
	var args []string
	if len(nameArgs) > 1 {
		args = nameArgs[1:]
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	abort := make(chan error)
	select {
	case err := <-abort:
		log.Warnf("Aborting %v", name)
		if errKill := cmd.Process.Kill(); errKill != nil {
			log.Warnf("killing %v caused an error %v", name, errKill)
		}
		return err
	case err := <-terminate:
		log.Infof("Gracefully terminate %v", name)
		tick := time.NewTicker(gracefullyTerminateTimeout)
		go func() {
			<-tick.C
			abort <- errors.New("gracefully terminate timeout")
		}()
		if errTerm := cmd.Process.Signal(syscall.SIGTERM); errTerm != nil {
			log.Warnf("terminating %v caused an error %v", name, errTerm)
		}
		return err
	case err := <-done:
		return err
	}
}
