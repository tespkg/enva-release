package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	"github.com/coreos/go-oidc"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"tespkg.in/envs/pkg/api"
	"tespkg.in/envs/pkg/kvs"
)

var nameRegex = regexp.MustCompile(`[\-_a-zA-Z0-9]*`)

type ClientReq struct {
	// supported authentication types
	//
	// authorization_code
	// implicit
	// password_credentials
	// client_credentials
	AllowedAuthTypes []string `json:"allowedAuthTypes" yaml:"allowedAuthTypes"`

	// A registered set of redirect URIs. When redirecting from dex to the client, the URI
	// requested to redirect to MUST match one of these values, unless the client is "public".
	RedirectURIs []string `json:"redirectURIs" yaml:"redirectURIs"`

	// Name fully qualified oidc client name, allowed name pattern `[\-_a-zA-Z0-9]*`.
	// Use as the prefix for clientID, clientSecret and redirectURI in env store
	// Eg, If the following oidc client registered in oidc provider:
	//	{
	//		Name:  "example-app",
	//		Secret: secret,
	//		RedirectUris: []string{
	//			"http://localhost:8080/oauth2",
	//		},
	//		AllowedAuthTypes: []string{
	//			"authorization_code",
	//			"implicit",
	//			"client_credentials",
	//			"password_credentials",
	//		},
	//	}
	// These key & value pairs will stored in envs
	// example-appClientID=example-app
	// example-appClientSecret=secret
	// example-appRedirectURI=http://localhost8080/oauth2
	Name string `json:"name" yaml:"name"`

	// OAuth2Host Added for front-end compatibility.
	// The way of frontend doing oauth redirect is:
	// They expose an oauth2 host to DevOps for customizing and use a
	// fix/hardcoded redirect path(prefixed with the oauth2 host), which is "sso/callback", to serve the oidc redirect URI callback,
	// instead of exposing oauth2 redirect URI option to the DevOps explicitly.
	OAuth2Host string `json:"OAuth2Host"`
}

type ClientReqs []ClientReq

var (
	oidcIssuer   string
	clientID     string
	clientSecret string
	username     string
	password     string
	envsHTTPAddr string
	requestFile  string
)

func cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oidcr [Options] <stdin|file>",
		Short: "oidc register",
		Long:  "OpenID Connect ClientID, ClientSecret, Redirect callback address registration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Connect to envs
			if envsHTTPAddr == "" {
				envsHTTPAddr = os.Getenv("ENVS_HTTP_ADDR")
			}
			if envsHTTPAddr == "" {
				return errors.New("envs HTTP address required")
			}
			envsClient, err := api.NewClient(&api.Config{Address: envsHTTPAddr})
			if err != nil {
				return fmt.Errorf("create envs cient fialed: %v", err)
			}

			fmt.Println("----", oidcIssuer, clientID, clientSecret)
			// Setup request source
			r := os.Stdin
			if requestFile != "" {
				f, err := os.Open(requestFile)
				if err != nil {
					return fmt.Errorf("open request file failed: %v", err)
				}
				defer f.Close()
				r = f
			}

			// Parse registration request
			data, err := ioutil.ReadAll(r)
			if err != nil {
				return fmt.Errorf("get registration request failed: %v", err)
			}
			fmt.Println("======data: \n", string(data))
			clients := ClientReqs{}
			if err := yaml.Unmarshal(data, &clients); err != nil {
				return fmt.Errorf("invalid registration request: %v", err)
			}

			// Get authentication from oidc via client credential flow
			oidcProvider, err := oidc.NewProvider(context.Background(), oidcIssuer)
			if err != nil {
				return fmt.Errorf("get oidc provider failed: %v", err)
			}
			oauth2Config := oauth2.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Endpoint:     oidcProvider.Endpoint(),
				Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
			}
			tok, err := oauth2Config.PasswordCredentialsToken(context.Background(), username, password)
			if err != nil {
				return fmt.Errorf("get token failed: %v", err)
			}

			// Register client to oidc provider & envs one by one.
			for _, client := range clients {
				if client.Name == "" {
					return errors.New("found empty name, not allowed")
				}
				r := nameRegex.FindAllStringSubmatch(client.Name, -1)
				if len(r) == 0 {
					return errors.New(`invalid name pattern, expected "[\-_a-zA-Z0-9]*"`)
				}
				if len(client.AllowedAuthTypes) == 0 {
					return fmt.Errorf("fuound empty auth types on %v, not allowed", client.Name)
				}
				if len(client.RedirectURIs) != 1 {
					return fmt.Errorf("unexpected redirectURI count on %v, expect 1, got %v", client.Name, len(client.RedirectURIs))
				}

				// Check if the client already existed in envs
				_, err = envsClient.Get(kvs.Key{
					Kind: kvs.EnvKind,
					Name: client.Name + "ClientID",
				})
				if err != nil && !errors.Is(err, kvs.ErrNotFound) {
					return err
				}
				if !errors.Is(err, kvs.ErrNotFound) {
					// Ignore existed/registered client
					continue
				}

				// Register new client to oidc provider
				data, _ := json.Marshal(&client)
				req, err := http.NewRequest("POST", oidcIssuer+"/client", bytes.NewBuffer(data))
				if err != nil {
					return err
				}
				tok.SetAuthHeader(req)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()

				data, err = ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				var result struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
					Error   string `json:"error"`
					Data    struct {
						ID               string   `json:"id" yaml:"id"`
						Secret           string   `json:"secret" yaml:"secret"`
						AllowedAuthTypes []string `json:"allowedAuthTypes" yaml:"allowedAuthTypes"`
						RedirectURIs     []string `json:"redirectURIs" yaml:"redirectURIs"`
						Name             string   `json:"name" yaml:"name"`
					} `json:"data,omitempty"`
				}
				if err := json.Unmarshal(data, &result); err != nil {
					return err
				}
				if !result.Success {
					return fmt.Errorf("create %v failed %v", client.Name, result.Error)
				}

				// Publish env keys to envs
				if err := publishOidcClient(envsClient, client.Name+"ClientID", result.Data.ID); err != nil {
					return fmt.Errorf("publish %v ClientID failed %v", client.Name, err)
				}
				if err := publishOidcClient(envsClient, client.Name+"ClientSecret", result.Data.Secret); err != nil {
					return fmt.Errorf("publish %v ClientSecret failed %v", client.Name, err)
				}
				if err := publishOidcClient(envsClient, client.Name+"RedirectURI", result.Data.RedirectURIs[0]); err != nil {
					return fmt.Errorf("publish %v RedirectURI failed %v", client.Name, err)
				}
				if client.OAuth2Host != "" {
					if err := publishOidcClient(envsClient, client.Name+"OauthHost", client.OAuth2Host); err != nil {
						return fmt.Errorf("publish %v RedirectURI failed %v", client.Name, err)
					}
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&oidcIssuer, "oidc", "", "oidc provider hostname")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID of this application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret of this application.")
	cmd.Flags().StringVar(&username, "username", "", "username")
	cmd.Flags().StringVar(&password, "password", "", "password")
	cmd.Flags().StringVar(&envsHTTPAddr, "envs-addr", "", "envs http address, if not given use ENVS_HTTP_ADDR as a alternative value")
	cmd.Flags().StringVarP(&requestFile, "reqf", "r", "", "path to registration req file, if not given use stdin")

	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		defaultHelpFunc(command, strings)
		clients := ClientReqs{
			{
				Name: "example-app",
				AllowedAuthTypes: []string{
					"authorization_code",
					"implicit",
					"client_credentials",
					"password_credentials",
				},
				RedirectURIs: []string{"http://localhost:8080/oauth2"},
			},
			{
				Name: "envs",
				AllowedAuthTypes: []string{
					"authorization_code",
					"implicit",
					"client_credentials",
					"password_credentials",
				},
				RedirectURIs: []string{"http://localhost:9112/oauth2"},
			},
		}
		out, _ := yaml.Marshal(&clients)
		command.Println()
		command.Println("Example of registration request: ")
		command.Println(string(out))
	})

	return cmd
}

func publishOidcClient(c *api.Client, key, value string) error {
	return c.Set(kvs.Key{
		Kind: kvs.EnvKind,
		Name: key,
	}, value)
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
