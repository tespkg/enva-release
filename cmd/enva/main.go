package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pborman/getopt/v2"
)

var (
	envStoreDsn      = "http://localhost:8502"
	registerLocation bool
	registerKVs      []string
	inspectFiles     bool
	verbose          bool
	help             bool
)

func init() {
	getopt.FlagLong(&envStoreDsn, "env-store-dsn", 'a', "env store dsn")
	getopt.FlagLong(&registerLocation, "register-location", 'l', "register Proc location")
	getopt.FlagLong(&registerKVs, "register-kvs", 'k', `register extra k=v values to env store, e.g: register a=b and c=d by using "-k a=b -k c=d"`)
	getopt.FlagLong(&inspectFiles, "inspect-files", 'i',
		`inspect files and replace the key with values in env store, if there is any key can't found in env store then prompt an error'`)
	getopt.Flag(&verbose, 'v', "be verbose")
	getopt.FlagLong(&help, "help", 'h', "display usage")
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
	fmt.Fprintln(w, `The supported key pattern in inspected files are: ^\{\{ \.Consul_([a-z].*)_([a-z].*) \}\}\n
For example if there is a string "{{ .Consul_Workspace_OauthID }}" in a file, 
then it will be replaced with value of key "Workspace/OauthID" in env store.`)
}

func main() {
	getopt.Parse()
	if help {
		printUsage(getopt.CommandLine, os.Stdout)
		return
	}

	// Connect to env store, i.e, consul.

	// Get Proc options & args from env store and start the Proc.
	// Name conversion for the options & args, e.g:
	// enva --env-store-dsn http://localhost:8500 /usr/local/example-svc --oidc env://sso --ac env://ac --dsn postgres://postgres:password@env://postgres:5432/example?sslmode=disable

	// Watch Proc options & args change and restart when the values changed.

	// Register Proc location if needed

	args := getopt.Args()
	fmt.Println(args)
}
