package envs

import (
	"bytes"
	"fmt"
	"path/filepath"

	"tespkg.in/envs/pkg/openapi"
	"tespkg.in/kit/log"
)

type Args struct {
	ListenAddr string

	Dsn string

	// Static site asset location
	StaticAssetDir string
	// Static site asset access endpoint
	StaticAssetPath string
	// Spec URI access endpoint
	SpecPath string
	// Where to put the generated swagger json.
	// Default value is: KnownHost/SpecPath/swagger.yaml
	// Used for static html render.
	SwaggerJSONURI     string
	SwaggerCallbackURL string

	openapi.SpecArgs

	// The logging options to use
	LoggingOptions *log.Options
}

func DefaultArgs() *Args {
	return &Args{
		ListenAddr: ":9112",
		Dsn:        "http://localhost:8500/envs",

		StaticAssetDir:  "static",
		StaticAssetPath: "/_/static",
		SpecPath:        "/_/openapi/swagger.yaml",

		// Spec options
		SpecArgs: openapi.SpecArgs{
			KnownHost:    "localhost:9112",
			BasePath:     "/",
			Version:      "0.1.0",
			ContactEmail: "support@target-energysolutions.com",
			Title:        "Swagger envs",
			Schema:       "http",
		},

		LoggingOptions: log.DefaultOptions(),
	}
}

func (a *Args) validate() error {
	if a.Schema != "http" && a.Schema != "https" {
		return fmt.Errorf("unsupported schema, accept http or https only")
	}

	a.SwaggerJSONURI = a.Schema + "://" + filepath.Join(a.KnownHost, a.SpecPath)
	a.SwaggerCallbackURL = a.Schema + "://" + filepath.Join(a.KnownHost, a.StaticAssetPath, "oauth2-redirect.html")

	return nil
}

func (a *Args) String() string {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "Listening on: ", a.ListenAddr)
	fmt.Fprintln(buf, "Underlying dsn ", a.Dsn)
	return buf.String()
}
