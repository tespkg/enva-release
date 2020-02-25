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
	a.LoggingOptions.AttachCobraFlags(cmd)

	return cmd
}

func runServer(sa *envs.Args) {
	s, err := envs.New(sa)
	if err != nil {
		log.Fatalf("ubable to intialise server: %v\n", err)
	}

	log.Printf("server started with: \n%v\n", sa)

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
