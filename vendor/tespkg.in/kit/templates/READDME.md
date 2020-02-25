# templates

templates based on std `text/template` pkg, and used for render a whole directory.

## Example

```go
package main

import (
    "os"
    "path/filepath"
    "strings"
    "text/template"

    "tespkg.in/kit/templates"
)

// Create a replacement in place render.
// Use assetDir for both template inputs dir and outputs dir.
func renderAssets(assetDir string, vars interface{}) error {
	render, err := templates.NewRender(
		assetDir,
		templates.WithTemplatePath(assetDir),
		templates.WithKeepAsItIs([]string{"*.js", "*.js.map", "*.css", "*.css.map"}),
		templates.WithTemplateOptions(
        		func(filename string, tpl *template.Template) *template.Template {
        			if strings.HasSuffix(filename, ".txt") {
        				tpl = tpl.Delims("{{{{", "}}}}")
        			}
        			return tpl
        		},
        ),
    )
	if err != nil {
		return err
	}

	err = filepath.Walk(render.TemplatePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				name, err := filepath.Rel(render.TemplatePath, path)
				if err != nil {
					return err
				}
				return render.ExecuteTemplate(vars, true, true, name)
			}
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}
```