package api

import (
	"github.com/zenoss/serviced/commons/layer"
)

// Squash flattens the image (or at least down the to optional downToLayer).
// The resulting image is retagged with newName.
func (a *api) Squash(imageName, downToLayer, newName, tempDir string) (resultImageID string, err error) {

	client, err := a.connectDocker()
	if err != nil {
		return "", err
	}

	return layer.Squash(client, imageName, downToLayer, newName, tempDir)
}
