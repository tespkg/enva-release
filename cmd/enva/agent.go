package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/time/rate"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/kit/log"
)

const (
	gracefullyTerminateTimeout = time.Second * 5
)

type createFunc func(name string) (*os.File, error)
type templateDirFunc func() string

// Replaceable set of functions for test & fault injection
type patchTable struct {
	create createFunc
	tplDir templateDirFunc
}

func defaultPatchTable() patchTable {
	return patchTable{
		create: os.Create,
		tplDir: func() string {
			dir, err := os.UserConfigDir()
			if err != nil {
				dir = "/tmp"
			}
			return dir
		},
	}
}

// retry configuration for the Process
type retry struct {
	// restart is the timestamp of the next scheduled restart attempt
	restart *time.Time

	// number of times to attempts left to retry applying the latest desired configuration
	budget int

	// maxRetries is the maximum number of retries
	maxRetries int

	// initialInterval is the delay between the first restart, from then on it is
	// multiplied by a factor of 2 for each subsequent retry
	initialInterval time.Duration
}

var (
	// defaultRetry configuration for Procs
	defaultRetry = retry{
		maxRetries:      10,
		initialInterval: 200 * time.Millisecond,
	}
)

type config struct {
	args      []string
	osEnvVars []string
}

type agent struct {
	// KV store
	kvs kvs.KVStore

	// Original args for the Proc
	rawArgs []string

	// Original os envs for the Proc
	rawOSEnvs []string

	// Template files
	osEnvTplFiles []string

	// Replaceable set of functions for test & fault injection
	pt patchTable

	// retry configuration
	retry retry

	// desired configuration state
	desiredConfig config

	// current configuration is the highest epoch configuration
	currentConfig config

	// channel for posting desired configurations
	configCh chan config

	// Channel for process exit notifications
	statusCh chan error

	// channel for terminate running process
	terminateCh chan struct{}

	// channel for abort running process
	abortCh chan struct{}
}

func newAgent(kvs kvs.KVStore, args, osEnvFiles []string, retry retry, pt patchTable) (*agent, error) {
	// Template osEnvFiles if this is any and use it later.
	osEnvTplFiles, err := templateOSEnvFiles(osEnvFiles, pt.tplDir())
	if err != nil {
		return nil, err
	}

	return &agent{
		kvs:           kvs,
		rawArgs:       args,
		rawOSEnvs:     os.Environ(),
		pt:            pt,
		osEnvTplFiles: osEnvTplFiles,
		retry:         retry,
		configCh:      make(chan config),
		statusCh:      make(chan error),
		terminateCh:   make(chan struct{}, 1),
		abortCh:       make(chan struct{}, 1),
	}, nil
}

func (a *agent) run(ctx context.Context) error {
	log.Infoa("Starting ", a.rawArgs)

	rateLimiter := rate.NewLimiter(1, 10)
	var reconcileTimer *time.Timer

	for {
		err := rateLimiter.Wait(ctx)
		if err != nil {
			a.terminate()
			return err
		}

		// Maximum duration or duration till next restart
		var delay time.Duration = 1<<63 - 1
		if a.retry.restart != nil {
			delay = time.Until(*a.retry.restart)
		}
		if reconcileTimer != nil {
			reconcileTimer.Stop()
		}
		reconcileTimer = time.NewTimer(delay)

		select {
		case config := <-a.configCh:
			if !reflect.DeepEqual(a.desiredConfig, config) {
				log.Infof("Received new config, resetting budget")
				a.desiredConfig = config

				// reset retry budget if and only if the desired config changes
				a.retry.budget = a.retry.maxRetries
				a.reconcile()
			}

		case statusErr := <-a.statusCh:
			if statusErr == nil {
				log.Infoa("enva has finished ", a.currentConfig.args)
				return nil
			}
			// Schedule a retry for an error.
			// skip retrying twice by checking retry restart delay
			if a.retry.restart == nil {
				if a.retry.budget > 0 {
					delayDuration := a.retry.initialInterval * (1 << uint(a.retry.maxRetries-a.retry.budget))
					restart := time.Now().Add(delayDuration)
					a.retry.restart = &restart
					a.retry.budget--
					log.Infof("Set retry delay to %v, budget to %d, caused by: %v", delayDuration, a.retry.budget, statusErr)
				} else {
					return fmt.Errorf("permanent error: budget exhausted trying to fulfill the desired configuration, latest status: %v", statusErr)
				}
			} else {
				log.Debugf("Restart already scheduled")
			}

		case <-reconcileTimer.C:
			a.reconcile()

		case <-ctx.Done():
			a.terminate()
			log.Infoa("enva has successfully terminated")
			return nil
		}
	}
}

