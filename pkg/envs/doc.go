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

	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/openapi"
	"tespkg.in/envs/pkg/spec"

	openspec "github.com/go-openapi/spec"
)

const (
	keyValTag  = "keyval"
	addOnsTag  = "add-ons"
	specTag    = "spec"
	specHdrDef = "spechdr"
)

// GenerateSpec generate openapi spec
func GenerateSpec(iw io.Writer, sa openapi.SpecArgs) error {
	// Generate model definitions
	// 1. Definition for spec.KeyVal & spec.KeyVals model
	// 2. Definition for spec.Header & spec.Spec & spec.Specs model
	specDefs := map[string]openspec.Schema{
		keyValTag:  openapi.GenerateModel(reflect.ValueOf(kvs.KeyVal{})),
		specHdrDef: openapi.GenerateModel(reflect.ValueOf(spec.Header{})),
		specTag:    openapi.GenerateModel(reflect.ValueOf(spec.Spec{})),
	}

	tags := []openspec.Tag{
		{
			TagProps: openspec.TagProps{
				Name: keyValTag,
			},
		},
		{
			TagProps: openspec.TagProps{
				Name: addOnsTag,
			},
		},
		{
			TagProps: openspec.TagProps{
				Name: specTag,
			},
		},
	}

	pathItems := make(map[string]openspec.PathItem)
	// 1. keys GET & PUT
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
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ArrRefSchema(keyValTag))),
				},
			},
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutKeys",
					Summary:     "Update value of keys",
					Description: "Update value of keys",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ArrRefSchema(keyValTag)).
							WithParameterDesc("Update multiple keyvals"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}
	// 2. key PUT
	pathItems["/key"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutKey",
					Summary:     "Update value of a single key",
					Description: "Update value of a single key",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ObjRefSchema(keyValTag)).
							WithParameterDesc("Update a single keyval"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 3. kvs GET & PUT
	pathItems["/kvs"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "ExportKVS",
					Summary:     "Export all env/envo kind key values",
					Description: "Export all env/envo kind key values",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Responses:   openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.FileSchema())),
				},
			},
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "ImportKVS",
					Summary:     "Import given key values",
					Description: "Import given key values",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("formData", "filename.1", "file", "", true, nil).
							WithParameterDesc("key values file"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 4. key/{fully_qualified_key_name} GET
	pathItems["/key/{fully_qualified_key_name}"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "GetKey",
					Summary:     "Get a keyval with the given key name",
					Description: "Get a keyval with the given key name",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("path", "fully_qualified_key_name", "string", "", true, nil).
							WithParameterDesc("Allowed format: kind/name,  supported kind: env, envf, envo, envof"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ObjRefSchema(keyValTag))),
				},
			},
		},
	}

	// 5. oidc registration
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
						openapi.BuildParam("formData", "filename.1", "file", "", true, nil).
							WithParameterDesc(fmt.Sprintf(`OAuth2.0 Registration file, accept env key usage, example file %s`,
								sa.Schema+"://"+filepath.Join(sa.KnownHost, sa.BasePath, "example/oidcr")),
							),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 6. specs GET
	pathItems["/specs"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "GetSpecs",
					Summary:     "Get service/application specs headers",
					Description: "Get service/application specs headers",
					Produces:    []string{"application/json"},
					Tags:        []string{specTag},
					Responses:   openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ArrRefSchema(specHdrDef))),
				},
			},
		},
	}
	// 7. spec/{name} GET & PUT
	pathItems["/spec/{name}"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Get: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "GetSpec",
					Summary:     "Get a service/application spec details",
					Description: "Get a service/application spec details",
					Produces:    []string{"application/json"},
					Tags:        []string{specTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("path", "name", "string", "", true, nil).
							WithParameterDesc("service/application spec name"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ObjRefSchema(specTag))),
				},
			},
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutSpec",
					Summary:     "Create or reset service/application spec",
					Description: "Create or reset service/application spec",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{specTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("path", "name", "string", "", true, nil).
							WithParameterDesc("Service/application spec name"),
						openapi.BuildParam("formData", "filename.1", "file", "", true, nil).
							WithParameterDesc("First file to upload, please change filename key in the request if needed"),
						openapi.BuildParam("formData", "filename.2", "file", "", false, nil).
							WithParameterDesc("Second file to upload, please change filename key in the request if needed"),
						openapi.BuildParam("formData", "filename...", "file", "", false, nil).
							WithParameterDesc("Mth file to upload, please change filename key in the request if needed"),
						openapi.BuildParam("formData", "filename.N", "file", "", false, nil).
							WithParameterDesc("Nth file to upload, please change filename key in the request if needed"),
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
  # The way of frontend doing oauth redirect is:
  # They expose an oauth2 host to DevOps for customizing and use a
  # fix/hardcoded redirect path(prefixed with the oauth2 host), which is "sso/callback", to serve the oidc redirect URI callback,
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
