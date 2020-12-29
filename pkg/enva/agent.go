package enva

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/time/rate"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/kit/log"
)

const (
	pollingWatchInterval       = time.Second * 10
	gracefullyTerminateTimeout = time.Second * 10
	hold                       = -1
	finished                   = 0
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
	procState int64
)

const (
	ProcStopped int64 = iota
	ProcStarting
	ProcRunning
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

type EnvFile struct {
	TemplateFilePath string
	Filename         string

	isChanged bool

	needRestart bool // TODO: config restart policy from args
}

type config struct {
	args     []string
	envFiles []EnvFile

	osEnvVars []string
}

func isEnvfArg(arg string) bool {
	return strings.Contains(arg, "envf-")
}

func isConfigDeepEqual(old, new config) (updatedEnvFiles []EnvFile, isOSEnvVarsEq, isArgsEq bool) {
	if reflect.DeepEqual(old.osEnvVars, new.osEnvVars) {
		isOSEnvVarsEq = true
	}

	for _, f := range new.envFiles {
		if f.isChanged {
			updatedEnvFiles = append(updatedEnvFiles, f)
		}
	}

	if len(old.args) != len(new.args) {
		return updatedEnvFiles, isOSEnvVarsEq, false
	}

	for i := 0; i < len(old.args); i++ {
		aArg := old.args[i]
		bArg := new.args[i]

		isAArgFile, isBArgFile := isEnvfArg(aArg), isEnvfArg(bArg)

		// If both are envf file
		if isAArgFile && isBArgFile {
			aDoc, _ := ioutil.ReadFile(aArg)
			bDoc, _ := ioutil.ReadFile(bArg)
			if !reflect.DeepEqual(aDoc, bDoc) {
				log.Infof("Un-equaled aArg: %v and bArg: %v", aArg, bArg)
				return updatedEnvFiles, isOSEnvVarsEq, false
			}
			continue
		}

		// Not envf file
		if aArg != bArg {
			log.Infof("Un-equaled aArg: %v and bArg: %v", aArg, bArg)
			return updatedEnvFiles, isOSEnvVarsEq, false
		}
	}

	return updatedEnvFiles, isOSEnvVarsEq, true
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

	// Original os env vars for the Proc
	rawOSEnvVars []string

	// Template files
	envTplFiles []EnvFile

	// Indicate if Proc will run only once or run as a daemon service
	isRunOnlyOnce bool

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

func NewAgent(kvs kvs.KVStore, args []string, envFiles []EnvFile, isRunOnlyOnce bool, retry retry, pt PatchTable) (*Agent, error) {
	envTplFiles, err := templateEnvFiles(envFiles, pt)
	if err != nil {
		return nil, err
	}

	return &Agent{
		kvs:           kvs,
		rawArgs:       args,
		rawOSEnvVars:  os.Environ(),
		pt:            pt,
		envTplFiles:   envTplFiles,
		isRunOnlyOnce: isRunOnlyOnce,
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
			// The first one is the env files specified by the enva in the hard way got changed, it might will trigger the Proc restart flow in our current use case,
			// The second one is the values in Proc's args/options got changed, the Proc restart flow should be triggered.
			needReconcile := false
			updatedEnvFiles, isOSEnvVarsEq, isArgsEq := isConfigDeepEqual(a.currentConfig, config)
			for _, envFile := range updatedEnvFiles {
				log.Infof("plain env file %v changed", envFile)
				if envFile.needRestart {
					needReconcile = true
				}
			}

			if !isOSEnvVarsEq {
				log.Infoa("os env vars changed")
				needReconcile = true
			}

			if !isArgsEq {
				log.Infoa("args changed")
				needReconcile = true
			}

			if !needReconcile {
				// Remove useless generated envf file
				removeEnvfFile(config)
				continue
			}

			log.Infoa("Received new config, resetting budget")
			a.desiredConfig = config

			// Reset retry budget if and only if the desired config changes
			a.retry.budget = a.retry.maxRetries

			// For most of the daemon service which will listen & serve on a TCP port,
			// There is a race condition, which is when the first new desired Proc start occurred before the previous Proc stopped take placed,
			// might cause the restart operation failed and starting to retry start
			log.Debugf("triggering the termination caused by config updated")
			a.terminate(newExitStatus(configUpdated, nil))
			a.reconcile()

		case status := <-a.statusCh:
			log.Debugf("status changed, code: %v, err: %v", status.code, status.err)
			if status.code == finished {
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
				a.terminate(newExitStatus(finished, nil))
			})

			// For none-daemon Proc there might no status event from status channel,
			// Check procState here and see if need to exit from here.
			if atomic.LoadInt64(&procState) == ProcStopped {
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
			c, err := render(a.kvs, a.rawArgs, a.rawOSEnvVars, a.envTplFiles, a.pt)
			if err != nil && !errors.Is(err, kvs.ErrNotFound) {
				log.Warna("Watch failed ", err)
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
	log.Infof("Reconciling budget %d", a.retry.budget)

	// check that the config is current
	if updatedEnvFiles, isOSEnvVarsEq, isArgsEq := isConfigDeepEqual(a.currentConfig, a.desiredConfig); len(updatedEnvFiles) == 0 && isArgsEq && isOSEnvVarsEq {
		log.Infof("Reapplying same desired & current configuration")
	}

	// cancel any scheduled restart
	a.retry.restart = nil

	a.terminateCh = make(chan exitStatus, 1)

	a.currentConfig = a.desiredConfig
	go a.runWait(a.desiredConfig.args, a.desiredConfig.osEnvVars, a.isRunOnlyOnce, a.terminateCh)
}

func (a *Agent) terminate(status exitStatus) {
	if a.terminateCh == nil {
		return
	}
	log.Debuga("sending terminate signal ", status)
	a.terminateCh <- status
}

func (a *Agent) runWait(nameArgs, osEnvs []string, isRunOnlyOnce bool, terminate chan exitStatus) {
	extStatus := runProc(nameArgs, osEnvs, isRunOnlyOnce, terminate)
	a.statusCh <- extStatus
}

func runProc(nameArgs, osEnvs []string, isRunOnlyOnce bool, terminate chan exitStatus) exitStatus {
	if len(nameArgs) == 0 {
		return newExitStatus(-1, errors.New("require at least proc name"))
	}
	atomic.StoreInt64(&procState, ProcStarting)
	name := nameArgs[0]
	var args []string
	if len(nameArgs) > 1 {
		args = nameArgs[1:]
	}
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = os.Stdin
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
	atomic.StoreInt64(&procState, ProcRunning)

	defaultExtStatusCode := hold
	if isRunOnlyOnce {
		defaultExtStatusCode = finished
	}
	status := newExitStatus(defaultExtStatusCode, nil)

	var err error
	var ok bool

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		log.Warnf("get pgid failed %v, fallback to use pid as pgid", err)
		pgid = cmd.Process.Pid
	}

	abort := make(chan struct{})
	once := sync.Once{}
	for {
		select {
		case <-abort:
			err = syscall.Kill(-pgid, syscall.SIGKILL)
			log.Infoa("Aborting ", cmd.Process.Pid, pgid, name, err)
		case status, ok = <-terminate:
			if !ok { // nolint
				// Channel got closed, but ignore here.
			}
			once.Do(func() {
				// TODO: fix zombie process later
				err = syscall.Kill(-pgid, syscall.SIGTERM)
				log.Infoa("Gracefully terminate ", cmd.Process.Pid, pgid, nameArgs, status, err)
				go func() {
					time.Sleep(gracefullyTerminateTimeout)
					abort <- struct{}{}
				}()
			})
		case err = <-done:
			log.Infof("%v(pgid: %v) %v done", cmd.Process.Pid, pgid, name)
			status.err = err
			goto end
		}
	}

end:
	atomic.StoreInt64(&procState, ProcStopped)
	return status
}

func render(kvStore kvs.KVStore, rawArgs, rawOSEnvVars []string, envTplFiles []EnvFile, pt PatchTable) (config, error) {
	// Render os.env
	osEnvVars := make([]string, len(rawOSEnvVars))
	for i, osEnv := range rawOSEnvVars {
		out := bytes.Buffer{}
		err := kvs.Render(kvStore, bytes.NewBufferString(osEnv), &out)
		if err != nil {
			return config{}, err
		}
		newOSEnvVar := out.String()
		if osEnv != newOSEnvVar {
			log.Debugf("render %v to %v", osEnv, newOSEnvVar)
		}
		osEnvVars[i] = newOSEnvVar
	}

	// Render env files
	envFiles, err := renderEnvFiles(kvStore, envTplFiles, pt)
	if err != nil {
		return config{}, err
	}

	// Render args
	argsStr := strings.Join(rawArgs, "#")
	log.Debuga("argsStr ", argsStr)
	out := bytes.Buffer{}
	err = kvs.Render(kvStore, bytes.NewBufferString(argsStr), &out)
	if err != nil {
		return config{}, err
	}
	finalisedArgs := strings.Split(out.String(), "#")
	log.Debuga("finalisedArgs ", finalisedArgs, " len ", len(finalisedArgs))

	return config{
		args:      finalisedArgs,
		envFiles:  envFiles,
		osEnvVars: osEnvVars,
	}, nil
}

func renderEnvFiles(kvStore kvs.KVStore, envTplFiles []EnvFile, pt PatchTable) ([]EnvFile, error) {
	var fds []io.Closer
	defer func() {
		for _, fd := range fds {
			fd.Close()
		}
	}()
	var envFiles []EnvFile
	for _, file := range envTplFiles {
		fn := file.TemplateFilePath
		input, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		fds = append(fds, input)

		tmpfile, err := ioutil.TempFile("", "plian-envf-")
		if err != nil {
			return nil, err
		}
		fds = append(fds, tmpfile)

		if err := kvs.Render(kvStore, input, tmpfile); err != nil {
			return nil, err
		}

		dstFn := file.Filename

		// check if the dst file same with the previous file or not
		isChanged := false
		aDoc, _ := ioutil.ReadFile(dstFn)
		tmpfile.Seek(0, io.SeekStart)
		bDoc, _ := ioutil.ReadAll(tmpfile)
		if !reflect.DeepEqual(aDoc, bDoc) {
			isChanged = true
		}

		log.Debugf("render env files from %v to %v", fn, dstFn)

		output, err := pt.create(dstFn)
		if err != nil {
			return nil, err
		}
		fds = append(fds, output)

		tmpfile.Seek(0, io.SeekStart)
		io.Copy(output, tmpfile)

		tmpfile.Close()
		os.Remove(tmpfile.Name())

		envFiles = append(envFiles, EnvFile{
			TemplateFilePath: file.TemplateFilePath,
			Filename:         dstFn,
			isChanged:        isChanged,
		})

	}
	return envFiles, nil
}

// templateEnvFiles Copy all the env files to template dir, use them as the templates files
func templateEnvFiles(files []EnvFile, pt PatchTable) ([]EnvFile, error) {
	tplDir := pt.tplDir()
	if tplDir == "" {
		return files, nil
	}

	var templateFiles []EnvFile
	var opendFds []io.Closer
	defer func() {
		for _, closer := range opendFds {
			closer.Close()
		}
	}()

	for _, file := range files {
		if file.TemplateFilePath != "" {
			templateFiles = append(templateFiles, file)
			continue
		}

		tplFn := filepath.Join(tplDir, file.Filename)
		if err := os.MkdirAll(filepath.Dir(tplFn), 0755); err != nil {
			return nil, fmt.Errorf("mkdir for tpl file: %v failed: %w", tplFn, err)
		}
		input, err := os.Open(file.Filename)
		if err != nil {
			return nil, err
		}
		opendFds = append(opendFds, input)

		output, err := pt.create(tplFn)
		if err != nil {
			return nil, err
		}
		opendFds = append(opendFds, output)

		if _, err := io.Copy(output, input); err != nil {
			return nil, err
		}

		templateFiles = append(templateFiles, EnvFile{
			TemplateFilePath: tplFn,
			Filename:         file.Filename,
		})
	}

	return templateFiles, nil
}

func (a *Agent) ServeStatusReport(ctx context.Context, endpoint string) error {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(endpointURL.Host, endpointURL.Port()))
	if err != nil {
		return err
	}

	http.HandleFunc(endpointURL.Path, func(w http.ResponseWriter, r *http.Request) {
		// starting.
		if atomic.LoadInt64(&procState) == ProcStarting {
			w.WriteHeader(http.StatusCreated)
			return
		}
		// stopped
		if atomic.LoadInt64(&procState) == ProcStopped {
			w.WriteHeader(http.StatusGone)
			return
		}
		// running.
		if atomic.LoadInt64(&procState) == ProcRunning {
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	httpServer := &http.Server{Addr: endpoint}
	go func() {
		httpServer.Serve(ln)
	}()

	<-ctx.Done()
	return httpServer.Shutdown(ctx)
}
