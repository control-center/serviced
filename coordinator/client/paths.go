
package client 


import (
	"errors"
	"strings"
)


func ValidatePath(path string) error {

	if len(path) == 0 {
		return errors.New("path cannot be empty")
	}
	if strings.HasPrefix(path, "/") {
		return errors.New("path must start with / character")
	}

	if strings.Contains(path, "//") {
		return errors.New("empty node specified")
	}

	if strings.Contains(path, "..") {
		return errors.New("relative paths are not supported")
	}

	return nil
}

func Mkdirs(driver Driver, path string, makeLastNode bool) error {

	if err := ValidatePath(path); err != nil {
		return err
	}

        parts := strings.Split(path, "/")

        subPath := "/"

	lastNodeIndex := len(parts) - 1
        for i, part := range parts {

		if i == lastNodeIndex && !makeLastNode{
			return nil
		}

                subPath += part
		exists, err := driver.Exists(subPath)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err = driver.Create(subPath, []byte{}); err != nil {
			if err == ErrNodeExists {
				continue
			}
			return err
		}
        }
	return nil
}

