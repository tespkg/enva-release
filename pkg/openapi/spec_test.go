package openapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/go-openapi/spec"

	"github.com/stretchr/testify/require"
)

type Iris struct {
	SepalLength int      `json:"sepal_length"`
	SepalWidth  int      `json:"sepal_width"`
	PetaLength  int      `json:"peta_length"`
	PetaWidth   int      `json:"peta_width"`
	Category    string   `json:"category"`
	Class       []string `json:"class"`
}

func generateIrisAPIs(iw io.Writer, sa SpecArgs) error {
	irisModel, irisTag := "iris", "iris"
	specDefs := map[string]spec.Schema{
		irisModel: GenerateModel(reflect.ValueOf(Iris{})),
	}
	tags := []spec.Tag{
		{
			TagProps: spec.TagProps{
				Name: irisTag,
			},
		},
	}

	pathItems := make(map[string]spec.PathItem)
	pathItems["/keys"] = spec.PathItem{
		PathItemProps: spec.PathItemProps{
			Get: &spec.Operation{
				OperationProps: spec.OperationProps{
					ID:          "GetIrisSet",
					Summary:     "Get multiple iris with or without filter",
					Description: "Get multiple iris with or without filter",
					Produces:    []string{"application/json"},
					Tags:        []string{irisTag},
					Parameters: []spec.Parameter{
						BuildParam("query", "category", "string", "", false, nil).
							WithParameterDesc("Get Schema by category"),
					},
					Responses: BuildResp(http.StatusOK, BuildSuccessResp(ArrRefSchema(irisModel))),
				},
			},
			Put: &spec.Operation{
				OperationProps: spec.OperationProps{
					ID:          "PutIrisSet",
					Summary:     "Update or create iris",
					Description: "Update or create iris",
					Produces:    []string{"application/json"},
					Tags:        []string{irisTag},
					Parameters: []spec.Parameter{
						BuildParam("body", "body", "", "", true, nil).
							WithNewSchema(ArrRefSchema(irisModel)).
							WithParameterDesc("Update or create multiple iris set"),
					},
					Responses: BuildResp(http.StatusOK, BuildSuccessResp(nil)),
				},
			},
		},
	}

	// Create a swagger spec & set the basic infos
	swspec := NewSpec(sa)

	// Set openapi details
	swspec.Definitions = specDefs
	swspec.Tags = tags
	swspec.Paths = &spec.Paths{Paths: pathItems}

	// Generate spec file from swagger spec
	if err := Write(swspec, iw); err != nil {
		return fmt.Errorf("write to file failed: %v", err)
	}

	return nil
}

func TestGenerateSpec(t *testing.T) {
	buf := bytes.Buffer{}
	err := generateIrisAPIs(&buf, SpecArgs{})
	require.Nil(t, err)
	expected := `
definitions:
  iris:
    properties:
      category:
        type: string
      class:
        items:
          type: string
        type: array
      peta_length:
        type: number
      peta_width:
        type: number
      sepal_length:
        type: number
      sepal_width:
        type: number
    required:
    - sepal_length
    - sepal_width
    - peta_length
    - peta_width
    - category
    - class
info:
  contact: {}
paths:
  /keys:
    get:
      description: Get multiple iris with or without filter
      operationId: GetIrisSet
      parameters:
      - description: Get Schema by category
        in: query
        name: category
        type: string
      produces:
      - application/json
      responses:
        "200":
          description: Success
          schema:
            items:
              $ref: '#/definitions/iris'
            type: array
        "400":
          description: Bad Request
        "401":
          description: Not authenticated
        "500":
          description: Internal server error, please report this
      summary: Get multiple iris with or without filter
      tags:
      - iris
    put:
      description: Update or create iris
      operationId: PutIrisSet
      parameters:
      - description: Update or create multiple iris set
        in: body
        name: body
        required: true
        schema:
          items:
            $ref: '#/definitions/iris'
          type: array
      produces:
      - application/json
      responses:
        "200":
          description: Success
        "400":
          description: Bad Request
        "401":
          description: Not authenticated
        "500":
          description: Internal server error, please report this
      summary: Update or create iris
      tags:
      - iris
produces:
- application/json
schemes:
- ""
swagger: "2.0"
tags:
- name: iris
`
	// Just for showcase, put a \n in front of the expected when initiating, remove it here.
	expected = strings.TrimPrefix(expected, "\n")
	require.Equal(t, expected, buf.String())
}
