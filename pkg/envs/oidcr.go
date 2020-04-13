package envs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"tespkg.in/kit/log"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/store"
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

	// OAuth2Host Added for front-end compatibility.
	// because the way of frontend doing oauth redirect is:
	// frontend expose an oauth2 host to DevOps for customizing and use a
	// fix/hardcoded redirect path(prefixed with the oauth2 host),
	// which is "sso/callback", to serve the oidc redirect URI callback,
	// instead of exposing oauth2 redirect URI option to the DevOps explicitly.
	OAuth2Host string `json:"OAuth2Host"`

	// It is IMPORTANT to know the clients.name is the key prefix to store the key & value pairs in env store
	// And the allowed name pattern is "[\-_a-zA-Z0-9]*".
	// For example, If the following oidc client was registered via oidcr API:
	//
	// clients:
	// - name: ssoOAuth2
	//   OAuth2Host: http://localhost:5555
	//   redirectURIs:
	//   - http://localhost:5555/callback
	//   allowedAuthTypes:
	//   - authorization_code
	//   - implicit
	//   - client_credentials
	//   - password_credentials
	//
	// These key & value pairs will stored in env store:
	// 1. ssoOAuth2ClientID=GeneratedClientID
	// 2. ssoOAuth2ClientSecret=GeneratedSecret
	// 3. ssoOAuth2RedirectURI=http://localhost:5555/callback
	// 4. ssoOAuth2Host=http://localhost:5555
	Name string `json:"name" yaml:"name"`
}

type ClientReqWithID struct {
	ID string `json:"id" yaml:"id"`
	ClientReq
}

type ClientReqs []ClientReq

type OAuthProviderConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

type OAuthRegistrationReq struct {
	ProviderConfig OAuthProviderConfig `json:"provider-config"`
	Requests       ClientReqs          `json:"clients"`
}

func registerOAuthClients(s kvs.KVStore, provider OAuthProviderConfig, reqs ClientReqs) error {
	// Get authentication from oidc via client credential flow
	oidcProvider, err := oidc.NewProvider(context.Background(), provider.Issuer)
	if err != nil {
		return fmt.Errorf("get oidc provider failed: %v", err)
	}
	oauth2Config := oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}
	tok, err := oauth2Config.PasswordCredentialsToken(context.Background(), provider.Username, provider.Password)
	if err != nil {
		return fmt.Errorf("get token failed: %v", err)
	}

	var closer []io.Closer
	defer func() {
		for _, c := range closer {
			c.Close()
		}
	}()

	// Register client to oidc provider & envs one by one.
	for _, client := range reqs {
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
		clientID, err := s.Get(
			kvs.Key{
				Kind: kvs.EnvKind,
				Name: client.Name + "ClientID",
			},
		)
		if err != nil && !errors.Is(err, kvs.ErrNotFound) {
			return err
		}
		if err == nil && clientID != "" {
			// Update existed/registered client
			reqWithID := ClientReqWithID{
				ID:        clientID,
				ClientReq: client,
			}

			data, _ := json.Marshal(&reqWithID)
			req, err := http.NewRequest(http.MethodPut, provider.Issuer+"/client", bytes.NewBuffer(data))
			if err != nil {
				return err
			}
			tok.SetAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("update client %v failed: %v", client.Name, err)
			}
			closer = append(closer, resp.Body)

			data, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			var result struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
				Error   string `json:"error"`
			}
			if err := json.Unmarshal(data, &result); err != nil {
				return err
			}
			if result.Success {
				// Publish env keys to envs
				if err := publishOIDCClient(s, client.Name+"RedirectURI", client.RedirectURIs[0]); err != nil {
					return fmt.Errorf("publish %v RedirectURI failed %v", client.Name, err)
				}
				if client.OAuth2Host != "" {
					if err := publishOIDCClient(s, client.Name+"Host", client.OAuth2Host); err != nil {
						return fmt.Errorf("publish %v RedirectURI failed %v", client.Name, err)
					}
				}

				continue
			}

			log.Warnf("update %v with id %v failed %v, trying creating", client.Name, clientID, result.Error)
		}

		// Register new client to oidc provider
		data, _ := json.Marshal(&client)
		req, err := http.NewRequest(http.MethodPost, provider.Issuer+"/client", bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		tok.SetAuthHeader(req)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		closer = append(closer, resp.Body)

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
		if err := publishOIDCClient(s, client.Name+"ClientID", result.Data.ID); err != nil {
			return fmt.Errorf("publish %v ClientID failed %v", client.Name, err)
		}
		if err := publishOIDCClient(s, client.Name+"ClientSecret", result.Data.Secret); err != nil {
			return fmt.Errorf("publish %v ClientSecret failed %v", client.Name, err)
		}
		if err := publishOIDCClient(s, client.Name+"RedirectURI", result.Data.RedirectURIs[0]); err != nil {
			return fmt.Errorf("publish %v RedirectURI failed %v", client.Name, err)
		}
		if client.OAuth2Host != "" {
			if err := publishOIDCClient(s, client.Name+"Host", client.OAuth2Host); err != nil {
				return fmt.Errorf("publish %v RedirectURI failed %v", client.Name, err)
			}
		}
	}
	return nil
}

func publishOIDCClient(s kvs.KVStore, key, value string) error {
	return s.Set(
		kvs.Key{
			Kind: kvs.EnvKind,
			Name: key,
		},
		value,
	)
}

type kvStore struct {
	store.Store
}

func (kv *kvStore) Get(key kvs.Key) (string, error) {
	val, err := kv.Store.Get(
		store.Key{
			Namespace: store.DefaultKVNs,
			Kind:      key.Kind,
			Name:      key.Name,
		},
	)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", kvs.ErrNotFound
		}
		return "", err
	}
	value, ok := val.(string)
	if !ok {
		return "", errors.New("invalid val type")
	}
	return value, nil
}

func (kv *kvStore) Set(key kvs.Key, val string) error {
	return kv.Store.Set(
		store.Key{
			Namespace: store.DefaultKVNs,
			Kind:      key.Kind,
			Name:      key.Name,
		},
		val,
	)
}

func newKVStore(s store.Store) *kvStore {
	return &kvStore{Store: s}
}
