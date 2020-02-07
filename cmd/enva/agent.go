package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"meera.tech/envs/pkg/store"
	"meera.tech/kit/templates"
)

const (
	envKeyword        = "env"
	envArgSchema      = envKeyword + "://"
	tplLeftDelimiter  = `%%`
	tplRightDelimiter = `%%`
)

var (
	// E.g, env://example
	envArgsRegex = regexp.MustCompile(fmt.Sprintf(`%s[a-z]*`, envArgSchema))
	// E.g, %% .ENV_project_env %%
	envInspectFilesRegex = regexp.MustCompile(fmt.Sprintf(`%s \.(%s)_([a-zA-Z].*)_([a-zA-Z].*) %s`,
		tplLeftDelimiter, strings.ToUpper(envKeyword), tplRightDelimiter))
)

type values map[string]string

// newValues allocate a empty values
func newValues() values {
	return make(map[string]string)
}

type createFunc func(name string) (*os.File, error)

// replaceable set of functions for fault injection
type patchTable struct {
	create createFunc
}

type agent struct {
	s               store.Store
	args            []string
	absInspectFiles []string
	relInspectFiles []string

	// Replaceable set of functions for fault injection
	p patchTable

	// Delimiters of template var indicator in files need to inspect
	tplLeftDelim  string
	tplRightDelim string

	// Values populated with the env store.
	finalisedArgs []string
	finalisedVars values
}

func (a *agent) populateEnvVars() error {
	var err error

	// Populate env var for args.
	a.finalisedArgs, err = populateArgs(a.s, a.args)
	if err != nil {
		return fmt.Errorf("populate args failed: %v", err)
	}

	// Populate inspect files
	a.finalisedVars, err = populateInspectedFiles(a.s, a.absInspectFiles, a.relInspectFiles, a.tplLeftDelim, a.tplRightDelim, a.p.create)
	if err != nil {
		return fmt.Errorf("populate inspected files failed: %v", err)
	}

	return nil
}

func populateArgs(s store.Store, args []string) ([]string, error) {
	var finalisedArgs []string
	for _, arg := range args {
		bNewArg := envArgsRegex.ReplaceAllFunc([]byte(arg), func(bytes []byte) []byte {
			rawArg := string(bytes)
			rawKey := rawArg[len(envArgSchema)-1:]
			val, err := s.Get(store.Key{
				Name: rawKey,
			})
			if err != nil {
				return bytes
			}
			return []byte(val.(string))
		})

		newArg := string(bNewArg)
		if strings.Contains(newArg, envArgSchema) {
			return nil, fmt.Errorf("unable to get env var for: %v", arg)
		}
		finalisedArgs = append(finalisedArgs, newArg)
	}

	return finalisedArgs, nil
}

func populateInspectedFiles(s store.Store, absFiles, relFiles []string, tplLD, tplRD string, create createFunc) (vars values, err error) {
	vars = newValues()

	// Get env var from abs path files
	allFiles := make(map[string]struct{})
	for _, fn := range absFiles {
		if _, ok := allFiles[fn]; ok {
			continue
		}

		f, err := os.Open(fn)
		if err != nil {
			return nil, err
		}
		vars, err = fetchVars(s, f, vars)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("populated vars failed fn: %v err: %v", fn, err)
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
		return nil, err
	}

	// Render files with populated vars
	for fn := range allFiles {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return nil, err
		}
		f, err := create(fn)
		if err != nil {
			return nil, err
		}

		tpl := template.New(fn)
		tpl, err = tpl.Delims(tplLD, tplRD).Parse(string(b))
		if err != nil {
			return nil, err
		}
		if err := tpl.Execute(f, vars); err != nil {
			return nil, err
		}
	}
	return
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
