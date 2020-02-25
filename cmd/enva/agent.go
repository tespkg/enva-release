package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"text/template"
	"time"

	"tespkg.in/envs/pkg/store"
	"tespkg.in/kit/log"
	"tespkg.in/kit/templates"
)

const (
	envKeyword        = "env"
	envfKeyword       = "envf"
	envArgSchema      = envKeyword + "://"
	envfArgSchema     = envfKeyword + "://"
	tplLeftDelimiter  = `%%`
	tplRightDelimiter = `%%`

	gracefullyTerminateTimeout = time.Second * 10
)

var (
	// E.g, env://example
	envArgsRegex = regexp.MustCompile(fmt.Sprintf(`\{(%s|%s)[\-A-Za-z0-9/]*\}`, envArgSchema, envfArgSchema))
	// E.g, %% .ENV_project_env %%
	envInspectFilesRegex = regexp.MustCompile(fmt.Sprintf(`%s \.(%s)_([\-A-Za-z0-9].*)_([\-A-Za-z0-9].*) %s`,
		tplLeftDelimiter, strings.ToUpper(envKeyword), tplRightDelimiter))
)

type values map[string]string

// newValues allocate a empty values
func newValues() values {
	return make(map[string]string)
}

func mergeValues(vals ...values) values {
	mergedVal := newValues()
	for _, val := range vals {
		for k, v := range val {
			mergedVal[k] = v
		}
	}
	return mergedVal
}

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

type agent struct {
	s               store.Store
	args            []string
	absInspectFiles []string
	relInspectFiles []string

	// Replaceable set of functions for test & fault injection
	p patchTable

	// Delimiters of template var indicator in files need to inspect
	tplLeftDelim  string
	tplRightDelim string

	// Channel for vars modification notifications
	varsCh chan values
	// Channel for process exit notifications
	statusCh    chan error
	terminateCh chan error

	// Template files
	tplFiles []string
	// Env values
	argVars values
	fsVars  values
	vars    values
}

func newAgent(s store.Store, args, absFiles, relFiles []string, pt patchTable) (*agent, error) {
	a := &agent{
		s:               s,
		args:            args,
		absInspectFiles: absFiles,
		relInspectFiles: relFiles,
		p:               pt,
		tplLeftDelim:    tplLeftDelimiter,
		tplRightDelim:   tplRightDelimiter,
		varsCh:          make(chan values),
		statusCh:        make(chan error),
		terminateCh:     make(chan error),
	}

	var err error
	// Parse args, get vars name & value from args
	a.argVars, err = parseArgs(a.s, a.args)
	if err != nil {
		return nil, fmt.Errorf("initialize args failed: %v", err)
	}

	// Parse inspected files, get vars name & value from inspected files and template files list
	a.fsVars, a.tplFiles, err = parseInspectedFiles(a.s, a.absInspectFiles, a.relInspectFiles, a.p.tplDir)
	if err != nil {
		return nil, fmt.Errorf("populate inspected files failed: %v", err)
	}
	a.vars = mergeValues(a.argVars, a.fsVars)

	return a, nil
}

func (a *agent) populateProcEnvVars(vars values) ([]string, error) {
	// Populate env var for args.
	finalisedArgs, err := populateArgs(vars, a.args)
	if err != nil {
		return nil, fmt.Errorf("populate args failed: %v", err)
	}

	// Populate inspect files
	err = populateInspectedFiles(vars, a.tplFiles, a.tplLeftDelim, a.tplRightDelim, a.p)
	if err != nil {
		return nil, fmt.Errorf("populate inspected files failed: %v", err)
	}

	return finalisedArgs, nil
}

