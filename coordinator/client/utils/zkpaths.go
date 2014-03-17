
package utils


import (
	"errors"
	"strings"

	"github.com/samuel/go-zookeeper/zk"
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

func Mkdirs(zookeeper *zk.Conn, path string, makeLastNode bool) error {

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
		exists, _, err := zookeeper.Exists(subPath)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err = zookeeper.Create(subPath, []byte{}, 0, zk.WorldACL(zk.PermAll)); err != nil {
			if err == zk.ErrNodeExists {
				continue
			}
			return err
		}
        }
	return nil
}

