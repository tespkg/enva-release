package templates

import (
	"strconv"
	"strings"
	"text/template"

	"meera.tech/kit/strcase"
)

func newTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"split":     strings.Split,
		"title":     strings.Title,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"camel":     strcase.ToCamel,
		"snake":     strcase.ToSnake,
		"lowercame": strcase.ToLowerCamel,
		"kebab":     strcase.ToKebab,
		"inc":       Inc,
	}
}

// Inc plus two number like string
func Inc(base float64, number float64) string {
	n := int64(base) + int64(number)
	return strconv.FormatInt(n, 10)
}
