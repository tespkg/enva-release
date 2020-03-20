package openapi

import (
	"reflect"
)

type Field struct {
	Name     string
	Required bool
	rv       reflect.Value
}

type Fields []Field

// typeFields returns a list of fields that JSON should recognize for the given type.
// The algorithm is breadth-first search over the set of plain fields any reachable anonymous structs.
func typeFields(rv reflect.Value) (fields Fields) {
	rv = elementOf(rv)
	typeOfT := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fieldTyp := typeOfT.Field(i)
		isUnexported := fieldTyp.PkgPath != ""
		if fieldTyp.Anonymous {
			t := fieldTyp.Type
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			if isUnexported && t.Kind() != reflect.Struct {
				// Ignore embedded fields of unexported non-struct types.
				continue
			}
			// Do not ignore embedded fields of unexported struct types
			// since they may have exported fields.
			embedFields := typeFields(rv.Field(i))
			fields = append(fields, embedFields...)

			// Ignore the embedded fields itself
			continue
		} else if isUnexported {
			// Ignore unexported non-embedded fields.
			continue
		} else if isStruct(fieldTyp.Type) {
			// Ignore struct fields.
			continue
		}
		tag := fieldTyp.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)
		if !isValidTag(name) {
			name = ""
		}
		if name == "" {
			name = fieldTyp.Name
		}
		required := true
		if opts.Contains("omitempty") {
			required = false
		}
		fields = append(fields, Field{
			Name:     name,
			Required: required,
			rv:       rv.Field(i),
		})
	}
	return
}

func isStruct(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct:
		return true
	case reflect.Ptr:
		eleT := t.Elem()
		return isStruct(eleT)
	default:
		return false
	}
}

func elementOf(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Ptr:
		ele := v.Elem()
		return elementOf(ele)
	case reflect.Slice, reflect.Array:
		eleT := v.Type().Elem()
		return elementOf(reflect.New(eleT))
	default:
		return v
	}
}
