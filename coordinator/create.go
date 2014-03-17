package coordinator

import (
	"path"
	"strings"
)


func (c *Coordinator) Create(pathToNode string, data *[]byte) error {
	parts := strings.Split(path.Join(c.basePath, pathToNode), "/")

	currentPath := "/"
	lastPath := currentPath
	for _, part := range parts {
		currentPath += part

		lastPath = currentPath
	}

}


