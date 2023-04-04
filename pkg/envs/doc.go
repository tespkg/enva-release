package envs

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	openspec "github.com/go-openapi/spec"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/openapi"
)

const (
	keyValTag                = "KeyVal"
	envKeyValTag             = "EnvKeyVal"
	fileKeyValTag            = "FileKeyVal"
	secretTag                = "Secret"
	keyValDef                = "keyVal"
	envKeyValDef             = "envKeyVal"
	generateSecretRequestDef = "generateSecret"
	secretsDef               = "secrets"
)

// GenerateSpec generate openapi spec
func GenerateSpec(iw io.Writer, sa openapi.SpecArgs) error {
	// Generate model definitions
	// 1. Definition for spec.KeyVal & spec.KeyVals model
	// 2. Definition for spec.Header & spec.Spec & spec.Specs model
	// 3. Definition for spec.
	specDefs := map[string]openspec.Schema{
		keyValDef:                openapi.GenerateModel(reflect.ValueOf(kvs.KeyVal{})),
		envKeyValDef:             openapi.GenerateModel(reflect.ValueOf(EnvKeyVal{})),
		generateSecretRequestDef: openapi.GenerateModel(reflect.ValueOf(SecretRequest{})),
		secretsDef:               openapi.GenerateModel(reflect.ValueOf(Secrets{})),
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
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("Get keyvals from the given namespace"),
						openapi.BuildParam("query", "kind", "string", "", false, nil).
							WithParameterDesc("Get keyvals by kind, supported kinds: env, envf, envo, envk"),
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
					Summary:     "Create or Update env kind keys",
					Description: "Create or Update `env kind` keys",
					Produces:    []string{"application/json"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("update normal env keyvals into the given namespace"),
						openapi.BuildParam("query", "json", "bool", "", false, false).
							WithParameterDesc("specify if the content of the given key is json that need to trim indent"),
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
					Summary:     "Create or Update single env kind key",
					Description: "Create or Update single `env kind` key",
					Produces:    []string{"application/json"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("Update a single normal env keyval into the given namespace"),
						openapi.BuildParam("query", "json", "bool", "", false, false).
							WithParameterDesc("specify if the content of the given key is json that need to trim indent"),
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
					Description: "Export all `env kind` key values",
					Produces:    []string{"application/json"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("export all env kind key values from the given namespace"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.FileSchema())),
				},
			},
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "ImportEnvKVS",
					Summary:     "Import given env kind key values",
					Description: "Import given `env kind` key values",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{envKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("import given env kind key values from the given namespace"),
						openapi.BuildParam("query", "json", "bool", "", false, false).
							WithParameterDesc("specify if the content of the given key is json that need to trim indent"),
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
					Description: "Create or Update a `envf kind` key value",
					Produces:    []string{"application/json"},
					Consumes:    []string{"multipart/form-data"},
					Tags:        []string{fileKeyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("import given envf kind key value into the given namespace"),
						openapi.BuildParam("query", "json", "bool", "", false, false).
							WithParameterDesc("specify if the content of the given key is json that need to trim indent"),
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
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("get a key from the given namespace"),
						openapi.BuildParam("path", "fully_qualified_key_name", "string", "", true, nil).
							WithParameterDesc("Allowed format: kind/name,  supported kind: env, envf, envk"),
						openapi.BuildParam("query", "is_prefix", "bool", "", false, "false").
							WithParameterDesc("return keyvals with the prefix in json"),
						openapi.BuildParam("query", "trim_prefix", "bool", "", false, "false").
							WithParameterDesc("return keyvals with the prefix in json, in which the top-level keys' prefix are been trimmed"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ObjRefSchema(keyValDef))),
				},
			},
		},
	}

	// 7. key PUT
	pathItems["/key"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Put: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PutKey",
					Summary:     "Create or Update a single key",
					Description: "Create or Update a single key, supported kind: env, envf, envk",
					Produces:    []string{"application/json"},
					Tags:        []string{keyValTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("query", "ns", "string", "", false, "kvs").
							WithParameterDesc("Create or Update a single keyval into the given namespace"),
						openapi.BuildParam("query", "json", "bool", "", false, false).
							WithParameterDesc("specify if the content of the given key is json that need to trim indent"),
						openapi.BuildParam("query", "plain", "bool", "", false, false).
							WithParameterDesc("indicate if the give value is encrypted or not. if it is not encrypted, the service will use the builtin suite to encrypt it"),
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ObjRefSchema(keyValDef)).
							WithParameterDesc("Create or Update a single keyval"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(nil)),
				},
			},
		},
	}

	// 7. generate secret post
	pathItems["/secret/generate"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Post: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "GenerateSecret",
					Summary:     "Generate secret",
					Description: "Generate secret with the given plain texts",
					Produces:    []string{"application/json"},
					Tags:        []string{secretTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ObjRefSchema(generateSecretRequestDef)).
							WithParameterDesc("plain texts"),
					},
					Responses: openapi.BuildResp(http.StatusOK, openapi.BuildSuccessResp(openapi.ObjRefSchema(secretsDef))),
				},
			},
		},
	}

	// 8. verify secret post
	pathItems["/secret/verify"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Post: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "VerifySecret",
					Summary:     "Verify secret",
					Description: "Verify secret with the given cipher & plain text pairs",
					Produces:    []string{"application/json"},
					Tags:        []string{secretTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(openapi.ObjRefSchema(secretsDef)).
							WithParameterDesc("cipher & plain text pairs"),
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
