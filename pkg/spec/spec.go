package spec

import "io"

type Spec struct {
	Name      string
	Filenames []string

	rds []io.Reader
}
