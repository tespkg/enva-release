package templates

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/h2non/filetype"
	"github.com/pkg/errors"
)

// TBuf is to hold the executed templates.
type TBuf struct {
	Name   string
	Rename string
	Buf    *bytes.Buffer

	FlushFile bool
}

// Render is to hold the files and generated files related to templates
type Render struct {
	// Path is the output path, as derived from Out.
	Path string
	// TemplatePath is the path to use the user supplied templates .
	TemplatePath string
	// Asset is the static asset handler when TemplatePath is empty
	Asset func(name string) ([]byte, error)

	// KeepAsItIs files, accept filepath and wildcard.
	keepAsItIs []string

	// exclude folder/files
	excludedDirs []string

	// generated is the data generated from templates
	generated map[string]TBuf
	// templateSet is the set of templates to use for generating data.
	templateSet *TemplateSet

	// templateHandler is to custom the template you want to do
	templateOptions []TemplateOptions

	// distName is to change dist name
	distName func(nowName string) string
}

// TemplateOptions is to custom the template you want to do
type TemplateOptions func(filename string, tpl *template.Template) *template.Template

// RenderOption is to render custom options
type RenderOption func(render *Render)

// WithTemplatePath source template path
func WithTemplatePath(tp string) RenderOption {
	return func(r *Render) {
		r.TemplatePath = tp
	}
}

// WithAsset asset func
func WithAsset(f func(name string) ([]byte, error)) RenderOption {
	return func(r *Render) {
		r.Asset = f
	}
}

// WithKeepAsItIs keep the files
func WithKeepAsItIs(files []string) RenderOption {
	return func(r *Render) {
		r.keepAsItIs = files
	}
}

// WithExcludedDirs excluded the dirs
func WithExcludedDirs(dirs []string) RenderOption {
	return func(r *Render) {
		r.excludedDirs = dirs
	}
}

// WithTemplateOptions with template options
func WithTemplateOptions(h ...TemplateOptions) RenderOption {
	return func(r *Render) {
		r.templateOptions = h
	}
}

// WithDistName this can change the dist name to write
func WithDistName(f func(nowName string) string) RenderOption {
	return func(r *Render) {
		r.distName = f
	}
}

// TemplateLoader loads templates from the specified name.
func (r *Render) TemplateLoader(name string) ([]byte, error) {
	if r.TemplatePath == "" {
		if r.Asset == nil {
			return nil, errors.New("invalid template asset")
		}
		return r.Asset(name)
	}
	return ioutil.ReadFile(path.Join(r.TemplatePath, name))
}

// TemplateSet retrieves the created template set.
func (r *Render) TemplateSet() *TemplateSet {
	if r.templateSet == nil {
		r.templateSet = &TemplateSet{
			funcs: newTemplateFuncs(),
			l:     r.TemplateLoader,
		}
	}

	return r.templateSet
}

// NewRender returns a NewRender instance with all operators applied on it except templateSet attribute.
func NewRender(outPath string, Options ...RenderOption) (*Render, error) {
	var r = Render{}
	r.Path = outPath

	for _, o := range Options {
		o(&r)
	}

	if r.Path == "" {
		return nil, errors.New("invalid path info")
	}

	if r.TemplatePath != "" {
		if !IsDirExists(r.TemplatePath) {
			return nil, errors.Errorf("template:%s not exists", r.TemplatePath)
		}
	}
	// stat outPath to determine if outPath exists
	if err := MkdirIfNotExist(r.Path); err != nil {
		return nil, errors.Wrap(err, "invalid out path")
	}

	r.generated = make(map[string]TBuf)

	if r.templateOptions == nil {
		r.templateOptions = make([]TemplateOptions, 0)
	}

	return &r, nil
}

