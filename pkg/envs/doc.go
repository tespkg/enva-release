package envs

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/openapi"
	"tespkg.in/envs/pkg/spec"

	openspec "github.com/go-openapi/spec"
)

const (
	keyValTag     = "keyval"
	specTag       = "spec"
	specHdrDef    = "spechdr"
	deploymentTag = "deployment"
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
				Name: specTag,
			},
		},
		{
			TagProps: openspec.TagProps{
				Name: deploymentTag,
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

	// 3. key/{fully_qualified_key_name} GET
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
	// 4. specs GET
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
	// 5. spec/{name} GET & PUT
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

	// 6. deployment/{name} POST
	pathItems["/deployment/{name}"] = openspec.PathItem{
		PathItemProps: openspec.PathItemProps{
			Post: &openspec.Operation{
				OperationProps: openspec.OperationProps{
					ID:          "PostDeployment",
					Summary:     "Start a service/application",
					Description: "Start a service/application",
					Produces:    []string{"application/json"},
					Tags:        []string{deploymentTag},
					Parameters: []openspec.Parameter{
						openapi.BuildParam("path", "name", "string", "", true, nil).
							WithParameterDesc("service/application spec name"),
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