func (a *agent) run(ctx context.Context) {
	log.Infoa("Starting ", a.args)

	err := a.reconcile(a.vars)
	if err != nil {
		log.Errora(err)
		return
	}

	for {
		select {
		case vars := <-a.varsCh:
			log.Infof("Restarting %v, caused by env vars changing", a.args[0])
			a.terminate(nil)
			err = a.reconcile(vars)
			if err != nil {
				log.Errora(err)
				return
			}
		case status := <-a.statusCh:
			log.Infof("Restarting %v, caused by %v", a.args[0], status)
			err = a.reconcile(nil)
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

func (a *agent) reconcile(vars values) error {
	if vars == nil {
		vars = a.vars
	}
	finalisedArgs, err := a.populateProcEnvVars(vars)
	if err != nil {
		return err
	}
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

func parseArgs(s store.Store, args []string) (values, error) {
	vars := newValues()
	for _, arg := range args {
		parts := envArgsRegex.FindAll([]byte(arg), -1)
		for _, part := range parts {
			ele := string(part)
			rawKey := ele[len(envArgSchema):]
			val, err := s.Get(store.Key{
				Name: rawKey,
			})
			if err != nil {
				return nil, err
			}
			vars[rawKey] = val.(string)
		}
	}

	return vars, nil
}

func parseInspectedFiles(s store.Store, absFiles, relFiles []string, tplDir templateDirFunc) (vars values, tplFiles []string, err error) {
	vars = newValues()
	// Get env var from abs path files
	allFiles := make(map[string]struct{})
	for _, fn := range absFiles {
		if _, ok := allFiles[fn]; ok {
			continue
		}

		f, err := os.Open(fn)
		if err != nil {
			return nil, nil, err
		}
		vars, err = fetchVars(s, f, vars)
		if err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("populated vars failed fn: %v err: %v", fn, err)
		}
		f.Close()
		allFiles[fn] = struct{}{}
	}
	// Get env var from rel path files
	wd, _ := os.Getwd()
	err = filepath.Walk(wd,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !matchWildcardsRelFiles(relFiles, path) {
				return nil
			}

			if _, ok := allFiles[path]; ok {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			vars, err = fetchVars(s, f, vars)
			if err != nil {
				f.Close()
				return fmt.Errorf("populated vars failed fn: %v err: %v", path, err)
			}
			f.Close()
			allFiles[path] = struct{}{}

			return nil
		})
	if err != nil {
		return nil, nil, err
	}

	tplFiles, err = backupTemplateFiles(allFiles, tplDir())
	if err != nil {
		return nil, nil, err
	}

	return
}

func populateArgs(vars values, args []string) ([]string, error) {
	var finalisedArgs []string
	for _, arg := range args {
		var rawKey string
		bNewArg := envArgsRegex.ReplaceAllFunc([]byte(arg), func(bytes []byte) []byte {
			rawArg := string(bytes)
			rawKey = rawArg[len(envArgSchema):]
			val, ok := vars[rawKey]
			if !ok {
				return bytes
			}
			return []byte(val)
		})

		newArg := string(bNewArg)
		if strings.Contains(newArg, envArgSchema) {
			return nil, fmt.Errorf("unable to get env var for: %v", arg)
		}
		finalisedArgs = append(finalisedArgs, newArg)
	}

	return finalisedArgs, nil
}

func populateInspectedFiles(vars values, tplFiles []string, tplLD, tplRD string, pt patchTable) error {
	for _, fn := range tplFiles {
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
		tpl, err = tpl.Delims(tplLD, tplRD).Parse(string(b))
		if err != nil {
			return err
		}
		if err := tpl.Execute(f, vars); err != nil {
			return err
		}
	}
	return nil
}

// backupTemplateFiles Copy all the inspected files to template dir, use them as the templates files
func backupTemplateFiles(files map[string]struct{}, tplDir string) ([]string, error) {
	var templateFiles []string
	if tplDir == "" {
		for fn := range files {
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

	for fn := range files {
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

func matchWildcardsRelFiles(wildcardPatterns []string, path string) bool {
	for _, pattern := range wildcardPatterns {
		if templates.Match(pattern, path) {
			return true
		}
	}
	return false
}

func fetchVars(s store.Store, rd io.Reader, vars values) (values, error) {
	b, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, fmt.Errorf("read inspect file failed: %v", err)
	}
	results := envInspectFilesRegex.FindAllStringSubmatch(string(b), -1)
	for _, res := range results {
		fKey := strings.Join([]string{res[1], res[2], res[3]}, "_")
		envKey := strings.Join([]string{res[2], res[3]}, "/")
		val, err := s.Get(store.Key{
			Name: envKey,
		})
		if err != nil {
			return nil, fmt.Errorf("%v not found", res[0])
		}
		vars[fKey] = val.(string)
	}
	return vars, nil
}
