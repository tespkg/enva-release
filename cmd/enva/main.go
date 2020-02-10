package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/pborman/getopt/v2"
	"meera.tech/envs/pkg/store"
	"meera.tech/envs/pkg/store/consul"
	"meera.tech/envs/pkg/store/etcd"
	"meera.tech/kit/log"
)

var (
	envStoreDsn      = "http://localhost:8502"
	registerLocation bool
	registerKVs      []string
	inspectFiles     []string
	verbose          bool
	help             bool

	logOptions = log.DefaultOptions()
)

func init() {
	getopt.FlagLong(&envStoreDsn, "env-store-dsn", 'a', "Required, env store dsn")
	getopt.FlagLong(&registerLocation, "register-location", 'l', "Optional, register Proc location")
	getopt.FlagLong(&registerKVs, "register-kvs", 'k',
		`Optional, register k=v values to env store, e.g: register a=b and c=d by using "-k a=b -k c=d"`)
	getopt.FlagLong(&inspectFiles, "inspect-files", 'i',
		`Optional, inspect files(support "*" and "?" wildcards) and replace the env key with the value in env store, `+
			`if there is any key can't found in env store then prompt an error, `+
			`e.g: "-i *.yaml -i *.html" will inspect all the yaml & html files under the working dir.`)
	getopt.Flag(&verbose, 'v', "Optional, be verbose")
	getopt.FlagLong(&help, "help", 'h', "Optional, display usage")
	getopt.SetUsage(func() {
		s := getopt.CommandLine
		printUsage(s, os.Stderr)
	})
}

func printUsage(s *getopt.Set, w io.Writer) {
	parts := []string{
		"Usage:",
		s.Program(),
		"[Options]",
		"Proc",
		"[Proc options]",
		"{Proc args]",
	}
	fmt.Fprintln(w, strings.Join(parts, " "))
	s.PrintOptions(w)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `The supported key regex pattern in inspected files is: "%% \.ENV_([a-z].*)_([a-z].*) %%"
For example if there is a string "%% .ENV_Workspace_OauthID %%" in a file, 
then it will be replaced with value of key "Workspace/OauthID" in env store.`)
	fmt.Fprintln(w)
}

func verifyInspectFiles(inspectFiles []string) ([]string, []string, error) {
	var absFiles, relFiles []string
	for _, fn := range inspectFiles {
		if filepath.IsAbs(fn) {
			if strings.Contains(fn, "*") || strings.Contains(fn, "?") {
				return nil, nil, fmt.Errorf("wildcard * or ? not allowed in abs path file: %v", fn)
			}
			absFiles = append(absFiles, fn)
		} else {
			if strings.Contains(fn, "..") {
				return nil, nil, fmt.Errorf("unsupported inspect file path: %v, support either abs path or path under working dir", fn)
			}
			relFiles = append(relFiles, fn)
		}
	}
	return absFiles, relFiles, nil
}

// waitSignal awaits for SIGINT or SIGTERM
func waitSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	_ = log.Sync()
}

func main() {
	getopt.Parse()
	if help {
		printUsage(getopt.CommandLine, os.Stdout)
		os.Exit(0)
	}

	// Initiate log facility.
	if verbose {
		logOptions.SetOutputLevel(log.DefaultScopeName, log.DebugLevel)
	}
	if err := log.Configure(logOptions); err != nil {
		fmt.Fprintln(os.Stderr, "initiate log failed: ", err)
		os.Exit(-1)
	}

	if envStoreDsn == "" {
		log.Fatala(`option "env-store-dsn" is missing`)
		os.Exit(-1)
	}

	// Verify the files which need to be inspected.
	absFiles, relFiles, err := verifyInspectFiles(inspectFiles)
	if err != nil {
		log.Fatala("invalid inspect files found: ", err)
		os.Exit(-1)
	}

	// Connect to env store, i.e, consul.
	u, err := url.Parse(envStoreDsn)
	if err != nil {
		log.Fatala("invalid env store dsn: ", envStoreDsn)
		os.Exit(-1)
	}
	var s store.Store
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		s, err = consul.NewStore(envStoreDsn)
		if err != nil {
			log.Fatala("initiate consul env store failed: ", err)
			os.Exit(-1)
		}
	case "etcd":
		s, err = etcd.NewStore(envStoreDsn)
		if err != nil {
			log.Fatala("initiate etcd env store failed: ", err)
			os.Exit(-1)
		}
	default:
		log.Fatala("unknown env store schema: ", u.Scheme)
		os.Exit(-1)
	}

	// Register kv pairs
	for _, kv := range registerKVs {
		parts := strings.Split(kv, "=")
		if len(parts) < 2 {
			log.Warnf("invalid kv pair found: %v", kv)
			continue
		}
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if err := s.Set(store.Key{Name: k}, v); err != nil {
			log.Errorf("failed to register kv pair: %v, err: %v", kv, err)
			os.Exit(-1)
		}
	}

	// Get Proc options & args from env store and start the Proc.
	// Name conversion for the options & args, e.g:
	// enva --env-store-dsn http://localhost:8500 \
	// /usr/local/example-svc --oidc env://sso --ac env://ac --dsn postgres://postgres:password@env://postgres/example?sslmode=disable
	args := getopt.Args()
	if len(args) == 0 {
		log.Fatala("Proc name is missing")
		os.Exit(-1)
	}
	a, err := newAgent(s, args, absFiles, relFiles, defaultPatchTable())
	if err != nil {
		log.Fatala(err)
		os.Exit(-1)
	}

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		log.Info("env agent is terminating")
		cancel()
		wg.Wait()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.run(ctx)
	}()

	// Watch Proc options & args change and restart when the values changed.
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.watch(ctx)
	}()

	// TODO: Register Proc location if needed

	waitSignal()
}
