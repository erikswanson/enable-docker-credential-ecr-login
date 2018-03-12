package dockerconfig

import "fmt"

type LoadError struct {
	Path  string
	Cause error
}

func (err LoadError) Error() string {
	return fmt.Sprintf("Failed to load Docker config file %+v: %+v", err.Path, err.Cause)
}

type SaveError struct {
	Path  string
	Cause error
}

func (err SaveError) Error() string {
	return fmt.Sprintf("Failed to write Docker config file %+v: %+v", err.Path, err.Cause)
}

type InvalidHomeDirectory string

func (err InvalidHomeDirectory) Error() string {
	return fmt.Sprintf("Unable to determine home directory; refusing to use %+v", string(err))
}
