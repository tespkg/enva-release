package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"meera.tech/envs/pkg/store"
	"meera.tech/kit/templates"
)

var (
	envKeyword        = "env"
	envArgSchema      = envKeyword + "://"
	envInspectPrefix  = strings.ToUpper(envKeyword)
	tplLeftDelimiter  = `%%`
	tplRightDelimiter = `%%`
)

var (
	// E.g, env://example
	envArgsRegex = regexp.MustCompile(fmt.Sprintf(`%s[a-z]*`, envArgSchema))
	// E.g, %% .ENV_project_env %%
	envInspectFilesRegex = regexp.MustCompile(fmt.Sprintf(`%s \.(%s)_([a-zA-Z].*)_([a-zA-Z].*) %s`, tplLeftDelimiter, envInspectPrefix, tplRightDelimiter))
)

// replaceable set of functions for fault injection
type patchTable struct {
	create func(name string) (*os.File, error)
}

type agent struct {
	s               store.Store
	args            []string
	absInspectFiles []string
	relInspectFiles []string

	// Values populated with the env store.
	finalisedArgs []string
	finalisedVars map[string]string

	// replaceable set of functions for fault injection
	p patchTable

	tplLeftDelim  string
	tplRightDelim string
}

func (a *agent) populateEnvVars() error {
	// Populate env var for args.
	var finalisedArgs []string
	for _, arg := range a.args {
		bNewArg := envArgsRegex.ReplaceAllFunc([]byte(arg), func(bytes []byte) []byte {
			rawArg := string(bytes)
			rawKey := rawArg[len(envArgSchema)-1:]
			val, err := a.s.Get(store.Key{
				Name: rawKey,
			})
			if err != nil {
				return bytes
			}
			return []byte(val.(string))
		})

		newArg := string(bNewArg)
		if strings.Contains(newArg, envArgSchema) {
			return fmt.Errorf("unable to get env var for: %v", arg)
		}
		finalisedArgs = append(finalisedArgs, newArg)
	}
	a.finalisedArgs = finalisedArgs

	// Populate inspect files
	// Get env var from abs path files
	var err error
	vars := make(map[string]string)
	allFiles := make(map[string]struct{})
	for _, fn := range a.absInspectFiles {
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		vars, err = populatedVars(a.s, f, vars)
		if err != nil {
			f.Close()
			return fmt.Errorf("populated vars failed fn: %v err: %v", fn, err)
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
			if !info.IsDir() {
				for _, fn := range a.relInspectFiles {
					if templates.Match(fn, path) {
						f, err := os.Open(path)
						if err != nil {
							return err
						}
						vars, err = populatedVars(a.s, f, vars)
						if err != nil {
							f.Close()
							return fmt.Errorf("populated vars failed fn: %v err: %v", path, err)
						}
						f.Close()
						allFiles[path] = struct{}{}
					}
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Render files with populated vars
	for fn := range allFiles {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}
		f, err := a.p.create(fn)
		if err != nil {
			return err
		}

		tpl := template.New(fn)
		tpl, err = tpl.Delims(a.tplLeftDelim, a.tplRightDelim).Parse(string(b))
		if err != nil {
			return err
		}
		if err := tpl.Execute(f, vars); err != nil {
			return err
		}
	}
	a.finalisedVars = vars

	return nil
}

func populatedVars(s store.Store, rd io.Reader, vars map[string]string) (map[string]string, error) {
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