// TODO: watch env store for key changes...
func (a *agent) watch(ctx context.Context) error {
	_ = ctx

	renderTimer := time.NewTimer(time.Millisecond * 100)
	for {
		select {
		case <-renderTimer.C:
			// Render args, osEnvs to new config
			c, err := render(a.kvs, a.rawArgs, a.rawOSEnvs, a.osEnvTplFiles, a.pt)
			if err != nil {
				return err
			}
			// Notify new desiredConfig
			a.configCh <- c

		case <-ctx.Done():
			return nil
		}
	}
}

func (a *agent) reconcile() {
	log.Infof("Reconciling retry (budget %d)", a.retry.budget)

	// check that the config is current
	if reflect.DeepEqual(a.desiredConfig, a.currentConfig) {
		log.Infof("Reapplying same desired & current configuration")
	}

	// cancel any scheduled restart
	a.retry.restart = nil

	a.currentConfig = a.desiredConfig

	go a.runWait(a.desiredConfig.args, a.desiredConfig.osEnvVars, a.terminateCh, a.abortCh)
}

func (a *agent) terminate() {
	a.terminateCh <- struct{}{}
	log.Infof("Graceful termination period is %v, starting...", gracefullyTerminateTimeout)
	time.Sleep(gracefullyTerminateTimeout)
	log.Infof("Graceful termination period complete, terminating remaining process.")
	a.abortCh <- struct{}{}
}

func (a *agent) runWait(nameArgs, osEnvs []string, terminate, abort <-chan struct{}) {
	err := runProc(nameArgs, osEnvs, terminate, abort)
	a.statusCh <- err
}

func runProc(nameArgs, osEnvs []string, terminate, abort <-chan struct{}) error {
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
	cmd.Env = osEnvs
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-abort:
		log.Warnf("Aborting %v", name)
		return cmd.Process.Signal(syscall.SIGKILL)
	case <-terminate:
		log.Infof("Gracefully terminate %v", name)
		return cmd.Process.Signal(syscall.SIGTERM)
	case err := <-done:
		return err
	}
}

func render(kvStore kvs.KVStore, rawArgs, rawOSEnvs, osEnvTplFiles []string, pt patchTable) (config, error) {
	// Render os.env
	osEnvs := make([]string, len(rawOSEnvs))
	osEnvsVars := make(map[string]string)
	for i, osEnv := range rawOSEnvs {
		out := bytes.Buffer{}
		err := kvs.Render(kvStore, bytes.NewBufferString(osEnv), &out)
		if err != nil {
			return config{}, err
		}
		osEnvs[i] = out.String()
		ii := strings.Split(out.String(), "=")
		osEnvsVars[ii[0]] = ii[1]
	}

	// Render OSEnvFiles from the saved template osEnvFiles by using os.env vars
	if err := renderOSEnvFiles(osEnvTplFiles, osEnvsVars, pt); err != nil {
		return config{}, err
	}

	// Render args
	argsStr := strings.Join(rawArgs, "#")
	out := bytes.Buffer{}
	err := kvs.Render(kvStore, bytes.NewBufferString(argsStr), &out)
	if err != nil {
		return config{}, err
	}
	finalisedArgs := strings.Split(out.String(), "#")

	return config{
		args:      finalisedArgs,
		osEnvVars: osEnvs,
	}, nil
}

func renderOSEnvFiles(osTplFiles []string, vars map[string]string, pt patchTable) error {
	for _, fn := range osTplFiles {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}

		// Trim tplDir and create new file
		dstFn := fn
		dirPrefix := pt.tplDir()
		if len(dirPrefix) > 0 {
			dstFn = fn[len(dirPrefix):]
		}
		f, err := pt.create(dstFn)
		if err != nil {
			return err
		}

		tpl := template.New(dstFn)
		tpl, err = tpl.Parse(string(b))
		if err != nil {
			return err
		}
		if err := tpl.Execute(f, vars); err != nil {
			return err
		}
	}
	return nil
}

// templateOSEnvFiles Copy all the os env files to template dir, use them as the templates files
func templateOSEnvFiles(files []string, tplDir string) ([]string, error) {
	var templateFiles []string
	if tplDir == "" {
		for _, fn := range files {
			templateFiles = append(templateFiles, fn)
		}
		return templateFiles, nil
	}

	var opendFds []io.Closer
	defer func() {
		for _, closer := range opendFds {
			closer.Close()
		}
	}()

	for _, fn := range files {
		tplFn := filepath.Join(tplDir, fn)
		if err := os.MkdirAll(filepath.Dir(tplFn), 0755); err != nil {
			return nil, fmt.Errorf("mkdir for tpl file: %v failed: %w", tplFn, err)
		}
		input, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		opendFds = append(opendFds, input)

		output, err := os.Create(tplFn)
		if err != nil {
			return nil, err
		}
		opendFds = append(opendFds, output)

		if _, err := io.Copy(output, input); err != nil {
			return nil, err
		}

		templateFiles = append(templateFiles, tplFn)
	}

	return templateFiles, nil
}
