package spec

import "fmt"

func Migrate(sourcePath string, register Register) error {
	if sourcePath == "" {
		return nil
	}
	if register == nil {
		return fmt.Errorf("invalid register")
	}
	// TODO: implement migrate here
	return nil
}
