package envs

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	openspec "github.com/go-openapi/spec"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/openapi"
)

const (
	keyValTag     = "KeyVal"
	envKeyValTag  = "EnvKeyVal"
	fileKeyValTag = "FileKeyVal"
	addOnsTag     = "Add-ons"
	keyValDef     = "keyVal"
	envKeyValDef  = "envKeyVal"
)

// GenerateSpec generate openapi spec
func GenerateSpec(iw io.Writer, sa openapi.SpecArgs) error {
	// Generate model definitions
	// 1. Definition for spec.KeyVal & spec.KeyVals model
	// 2. Definition for spec.Header & spec.Spec & spec.Specs model
	specDefs := map[string]openspec.Schema{
		keyValDef:    openapi.GenerateModel(reflect.ValueOf(kvs.KeyVal{})),
		envKeyValDef: openapi.GenerateModel(reflect.ValueOf(kvs.EnvKeyVal{})),
	}

	tags := []openspec.Tag{
		{
			TagProps: openspec.TagProps{
				Name: keyValTag,
			},
		},
		{
			TagProps: openspec.TagProps{
				Name: envKeyValTag,
			},
		},
		{
			TagProps: openspec.TagProps{
				Name: fileKeyValTag,
			},
		},
		{
			TagProps: openspec.TagProps{
				Name: addOnsTag,
			},
		},
	}

	pathItems := make(map[string]openspec.PathItem)

	// 1. keys GET
	pathItems["/keys"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "GetKeys",
					Summary:     "Get multiple keys with or without filter",
					Description: "Get multiple keys with or without filter",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "kind", "string", "", false, nil).
							WithParameterDesc("Get keyvals by kind, supported kinds: env, envf, envo, envof"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ArrRefSchema(keyValDef))),
				},
			},
		},
	}

	// 2. envkeys PUT
	pathItems["/envkeys"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutEnvKeys",
					Summary:     "Create or Update value of env kind keys",
					Description: "Create Update value of env kind keys",
					Produces:    []string{"application/json"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ArrRefSchema(envKeyValDef)).
							WithParameterDesc("Update multiple normal env keyvals"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 3. envkey PUT
	pathItems["/envkey"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutEnvKey",
					Summary:     "Create or Update value of a single env kind key",
					Description: "Create or Update value of a single env kind key",
					Produces:    []string{"application/json"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ObjRefSchema(envKeyValDef)).
							WithParameterDesc("Update a single normal env keyval"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 4. envkvs GET & PUT
	pathItems["/envkvs"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "ExportEnvKVS",
					Summary:     "Export all env kind key values",
					Description: "Export all env kind key values",
					Produces:    []string{"application/json"},
					Tags:        []string{envKeyValTag},
					Responses:   openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.FileSchema())),
				},
			},
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "ImportEnvKVS",
					Summary:     "Import given env kind key values",
					Description: "Import given env kind key values",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("formData", "file", "file", "", true, nil).
							WithParameterDesc("key values file"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	//5. envfkey PUT
	pathItems["/envfkey"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutEnvfKey",
					Summary:     "Create or Update a envf kind key value",
					Description: "Create or Update a envf kind key value",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{fileKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "name", "string", "", true, nil).
							WithParameterDesc("key name for the env file"),
						openapi.BuildParam("formData", "file", "file", "", true, nil).
							WithParameterDesc("env file"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 6. key/{fully_qualified_key_name} GET
	pathItems["/key/{fully_qualified_key_name}"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "GetKey",
					Summary:     "Get a keyval with the given key name and kind",
					Description: "Get a keyval with the given key name and kind",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("path", "fully_qualified_key_name", "string", "", true, nil).
							WithParameterDesc("Allowed format: kind/name,  supported kind: env, envf, envo"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ObjRefSchema(keyValDef))),
				},
			},
		},
	}

	// 7. oidc registration
	pathItems["/oidcr"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutOIDCClients",
					Summary:     "Register OAuth2.0 Client",
					Description: "Register OAuth2.0 Client",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{addOnsTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("formData", "file", "file", "", true, nil).
							WithParameterDesc(fmt.Sprintf(`
OAuth2.0 Registration file, accept env key usage, example file %s

There are two steps for the OAuth2.0 client registration, 
The first one is, Register client with the given parameters in the file to oidc provider,
And the second step is, Create OAuth2.0 client-related key & value pairs that come from oidc provider registration response, 
such as client-id, client-secret, redirect-uri, etc. by following the name conventions described below:
1. client-id would be: "\<RegistrationName\>ClientID=****"
2. client-secret would be: "\<RegistrationName\>ClientSecret=****"
3. redirect-uri would be: "\<RegistrationName\>RedirectURI=ValueOfRedirectURI"
4. host, which added for front-end compatibility, would be: "\<RegistrationName\>Host=ValueOfHost"

It is IMPORTANT to know clients.name in the registration file is the RegistrationName in this context, 
which is the key prefix to store the key & value pairs in env store.
`,
								sa.Schema+"://"+filepath.Join(sa.KnownHost, sa.BasePath, "example/oidcr")),
							),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// Create a swagger spec & set the basic infos
	swspec := openapi.NewSpec(sa)

	// Set openapi details
	swspec.Definitions = specDefs
	swspec.Tags = tags
	swspec.Paths = &openspec.Paths{Paths: pathItems}

	// Generate spec file from swagger spec
	if err := openapi.Write(swspec, iw); err != nil {
		return fmt.Errorf("write to file failed: %v", err)
	}

	return nil
}

func AddOnsExample(c *gin.Context) {
	var filename string
	var out string

	typ := strings.TrimPrefix(c.Param("typ"), "/")
	switch typ {
	case "", "oidcr":
		filename = "oidcr-example.yaml"
		out = `
# Oidc registration example file
#
# 
# It is IMPORTANT to know the clients.name is the key prefix to store the key & value pairs in env store
# And the allowed name pattern is "[\-_a-zA-Z0-9]*".
# For example, If the following oidc client was registered via oidcr API:
# 
# clients:
# - name: ssoOAuth2
#   OAuth2Host: http://localhost:5555
#   redirectURIs:
#   - http://localhost:5555/callback
#   allowedAuthTypes:
#   - authorization_code
#   - implicit
#   - client_credentials
#   - password_credentials
#
# These key & value pairs will stored in env store:
# 1. ssoOAuth2ClientID=GeneratedClientID
# 2. ssoOAuth2ClientSecret=GeneratedSecret
# 3. ssoOAuth2RedirectURI=http://localhost:5555/callback
# 4. ssoOAuth2Host=http://localhost:5555
# 
# Configs for oidc issuer, ie, sso
provider-config:
  issuer: ${env:// .ssoIssuer }
  client-id: ${env:// .internalAppClientID }
  client-secret: ${env:// .internalAppClientSecret }
  username: ${env:// .internalAppUsername }
  password: ${env:// .internalAppPassword }
clients:
- name: ssoOAuth2
  redirectURIs:
  - http://localhost:5555/callback
  allowedAuthTypes:
  - authorization_code
  - implicit
  - client_credentials
  - password_credentials
- name: acOAuth2
  redirectURIs:
  - http://localhost:8080/oauth2
  allowedAuthTypes:
  - authorization_code
  - implicit
  - client_credentials
  - password_credentials
- name: configuratorOAuth2
  # OAuth2Host Added for front-end compatibility.
  # because the way of frontend doing oauth redirect is:
  # frontend expose an oauth2 host to DevOps for customizing and use a
  # fix/hardcoded redirect path(prefixed with the oauth2 host), 
  # which is "sso/callback", to serve the oidc redirect URI callback,
  # instead of exposing oauth2 redirect URI option to the DevOps explicitly.
  OAuth2Host: http://localhost
  redirectURIs:
  - http://localhost/sso/callback
  allowedAuthTypes:
  - authorization_code
  - implicit
  - client_credentials
  - password_credentials
`
	default:
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("unsupported typ: %v", typ))
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/yaml")
	c.Header("Content-Length", strconv.Itoa(len(out)))

	if _, err := io.Copy(c.Writer, bytes.NewBufferString(out)); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("write file failed: %v", err))
		return
	}
}