// ExecuteTemplate loads and parses the supplied template with name and
// executes it with obj as the context.
// the name should be a file name, either has suffix with ".tpl" or the original name
// naming a optional func to rename the dist name
func (r *Render) ExecuteTemplate(obj interface{}, flushFile bool, withoutTplSuffix bool, names ...string) error {
	var err error

	var genernatedTBuf []TBuf
	for _, name := range names {
		if r.shouldExclude(name) {
			continue
		}
		// create store
		v := TBuf{
			Name:      strings.TrimSuffix(name, ".tpl"),
			Buf:       new(bytes.Buffer),
			FlushFile: flushFile,
		}

		templateName := name

		if withoutTplSuffix {
			v.Name = name
			templateName = name
		}

		keepAsItIs := false
		for _, a := range r.keepAsItIs {
			if Match(a, templateName) {
				keepAsItIs = true
				break
			}
		}

		if r.distName != nil {
			v.Rename = r.distName(templateName)
		}

		// execute template
		err = r.TemplateSet().Execute(v.Buf, templateName, keepAsItIs, obj, r.templateOptions...)
		if err != nil {
			return err
		}

		r.generated[name] = v

		genernatedTBuf = append(genernatedTBuf, v)
	}
	if err := r.writeFiles(genernatedTBuf); err != nil {
		return errors.Wrap(err, "exe tpl: write files failed")
	}

	return nil
}

// GetBufData get the buf bytes data from the memory if the data didn't flush to file.
func (r *Render) GetBufData(name string) (*bytes.Buffer, error) {
	b, ok := r.generated[name]
	if !ok {
		return nil, errors.Errorf("try to get %s's buf data failed, not existed", name)
	}

	if b.FlushFile {
		return nil, errors.New("since the data has been flush to file, the buf was reset")
	}

	return b.Buf, nil
}

func (r *Render) shouldExclude(name string) bool {
	for _, d := range r.excludedDirs {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}

// writeFiles writes the generated definitions.
func (r *Render) writeFiles(generatedTBufs []TBuf) error {
	var err error

	var files []*os.File
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()

	// loop, writing in order
	for _, t := range generatedTBufs {
		if !t.FlushFile {
			continue
		}

		var f *os.File

		// check if generated template is only whitespace/empty
		bufStr := strings.TrimSpace(t.Buf.String())
		if len(bufStr) == 0 {
			continue
		}

		// get file and filename
		f, err = getFile(r, &t)
		if err != nil {
			return err
		}

		if f == nil {
			continue
		}

		files = append(files, f)

		_, err = t.Buf.WriteTo(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// TemplateSet is r set of templates.
type TemplateSet struct {
	funcs template.FuncMap
	l     func(string) ([]byte, error)
}

// Execute executes r specified template in the template set using the supplied
// obj as its parameters and writing the output to w.
func (ts *TemplateSet) Execute(w io.Writer, name string, keepAsItIs bool, obj interface{}, tplOptions ...TemplateOptions) error {
	// attempt to load and parse the template
	buf, err := ts.l(name)
	if err != nil {
		return err
	}

	// Skip image files
	if filetype.IsImage(buf) || keepAsItIs {
		_, err = w.Write(buf)
		return err
	}

	tpl := template.New(name)

	if len(tplOptions) != 0 {
		for _, opt := range tplOptions {
			tpl = opt(name, tpl)
		}
	}

	// parse template
	tpl, err = tpl.Funcs(ts.funcs).Parse(string(buf))
	if err != nil {
		return err
	}

	return tpl.Execute(w, obj)
}

// getFile builds the filepath from the TBuf information, and retrieves the
// file from files. If the built filename is not already defined, then it calls
// the os.OpenFile with the correct parameters depending on the state of args.
func getFile(r *Render, t *TBuf) (*os.File, error) {
	var f *os.File
	var err error

	// determine filename
	filename := path.Join(r.Path, t.Name)
	if t.Rename != "" {
		filename = path.Join(r.Path, t.Rename)
	}

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// default open mode
	mode := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	// open file
	f, err = os.OpenFile(filename, mode, 0666)
	if err != nil {
		return nil, err
	}

	return f, nil
}
