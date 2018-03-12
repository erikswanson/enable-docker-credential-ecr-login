package dockerconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
)

func ForCurrentUser() (*Updater, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	home := path.Clean(usr.HomeDir)
	if home == "." || home == "/" {
		return nil, InvalidHomeDirectory(home)
	}
	return &Updater{
		Path: path.Join(home, ".docker", "config.json"),
	}, nil
}

type Updater struct {
	Path string
}

func (u Updater) load() (map[string]interface{}, error) {
	f, err := os.Open(u.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	result := make(map[string]interface{})
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&result)
	if err == io.EOF {
		// This occurs only when there's an empty file at the path. That's OK to ignore and overwrite.
		// Other (serious) situations are io.ErrUnexpectedEOF or an error from the json package.
		err = nil
	}
	return result, err
}

func (u Updater) save(document map[string]interface{}) error {
	dir := path.Dir(u.Path)
	base := path.Base(u.Path)
	if base == "/" || base == "." {
		panic("invalid path")
	}
	output, err := ioutil.TempFile(dir, base+".new")
	if err != nil {
		if os.IsNotExist(err) {
			// The directory does not exist.
			err = os.MkdirAll(dir, 0700)
			if err != nil {
				return err
			}
			output, err = ioutil.TempFile(dir, "config.json.new")
			if err != nil {
				return err
			}
		}
	}
	outputName := output.Name()

	defer func() {
		if err != nil {
			output.Close()
			if removeErr := os.Remove(outputName); removeErr != nil {
				if !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Failed to remove temporary file %+v: %+v\n", outputName, removeErr)
				}
			}
		}
	}()

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(document)
	if err != nil {
		return err
	}

	err = output.Close()
	if err != nil {
		return err
	}

	return os.Rename(outputName, u.Path)
}

func (u Updater) EnsureCredHelpers(helper string, registries []string) (bool, error) {
	config, err := u.load()
	if err != nil {
		return false, LoadError{
			Path:  u.Path,
			Cause: err,
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	var credHelpers map[string]interface{}
	if maybeCredHelpers, ok := config["credHelpers"].(map[string]interface{}); ok {
		credHelpers = maybeCredHelpers
	}
	if credHelpers == nil {
		credHelpers = make(map[string]interface{})
	}

	var dirty bool
	for _, registry := range registries {
		var existing string
		if s, ok := credHelpers[registry].(string); ok {
			existing = s
		}
		if existing != "ecr-login" {
			credHelpers[registry] = "ecr-login"
			dirty = true
		}
	}

	if dirty {
		config["credHelpers"] = credHelpers
		err = u.save(config)
		if err != nil {
			return true, SaveError{
				Path:  u.Path,
				Cause: err,
			}
		}
		return true, nil
	}
	return false, nil
}
