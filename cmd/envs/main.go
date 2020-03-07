package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"tespkg.in/envs/pkg/envs"
)

func cmd() *cobra.Command {
	a := envs.DefaultArgs()

	cmd := &cobra.Command{
		Use:   "envs",
		Short: "environment store",
		Long:  "environment store service",
		Run: func(cmd *cobra.Command, args []string) {
			runServer(a)
		},
	}

	cmd.Flags().StringVar(&a.ListenAddr, "address", a.ListenAddr, "The address which server will serve at")
	cmd.Flags().StringVar(&a.Dsn, "dsn", a.Dsn, "Data source name, support etcd & consul, e.g, consul: http://localhost:8502/envs, etcd: etcd://localhost:2379")
	cmd.Flags().StringVar(&a.SpecsMigrationPath, "specs-path", "", "Pre-configured application/service specs")

	cmd.Flags().StringVar(&a.StaticAssetDir, "asset-dir", a.StaticAssetDir, "the static site asset dir")
	cmd.Flags().StringVar(&a.StaticAssetPath, "asset-path", a.StaticAssetPath, "asset path which can be access publicly")
	cmd.Flags().StringVar(&a.OpenAPISpecPath, "openapi-spec-path", a.OpenAPISpecPath, "openAPI spec base URI which can be access publicly")

	cmd.Flags().StringVar(&a.KnownHost, "host", a.KnownHost, "the host for serving these generated OpenAPIs")
	cmd.Flags().StringVar(&a.BasePath, "base-path", a.BasePath, "the base path for the generated OpenAPIs")
	cmd.Flags().StringVar(&a.Version, "version", a.Version, "the api version")
	cmd.Flags().StringVar(&a.ContactEmail, "contact-email", a.ContactEmail, "the contact email for supporting")
	cmd.Flags().StringVar(&a.Title, "title", a.Title, "title for the generated OpenAPIs")
	cmd.Flags().StringVar(&a.Schema, "schema", a.Schema, "the supported schema, accept http or https")

	a.LoggingOptions.AttachCobraFlags(cmd)

	return cmd
}

func runServer(sa *envs.Args) {
	log.Printf("server started with: \n%v\n", sa)

	s, err := envs.New(sa)
	if err != nil {
		log.Fatalf("ubable to intialise server: %v\n", err)
	}

	s.Run()
	err = s.Wait()
	if err != nil {
		log.Fatalf("server unexpectedly terminated: %v\n", err)
	}

	s.Close()
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
