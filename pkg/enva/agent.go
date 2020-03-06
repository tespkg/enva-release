package enva

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
	"sync"
	"sync/atomic"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/time/rate"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/kit/log"
)

const (
	pollingWatchInterval       = time.Second * 5
	gracefullyTerminateTimeout = time.Second * 5
	hold                       = -1
	success                    = 0
	configUpdated              = 1
)

type createFunc func(name string) (*os.File, error)
type templateDirFunc func() string

// Replaceable set of functions for test & fault injection
type PatchTable struct {
	create createFunc
	tplDir templateDirFunc
}

func DefaultPatchTable() PatchTable {
	return PatchTable{
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
	// DefaultRetry configuration for Procs
	DefaultRetry = retry{
		maxRetries:      10,
		initialInterval: 200 * time.Millisecond,
	}

	// used for the none-daemon Proc,
	// so that enva will be able to exist gracefully even there is no statusCh event received.
	isProcRunning int64
)

type exitStatus struct {
	code int
	err  error
}

func newExitStatus(code int, err error) exitStatus {
	return exitStatus{
		code: code,
		err:  err,
	}
}

type config struct {
	args      []string
	osEnvVars []string
}

func isEnvfArg(arg string) bool {
	return strings.Contains(arg, "envf-")
}

func isConfigDeepEqual(a, b config) (isOSEnvsEq, isArgsEq bool) {
	if reflect.DeepEqual(a.osEnvVars, b.osEnvVars) {
		isOSEnvsEq = true
	}

	if len(a.args) != len(b.args) {
		return isOSEnvsEq, false
	}
	for i := 0; i < len(a.args); i++ {
		aArg := a.args[i]
		bArg := b.args[i]

		isAArgFile, isBArgFile := isEnvfArg(aArg), isEnvfArg(bArg)

		// If both are envf file
		if isAArgFile && isBArgFile {
			aDoc, _ := ioutil.ReadFile(aArg)
			bDoc, _ := ioutil.ReadFile(bArg)
			if !reflect.DeepEqual(aDoc, bDoc) {
				return isOSEnvsEq, false
			}
			continue
		}

		// Not envf file
		if aArg != bArg {
			return isOSEnvsEq, false
		}
	}

	return isOSEnvsEq, true
}

func removeEnvfFile(c config) {
	for _, arg := range c.args {
		if !isEnvfArg(arg) {
			continue
		}
		if err := os.RemoveAll(arg); err != nil {
			log.Warnf("Remove %v failed: %v", arg, err)
		}
	}
}

type Agent struct {
	// KV store
	kvs kvs.KVStore

	// Original args for the Proc
	rawArgs []string

	// Original os envs for the Proc
	rawOSEnvs []string

	// Template files
	osEnvTplFiles []string

	// Replaceable set of functions for test & fault injection
	pt PatchTable

	// retry configuration
	retry retry

	// desired configuration state
	desiredConfig config

	// current configuration is the highest epoch configuration
	currentConfig config

	// channel for posting desired configurations
	configCh chan config

	// Channel for process exit notifications
	statusCh chan exitStatus

	// channel for terminate running process
	terminateCh chan exitStatus
}

func NewAgent(kvs kvs.KVStore, args, osEnvFiles []string, retry retry, pt PatchTable) (*Agent, error) {
	// Template osEnvFiles if this is any and use it later.
	osEnvTplFiles, err := templateOSEnvFiles(osEnvFiles, pt.tplDir())
	if err != nil {
		return nil, err
	}

	return &Agent{
		kvs:           kvs,
		rawArgs:       args,
		rawOSEnvs:     os.Environ(),
		pt:            pt,
		osEnvTplFiles: osEnvTplFiles,
		retry:         retry,
		configCh:      make(chan config),
		statusCh:      make(chan exitStatus),
	}, nil
}

func (a *Agent) Run(ctx context.Context) error {
	log.Infoa("Starting ", a.rawArgs)

	rateLimiter := rate.NewLimiter(1, 10)
	var reconcileTimer *time.Timer
	once := &sync.Once{}

	for {
		err := rateLimiter.Wait(context.Background())
		if err != nil {
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
			// There are two kinds of config update
			// The first one is the values in OS env vars got changed, it will not trigger the Proc restart flow in our current use case,
			// The second one is the values in Proc's args/options got changed, the Proc restart flow should been triggered.
			isOSEnvsEq, isArgsEq := isConfigDeepEqual(a.currentConfig, config)
			if !isOSEnvsEq {
				log.Infof("Received new os envs, re-rendering them")
				log.Debuga("current os envs ", a.currentConfig.osEnvVars)
				log.Debuga("new os envs ", config.osEnvVars)
				osEnvsVars := make(map[string]string)
				for _, osEnv := range config.osEnvVars {
					ii := strings.Split(osEnv, "=")
					osEnvsVars[ii[0]] = ii[1]
				}
				// Render OSEnvFiles from the saved template osEnvFiles by using new os.env vars
				if err := renderOSEnvFiles(a.osEnvTplFiles, osEnvsVars, a.pt); err != nil {
					log.Warna("render os env file failed ", err)
				}

				// Since there might have no restart required, set current osEnvVars directly
				a.currentConfig.osEnvVars = config.osEnvVars
			}

			if !isArgsEq {
				log.Infof("Received new config, resetting budget")
				a.desiredConfig = config

				// Reset retry budget if and only if the desired config changes
				a.retry.budget = a.retry.maxRetries

				// For most of the daemon service which will listen & serve on a TCP port,
				// There is a race condition, which is when the first new desired Proc start occurred before the previous Proc stopped take placed,
				// might cause the restart operation failed and starting to retry start
				log.Debugf("triggering the termination caused by config updated")
				a.terminate(newExitStatus(configUpdated, nil))
				a.reconcile()
			} else {
				// Remove useless generated envf file
				removeEnvfFile(config)
				continue
			}

		case status := <-a.statusCh:
			log.Debugf("status changed, code: %v, err: %v", status.code, status.err)
			if status.code == success {
				log.Infoa("enva has finished ", a.currentConfig.args)
				return nil
			}
			if status.code == configUpdated {
				continue
			}
			if status.err != nil {
				log.Warnf("%v got unexpected err %v", a.currentConfig.args, status.err)
			}

			// Schedule a retry for an error.
			// skip retrying twice by checking retry restart delay
			if a.retry.restart == nil {
				if a.retry.budget > 0 {
					delayDuration := a.retry.initialInterval * (1 << uint(a.retry.maxRetries-a.retry.budget))
					restart := time.Now().Add(delayDuration)
					a.retry.restart = &restart
					a.retry.budget--
					log.Infof("Set retry delay to %v, budget to %d", delayDuration, a.retry.budget)
				} else {
					return fmt.Errorf("permanent error: budget exhausted trying to fulfill the desired configuration")
				}
			} else {
				log.Debugf("Restart already scheduled")
			}

		case <-reconcileTimer.C:
			a.reconcile()

		case <-ctx.Done():
			// It might have multiple notifications, if ctx canceled and we didn't return immediately.
			// introduce the once to make sure only trigger terminate operation once.
			once.Do(func() {
				a.terminate(newExitStatus(success, nil))
			})

			// For none-daemon Proc there might no status event from status channel,
			// Check isProcRunning here and see if need to exit from here.
			if atomic.LoadInt64(&isProcRunning) == 0 {
				log.Debugf("ctx done exit")
				return nil
			}
		}
	}
}

func (a *Agent) Watch(ctx context.Context) error {
	renderTimer := time.NewTimer(time.Millisecond * 100)
	for {
		select {
		case <-renderTimer.C:
			// Render args, osEnvs to new config
			c, err := render(a.kvs, a.rawArgs, a.rawOSEnvs)
			if err != nil && !errors.Is(err, kvs.ErrNotFound) {
				return err
			}
			if errors.Is(err, kvs.ErrNotFound) {
				log.Infoa(err)
			}
			if err == nil {
				// Notify config periodically
				a.configCh <- c
			}

			renderTimer.Stop()
			// Polling might is not good enough,
			// Assume there are 500 Proc(include replica) under the control of enva in total
			// And each Proc have about 10 keys need to be rendered from envs
			// If polling interval is 5 seconds, the readonly qps to envs would be 500 * 10 / 5 = 1000 in average, which is acceptable
			// We might need to introduce stream watching later.
			renderTimer = time.NewTimer(pollingWatchInterval)

		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Agent) reconcile() {
	log.Infof("Reconciling retry (budget %d)", a.retry.budget)

	// check that the config is current
	if isEnvsEq, isArgsEq := isConfigDeepEqual(a.desiredConfig, a.currentConfig); isEnvsEq && isArgsEq {
		log.Infof("Reapplying same desired & current configuration")
	}

	// cancel any scheduled restart
	a.retry.restart = nil

	a.terminateCh = make(chan exitStatus, 1)

	a.currentConfig = a.desiredConfig
	go a.runWait(a.desiredConfig.args, a.desiredConfig.osEnvVars, a.terminateCh)
}

func (a *Agent) terminate(status exitStatus) {
	if a.terminateCh == nil {
		return
	}
	log.Debuga("sending terminate signal ", status)
	a.terminateCh <- status
}

func (a *Agent) runWait(nameArgs, osEnvs []string, terminate chan exitStatus) {
	extStatus := runProc(nameArgs, osEnvs, terminate)
	a.statusCh <- extStatus
}

func runProc(nameArgs, osEnvs []string, terminate chan exitStatus) exitStatus {
	if len(nameArgs) == 0 {
		return newExitStatus(-1, errors.New("require at least proc name"))
	}

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
		return newExitStatus(-1, errors.New("require at least proc name"))
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	log.Infoa("Running ", cmd.Process.Pid, nameArgs)
	atomic.StoreInt64(&isProcRunning, 1)

	var err error
	var ok bool
	abort := make(chan struct{})
	status := newExitStatus(hold, nil)
	once := sync.Once{}
	for {
		select {
		case <-abort:
			log.Infoa("Aborting ", cmd.Process.Pid, name)
			cmd.Process.Signal(syscall.SIGKILL)
		case status, ok = <-terminate:
			if !ok {
				// Channel got closed, but ignore here.
			}
			once.Do(func() {
				log.Infoa("Gracefully terminate ", cmd.Process.Pid, nameArgs, status)
				cmd.Process.Signal(syscall.SIGTERM)
				go func() {
					time.Sleep(gracefullyTerminateTimeout)
					abort <- struct{}{}
				}()
			})
		case err = <-done:
			log.Infof("%v %v done", cmd.Process.Pid, name)
			status.err = err
			goto end
		}
	}

end:
	atomic.StoreInt64(&isProcRunning, 0)
	return status
}

func render(kvStore kvs.KVStore, rawArgs, rawOSEnvs []string) (config, error) {
	// Render os.env
	osEnvs := make([]string, len(rawOSEnvs))
	for i, osEnv := range rawOSEnvs {
		out := bytes.Buffer{}
		err := kvs.Render(kvStore, bytes.NewBufferString(osEnv), &out)
		if err != nil {
			return config{}, err
		}
		newOSEnv := out.String()
		if osEnv != newOSEnv {
			log.Debugf("render %v to %v", osEnv, newOSEnv)
		}
		osEnvs[i] = newOSEnv
	}

	// Render args
	argsStr := strings.Join(rawArgs, "#")
	log.Debuga("argsStr ", argsStr)
	out := bytes.Buffer{}
	err := kvs.Render(kvStore, bytes.NewBufferString(argsStr), &out)
	if err != nil {
		return config{}, err
	}
	finalisedArgs := strings.Split(out.String(), "#")
	log.Debuga("finalisedArgs ", finalisedArgs, " len ", len(finalisedArgs))

	return config{
		args:      finalisedArgs,
		osEnvVars: osEnvs,
	}, nil
}

func renderOSEnvFiles(osTplFiles []string, vars map[string]string, pt PatchTable) error {
	var fds []io.Closer
	defer func() {
		for _, fd := range fds {
			fd.Close()
		}
	}()
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
		log.Debugf("render os env vars from %v to %v", fn, dstFn)
		f, err := pt.create(dstFn)
		if err != nil {
			return err
		}
		fds = append(fds, f)

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
