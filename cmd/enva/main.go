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

	"github.com/pborman/getopt/v2"
	"tespkg.in/envs/pkg/api"
	"tespkg.in/envs/pkg/enva"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/kit/log"
)

var (
	envsAddr             = ""
	osEnvFiles           = ""
	publishedKVs         []string
	locationRegistration bool
	verbose              bool
	help                 bool

	logOptions = log.DefaultOptions()
)

func init() {
	getopt.FlagLong(&envsAddr, "envs-addr", 'a', "Optional, envs address, eg: http://localhost:8502/a/bc")
	getopt.FlagLong(&osEnvFiles, "os-env-files", 'f', `Optional, os env files, separated by comma, eg: "path/to/index.html, path/to/config.js"`)
	getopt.FlagLong(&publishedKVs, "publish", 'p', `Optional, publish kvs, eg: --publish k1=v1 publish k2=v2`)
	getopt.FlagLong(&locationRegistration, "location", 'l', "Optional, enable Proc location registration")
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
	fmt.Fprintln(w, `Apart from the Command Options, there are OS Envs supported as well, 
ENVS_HTTP_ADDR, equivalent of Option "envs-addr", 
ENVA_OS_ENV_FILES, equivalent of Option "os-env-files", separated by comma, eg: "path/to/file1, path/to/file2",
ENVA_PUBLISH, equivalent of Option "publish", separated by comma, eg: "k1=v1, k2=v2",
ENVA_DEBUG, equivalent of Option "verbose", eg: ENVA_DEBUG=true equal to honor --verbose Command Option.
If both the command options & env are set at same time, Command Options have priority`)
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
	if verbose || os.Getenv("ENVA_DEBUG") == "true" {
		logOptions.SetLogCallers("default", true)
		logOptions.SetOutputLevel(log.DefaultScopeName, log.DebugLevel)
	}
	if err := log.Configure(logOptions); err != nil {
		fmt.Fprintln(os.Stderr, "initiate log failed: ", err)
		os.Exit(-1)
	}

	// Initiate envs client.
	if envsAddr == "" {
		envsAddr = os.Getenv("ENVS_HTTP_ADDR")
	}
	kvsClient, err := api.NewClient(&api.Config{
		Address: envsAddr,
	})
	if err != nil {
		log.Fatala("Initiate envs client failed", err)
	}

	// Analyse os env files
	if osEnvFiles == "" {
		osEnvFiles = os.Getenv("ENVA_OS_ENV_FILES")
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

	// Analyse publish key value pair
	if len(publishedKVs) == 0 {
		osEnvPublishKVs := os.Getenv("ENVA_PUBLISH")
		if osEnvPublishKVs != "" {
			parts := strings.Split(osEnvPublishKVs, ",")
			for _, part := range parts {
				kv := strings.TrimSpace(part)
				if kv == "" {
					continue
				}
				publishedKVs = append(publishedKVs, kv)
			}
		}
	}
	// Publish key value pair to envs
	for _, kv := range publishedKVs {
		ii := strings.Split(kv, "=")
		if len(ii) != 2 {
			log.Fatalf("invalid ENVA_PUBLISH key value pair, require key=value, got: %v", kv)
		}
		// Support publish env value only
		if err := kvsClient.Set(kvs.Key{
			Kind: kvs.EnvKind,
			Name: ii[0],
		}, ii[1]); err != nil {
			log.Fatalf("publish key value pair %v failed: %v", kv, err)
		}
	}

	// Get Proc options & args from env store and start the Proc.
	// Name conversion for the options & args, eg:
	// enva --envs-addr http://localhost:9112 \
	// /usr/local/example-svc --oidc ${env:// .sso } --ac ${env:// .ac } --dsn ${env:// .exampleServiceDSN } --config ${envf:// .exampleServiceConfig }
	args := getopt.Args()
	if len(args) == 0 {
		log.Fatala("Proc name is missing")
	}
	a, err := enva.NewAgent(kvsClient, args, finalisedOSEnvFiles, enva.DefaultRetry, enva.DefaultPatchTable())
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
		if err := a.Run(ctx); err != nil {
			log.Fatala(err)
		}
		log.Debuga("exit from run")
	}()

	// Watch Proc options & args change and restart when the values changed.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.Watch(ctx); err != nil {
			log.Fatala(err)
		}
		log.Debuga("exit from watch")
	}()

	// TODO: Register Proc location if needed
	if locationRegistration {
		log.Warna("location registration is unsupported yet")
	}

	waitSignal()
}
