package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"tespkg.in/envs/pkg/api"

	"github.com/pborman/getopt/v2"
	"tespkg.in/kit/log"
)

var (
	envsAddr         = ""
	osEnvFiles       = ""
	registerLocation bool
	verbose          bool
	help             bool

	logOptions = log.DefaultOptions()
)

func init() {
	getopt.FlagLong(&envsAddr, "envs-addr", 'e', "Optional, envs address, use ENVS_HTTP_ADDR env if not given, e.g, http://localhost:8502/a/bc")
	getopt.FlagLong(&osEnvFiles, "os-env-files", 'f', `Optional, os env files, use OS_ENV_FILES env if not given, e.g, "path/to/index.html, path/to/config.js"`)
	getopt.FlagLong(&registerLocation, "location", 'l', "Optional, enable Proc location register")
	getopt.FlagLong(&verbose, "verbose", 'v', "Optional, be verbose")
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
		"<Proc>",
		"[Proc options]",
		"[Proc args]",
	}
	fmt.Fprintln(w, strings.Join(parts, " "))
	s.PrintOptions(w)
	fmt.Fprintln(w)
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

	// Initiate envs client.
	kvs, err := api.NewClient(&api.Config{
		Address: envsAddr,
	})
	if err != nil {
		log.Fatala("Initiate envs client failed", err)
	}

	// Analyse os env files
	if osEnvFiles == "" {
		osEnvFiles = os.Getenv("OS_ENV_FILES")
	}
	var finalisedOSEnvFiles []string
	parts := strings.Split(osEnvFiles, ",")
	for _, part := range parts {
		fn := strings.TrimSpace(part)
		if fn == "" {
			continue
		}
		finalisedOSEnvFiles = append(finalisedOSEnvFiles, fn)
	}

	// Get Proc options & args from env store and start the Proc.
	// Name conversion for the options & args, e.g:
	// enva --env-addr http://localhost:9112 \
	// /usr/local/example-svc --oidc ${env:// .sso } --ac ${env:// .ac } --dsn ${env:// .exampleServiceDSN } --config ${envf:// .exampleServiceConfig }
	args := getopt.Args()
	if len(args) == 0 {
		log.Fatala("Proc name is missing")
	}
	a, err := newAgent(kvs, args, finalisedOSEnvFiles, defaultRetry, defaultPatchTable())
	if err != nil {
		log.Fatala(err)
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
		if err := a.run(ctx); err != nil {
			log.Fatala(err)
		}
	}()

	// Watch Proc options & args change and restart when the values changed.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.watch(ctx); err != nil {
			log.Fatala(err)
		}
	}()

	// TODO: Register Proc location if needed

	waitSignal()
}
