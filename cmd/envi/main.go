package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pborman/getopt/v2"
	"meera.tech/envs/pkg/store"
	"meera.tech/envs/pkg/store/consul"
	"meera.tech/envs/pkg/store/etcd"
	"meera.tech/kit/log"
)

var (
	envStoreDsn = "http://localhost:8502/a/bc"
	registerKVs []string
	verbose     bool
	help        bool

	logOptions = log.DefaultOptions()
)

func init() {
	getopt.FlagLong(&envStoreDsn, "envs", 'e', "Required, env store dsn")
	getopt.FlagLong(&registerKVs, "kvs", 'k',
		`Optional, register k=v value pair to env store, e.g: register a=b and c=d by using "-k a=b -k c=d"`)
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
		"<vars.yaml>",
	}
	fmt.Fprintln(w, strings.Join(parts, " "))
	fmt.Fprintln(w, "Import vars into env store, available options are:")
	s.PrintOptions(w)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `Check vars.yaml example:
$ cat vars.yaml
postgres: "localhost:5432"
mongo: "localhost:27017"`)
	fmt.Fprintln(w)
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

	args := getopt.Args()
	if len(args) != 1 {
		log.Fatala(`should have exact on arg to specific vars yaml filename`)
		os.Exit(-1)
	}
	bVars, err := ioutil.ReadFile(args[0])
	if err != nil {
		log.Fatala("read vars yaml file failed: ", err)
		os.Exit(-1)
	}

	var kvs []string
	for _, kv := range registerKVs {
		parts := strings.Split(kv, "=")
		if len(parts) < 2 {
			log.Warnf("invalid kv pair found: %v", kv)
			continue
		}
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		kvs = append(kvs, k, v)
	}

	vars := make(map[string]string)
	if err := yaml.Unmarshal(bVars, &vars); err != nil {
		log.Fatala("unmarshal vars yaml file failed: ", err)
		os.Exit(-1)
	}
	for k, v := range vars {
		kvs = append(kvs, k, v)
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

	for i := 0; i < len(kvs); i += 2 {
		k, v := kvs[i], kvs[i+1]
		if err := s.Set(store.Key{Name: k}, v); err != nil {
			log.Errorf("failed to register kv pair: %v=%v, err: %v", k, v, err)
			os.Exit(-1)
		}
		log.Infof("set %v=%v pair", k, v)
	}
}
