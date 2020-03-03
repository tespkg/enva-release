package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tespkg.in/kit/templates"

	"golang.org/x/tools/godoc/util"
	"tespkg.in/envs/pkg/kvs"

	"tespkg.in/envs/pkg/api"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"tespkg.in/kit/log"
)

type args struct {
	listenAddr string
	// Static site asset location
	staticAssetDir string

	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	exclusiveFiles   []string
	allowCredentials bool

	// The logging options to use
	loggingOptions *log.Options
}

func defaultArgs() *args {
	return &args{
		listenAddr:       ":9112",
		staticAssetDir:   "static",
		allowOrigins:     []string{"*"},
		allowMethods:     []string{"*"},
		allowHeaders:     []string{"*"},
		exposeHeaders:    []string{"*"},
		exclusiveFiles:   []string{"*.css", "*.css.map", "js/*"},
		allowCredentials: true,
		loggingOptions:   log.DefaultOptions(),
	}
}

func cmd() *cobra.Command {
	a := defaultArgs()

	cmd := &cobra.Command{
		Use:   "s4",
		Short: "Simple Static Site Service",
		Long:  "Simple static site service serve static assets as a site",
		Run: func(cmd *cobra.Command, args []string) {
			runServer(a)
		},
	}

	cmd.Flags().StringVar(&a.listenAddr, "address", a.listenAddr, "the address which server will serve at")
	cmd.Flags().StringVar(&a.staticAssetDir, "asset-dir", a.staticAssetDir, "the static site asset dir")
	cmd.Flags().StringSliceVar(&a.allowOrigins, "allow-origins", a.allowOrigins, "allow origins")
	cmd.Flags().StringSliceVar(&a.allowMethods, "allow-methods", a.allowMethods, "allow methods")
	cmd.Flags().StringSliceVar(&a.allowHeaders, "allow-headers", a.allowHeaders, "allow headers")
	cmd.Flags().StringSliceVar(&a.exposeHeaders, "expose-headers", a.exposeHeaders, "expose headers")
	cmd.Flags().StringSliceVar(&a.exclusiveFiles, "skip-render-files", a.exclusiveFiles, "skip render files' pattern")
	cmd.Flags().BoolVar(&a.allowCredentials, "allow-cres", a.allowCredentials, "allow credentials")
	a.loggingOptions.AttachCobraFlags(cmd)

	return cmd
}

func runServer(a *args) {
	a.loggingOptions.SetLogCallers("default", true)
	if err := log.Configure(a.loggingOptions); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	// Initiate envs client by using os.env ENVS_HTTP_ADDR.
	kvStore, err := api.NewClient(&api.Config{})
	if err != nil {
		log.Fatala("Initiate envs client failed", err)
	}

	// Create gin engine
	ge := gin.Default()
	ge.Use(cors.New(cors.Config{
		AllowOrigins:     a.allowOrigins,
		AllowMethods:     a.allowMethods,
		AllowHeaders:     a.allowHeaders,
		ExposeHeaders:    a.exposeHeaders,
		AllowCredentials: a.allowCredentials,
		MaxAge:           12 * time.Hour,
	}))

	// Static assets
	if err := renderAssets(kvStore, a.staticAssetDir, a.exclusiveFiles...); err != nil {
		log.Fatala("render assets failed ", err)
	}
	ge.Static("/", finalisedAssetsDir(a.staticAssetDir)).Use(gzip.Gzip(gzip.DefaultCompression))

	err = http.ListenAndServe(a.listenAddr, ge)
	if err != nil {
		log.Errora(err)
	}
}

func finalisedAssetsDir(assetDir string) string {
	return "." + filepath.Base(assetDir)
}

func renderAssets(kvStore kvs.KVStore, assetDir string, exclusiveFiles ...string) error {
	outputDir := finalisedAssetsDir(assetDir)

	err := filepath.Walk(assetDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				name, err := filepath.Rel(assetDir, path)
				if err != nil {
					return err
				}
				bs, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				output := filepath.Join(outputDir, name)
				if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
					return err
				}
				f, err := os.Create(output)
				if err != nil {
					return err
				}
				defer f.Close()

				if !util.IsText(bs) || matchWildcardsPatterns(exclusiveFiles, name) {
					_, err := f.Write(bs)
					return err
				}

				if err := kvs.Render(kvStore, bytes.NewBuffer(bs), f); err != nil {
					return fmt.Errorf("render file %v failed: %v", name, err)
				}

				return nil
			}
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

func matchWildcardsPatterns(wildcardPatterns []string, path string) bool {
	for _, pattern := range wildcardPatterns {
		if templates.Match(pattern, path) {
			return true
		}
	}
	return false
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
