package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/spec"
)

type SpecArgs struct {
	KnownHost    string
	BasePath     string
	Version      string
	ContactEmail string
	Title        string
	Description  string
	Schema       string
}

// NewSpec create a swagger spec and set some basic information.
func NewSpec(sa SpecArgs) *spec.Swagger {
	return &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:  "2.0",
			Schemes:  []string{sa.Schema},
			Produces: []string{"application/json"},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       sa.Title,
					Description: sa.Description,
					Contact: &spec.ContactInfo{
						ContactInfoProps: spec.ContactInfoProps{
							Email: sa.ContactEmail,
						},
					},
					Version: sa.Version,
				},
			},
			Host:     sa.KnownHost,
			BasePath: sa.BasePath,
		},
	}
}

func GenerateModel(rv reflect.Value) spec.Schema {
	properties := make(map[string]spec.Schema)
	var requiredFields []string
	fields := typeFields(rv)
	for _, field := range fields {
		properties[field.Name] = fieldSchema(field)
		if field.Required {
			requiredFields = append(requiredFields, field.Name)
		}
	}
	return spec.Schema{
		SchemaProps: spec.SchemaProps{
			Properties: properties,
			Required:   requiredFields,
		},
	}
}

func fieldSchema(field Field) spec.Schema {
	return spec.Schema{
		SchemaProps: specTyp(field.rv),
	}
}

func specTyp(rv reflect.Value) spec.SchemaProps {
	schPro := spec.SchemaProps{}
	switch rv.Type().Kind() {
	case reflect.Bool:
		schPro.Type = spec.StringOrArray{"boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schPro.Type = spec.StringOrArray{"number"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		schPro.Type = spec.StringOrArray{"boolean"}
	case reflect.Float32, reflect.Float64:
		schPro.Type = spec.StringOrArray{"number"}
	case reflect.String:
		schPro.Type = spec.StringOrArray{"string"}
	case reflect.Slice, reflect.Array:
		schPro.Type = spec.StringOrArray{"array"}
		schema := fieldSchema(Field{
			Name: rv.Type().Name(),
			rv:   elementOf(reflect.New(rv.Type().Elem())),
		})
		schPro.Items = &spec.SchemaOrArray{
			Schema: &schema,
		}
	case reflect.Interface, reflect.Ptr, reflect.Map:
		panic("unsupported typ")
	}
	return schPro
}

type Param struct {
	spec.Parameter
}

func (p *Param) WithParameterDesc(desc string) spec.Parameter {
	p.Description = desc
	return p.Parameter
}

func (p *Param) WithNewSchema(schema *spec.Schema) *Param {
	p.SimpleSchema = spec.SimpleSchema{}
	p.Schema = schema
	return p
}

func BuildParam(in, name, typ, format string, required bool, defaultValue interface{}, enum ...interface{}) *Param {
	if typ == "array" {
		return &Param{
			Parameter: spec.Parameter{
				ParamProps: spec.ParamProps{
					In:       in,
					Name:     name,
					Required: required,
					Schema:   ArrRefSchema(name),
				},
			},
		}
	}
	return &Param{
		Parameter: spec.Parameter{
			ParamProps: spec.ParamProps{
				In:       in,
				Name:     name,
				Required: required,
			},
			SimpleSchema: spec.SimpleSchema{
				Type:    typ,
				Format:  format,
				Default: defaultValue,
			},
			CommonValidations: spec.CommonValidations{
				Enum: enum,
			},
		},
	}
}

func BuildResp(respPairs ...interface{}) *spec.Responses {
	stResponses := make(map[int]spec.Response)
	for i := 0; i < len(respPairs); i += 2 {
		stResponses[respPairs[i].(int)] = respPairs[i+1].(spec.Response)
	}

	commonResp := map[int]spec.Response{
		http.StatusBadRequest: {
			ResponseProps: spec.ResponseProps{
				Description: "Bad Request",
			},
		},
		http.StatusInternalServerError: {
			ResponseProps: spec.ResponseProps{
				Description: "Internal server error, please report this",
			},
		},
		http.StatusUnauthorized: {
			ResponseProps: spec.ResponseProps{
				Description: "Not authenticated",
			},
		},
	}

	for k, v := range commonResp {
		stResponses[k] = v
	}

	return &spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: stResponses,
		},
	}
}

func BuildSuccessResp(schema *spec.Schema) spec.Response {
	return spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: "Success",
			Schema:      schema,
		},
	}
}

func FileSchema() *spec.Schema {
	return &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"file"},
		},
	}
}

func ArrRefSchema(defName string) *spec.Schema {
	return &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: spec.StringOrArray{"array"},
			Items: &spec.SchemaOrArray{
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%v", defName)),
					},
				},
			},
		},
	}
}

func ObjRefSchema(defName string) *spec.Schema {
	return &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%v", defName)),
		},
	}
}

func Write(swspec *spec.Swagger, output io.Writer) error {
	b, err := marshalToYAMLFormat(swspec)
	if err != nil {
		return err
	}

	if output == nil {
		return fmt.Errorf("invalid output")
	}

	n, err := output.Write(b)
	if err != nil {
		return fmt.Errorf("write spec failed: %v", err)
	}
	if n != len(b) {
		return fmt.Errorf("write failed, expecte: %v actual: %v", len(b), n)
	}
	return nil
}

func marshalToYAMLFormat(swspec *spec.Swagger) ([]byte, error) {
	b, err := json.Marshal(swspec)
	if err != nil {
		return nil, err
	}

	var jsonObj interface{}
	if err := yaml.Unmarshal(b, &jsonObj); err != nil {
		return nil, err
	}

	return yaml.Marshal(jsonObj)
}
