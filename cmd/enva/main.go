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
	"tespkg.in/envs/pkg/ssparser"
	"tespkg.in/envs/version"
	"tespkg.in/kit/log"
)

var (
	envsAddr             = ""
	envsNamespace        = ""
	envFiles             = ""
	envTplFiles          = ""
	publishedKVs         []string
	isRunOnlyOnce        bool
	locationRegistration bool
	logLevel             string
	help                 bool
	metricsEndpoint      string

	logOptions = log.DefaultOptions()
)

func init() {
	getopt.FlagLong(&envsAddr, "envs-addr", 'a', "Optional, envs address, eg: http://localhost:8502/a/bc")
	getopt.FlagLong(&envsNamespace, "envs-namespace", 'n', "Optional, envs namespace, eg: dev")
	getopt.FlagLong(&envFiles, "env-files", 'f', `Optional, env files, separated by comma, eg: "path/to/index.html, path/to/config.js"`)
	getopt.FlagLong(&envTplFiles, "env-tpl-files", 't', `Optional, env template files, separated by comma, eg: "path/to/index.html.tpl, path/to/config.js.tpl", pair to env-files with same index`)
	getopt.FlagLong(&publishedKVs, "publish", 'p', `Optional, publish kvs, eg: --publish k1=v1 --publish k2=v2, support os env evaluation in value, e.g k2=$VERSION, or k2=v${VERSION}r1`)
	getopt.FlagLong(&isRunOnlyOnce, "run-only-once", 'r', "Optional, run Proc only once then exit")
	getopt.FlagLong(&locationRegistration, "location", 'l', "Optional, enable Proc location registration")
	getopt.FlagLong(&logLevel, "log-level", 'v', "Optional, log level, can be one of debug, info, warn, error, fatal, none")
	getopt.FlagLong(&metricsEndpoint, "metrics-endpoint", 's', "Optional, start a HTTP server to serve metrics endpoints, i.e, /healthz and /metrics")
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
ENVS_NAMESPACE, equivalent of Option "envs-namespace", 
ENVA_ENV_FILES, equivalent of Option "env-files", separated by comma, eg: "path/to/file1, path/to/file2",
ENVA_ENV_TEMPLATE_FILES, equivalent of Option "env-tpl-files", separated by comma, eg: "path/to/file1.tpl, path/to/file2.tpl, pair to env-files with same index",
ENVA_PUBLISH, equivalent of Option "publish", separated by comma, eg: "k1=v1, k2=v2",
ENVA_RUN_ONLY_ONCE, equivalent of Option "run-only-once", eg: ENVA_RUN_ONLY_ONCE=true equal to honor --run-only-once Option.
ENVA_LOG_LEVEL, equivalent of Option "log-level", eg: ENVA_LOG_LEVEL=debug equal to honor --log-level=debug Command Option.
ENVA_METRICS_ENDPOINT, equivalent of Option "metrics-endpoint", eg: ENVA_METRICS_ENDPOINT=http://127.0.0.1:8503 equal to honor --metrics-endpoint=http://127.0.0.1:8503 Command Option.
If both the command options & env are set at same time, Command Options have priority`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "version: "+version.Version)
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
	if logLevel == "" {
		logLevel = os.Getenv("ENVA_LOG_LEVEL")
	}
	stringToLevel := map[string]log.Level{
		"debug": log.DebugLevel,
		"info":  log.InfoLevel,
		"warn":  log.WarnLevel,
		"error": log.ErrorLevel,
		"fatal": log.FatalLevel,
		"none":  log.NoneLevel,
	}
	if logLevel != "" {
		lvl, ok := stringToLevel[logLevel]
		if !ok {
			fmt.Fprintln(os.Stderr, "invalid log level", logLevel)
			os.Exit(-1)
		}
		if lvl == log.DebugLevel {
			logOptions.SetLogCallers("default", true)
		}
		logOptions.SetOutputLevel(log.DefaultScopeName, lvl)
	}
	if err := log.Configure(logOptions); err != nil {
		fmt.Fprintln(os.Stderr, "initiate log failed: ", err)
		os.Exit(-1)
	}

	if !isRunOnlyOnce && os.Getenv("ENVA_RUN_ONLY_ONCE") == "true" {
		isRunOnlyOnce = true
	}

	// Initiate envs client.
	if envsAddr == "" {
		envsAddr = os.Getenv("ENVS_HTTP_ADDR")
	}
	if envsNamespace == "" {
		envsNamespace = os.Getenv("ENVS_NAMESPACE")
	}
	kvsClient, err := api.NewClient(&api.Config{
		Address:   envsAddr,
		Namespace: envsNamespace,
	})
	if err != nil {
		log.Fatala("Initiate envs client failed", err)
	}

	// Analyze env files
	var envFilenames []string
	osEnvFiles := os.Getenv("ENVA_ENV_FILES")
	if envFiles == "" {
		envFiles = osEnvFiles
	} else {
		envFiles = strings.Join([]string{envFiles, osEnvFiles}, ",")
	}
	parts := strings.Split(envFiles, ",")
	for _, part := range parts {
		fn := strings.TrimSpace(part)
		if fn == "" {
			continue
		}
		envFilenames = append(envFilenames, fn)
	}

	// Analyze env template files
	var envTplFilenames []string
	osEnvTplFiles := os.Getenv("ENVA_ENV_TEMPLATE_FILES")
	if envTplFiles == "" {
		envTplFiles = osEnvTplFiles
	} else {
		envTplFiles = strings.Join([]string{envTplFiles, osEnvTplFiles}, ",")
	}
	parts = strings.Split(envTplFiles, ",")
	for _, part := range parts {
		fn := strings.TrimSpace(part)
		if fn == "" {
			continue
		}
		envTplFilenames = append(envTplFilenames, fn)
	}
	if len(envTplFilenames) > 0 {
		if len(envTplFilenames) != len(envFilenames) {
			log.Fatala("invalid pairs of env-files to env-template-files")
		}
	}

	// finalise env files
	var finalisedEnvFiles []enva.EnvFile
	for i, envFilename := range envFilenames {
		tplFilename := ""
		if len(envTplFilenames) > i {
			tplFilename = envTplFilenames[i]
		}

		finalisedEnvFiles = append(finalisedEnvFiles, enva.EnvFile{
			TemplateFilePath: tplFilename,
			Filename:         envFilename,
		})
	}

	// Analyze publish key value pair
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

	// Publish key value pair to envs
	visitedKey := make(map[string]struct{})
	for _, kv := range publishedKVs {
		k, v, err := extractKV(strings.TrimSpace(kv))
		if err != nil {
			log.Fatalf("invalid publish key value pair, require key=value, got: %v, err: %v", kv, err)
		}

		// If duplicated key found, command option has priority
		// If duplicated found in both command options or os env, only the first one would be count
		if _, ok := visitedKey[k]; ok {
			log.Warnf("ignore duplicated publish key:%v with value: %v ", k, v)
			continue
		}
		visitedKey[k] = struct{}{}

		// Support publish env value only
		if err := kvsClient.Set(kvs.Key{
			Kind: kvs.EnvKind,
			Name: k,
		}, v); err != nil {
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
	a, err := enva.NewAgent(kvsClient, args, finalisedEnvFiles, isRunOnlyOnce, enva.DefaultRetry, enva.DefaultPatchTable())
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

		// If run got finished, exit the main Proc directly.
		_ = raise(syscall.SIGTERM)
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

	if metricsEndpoint == "" {
		metricsEndpoint = os.Getenv("ENVA_METRICS_ENDPOINT")
	}
	if metricsEndpoint != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.ServeMetrics(ctx, metricsEndpoint); err != nil {
				log.Fatala(err)
			}
		}()
	}

	// TODO: Register Proc location if needed
	if locationRegistration {
		log.Warna("location registration is unsupported yet")
	}

	waitSignal()
}

func raise(sig os.Signal) error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(sig)
}

func extractKV(kv string) (k, v string, err error) {
	ii := strings.SplitN(kv, "=", 2)
	if len(ii) != 2 {
		return "", "", fmt.Errorf("invalid ENVA_PUBLISH key value pair, require key=value, got: %v", kv)
	}
	k = strings.TrimSpace(ii[0])
	rv := strings.TrimSpace(ii[1])
	v, err = ssparser.Parse(rv)
	if err != nil {
		return "", "", fmt.Errorf("invalid key %v with value: %v, err: %v", k, rv, err)
	}
	return k, v, nil
}
