// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
