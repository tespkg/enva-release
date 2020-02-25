package templates

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// MkdirIfNotExist create dir if not exist
func MkdirIfNotExist(dir string) error {
	// stat dir to determine if dir exists
	fi, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "stat out path failed")
		}
		return os.MkdirAll(dir, 0755)
	}

	if !fi.IsDir() {
		return errors.New("out path already existed, but not dir")
	}

	return nil
}

// IsDirExists validate if a dir is exists or not
func IsDirExists(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return true
	}
	return false
}

// CopyDir copy dir from one to another place
func CopyDir(fromDir, toDir string) error {
	err := filepath.Walk(fromDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				name, err := filepath.Rel(fromDir, path)
				if err != nil {
					return err
				}
				targetDir := filepath.Join(toDir, filepath.Dir(name))
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return err
				}
				b, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				f, err := os.Create(filepath.Join(targetDir, filepath.Base(name)))
				if err != nil {
					return err
				}
				if _, err := f.Write(b); err != nil {
					f.Close()
					return err
				}
				f.Close()
			}
			return nil
		})
	if err != nil {
		return err
	}
	return nil
}
