// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"path"

	"fmt"
	"github.com/Sirupsen/logrus"
)

type ErrTenantMountsInvalid struct {
	volumePath string
	exportPath string
}

func (err ErrTenantMountsInvalid) Error() string {
	return fmt.Sprintf("Tenant mounts are invalid: %s and %s are backed by different devices", err.volumePath, err.exportPath)
}

// Verifies that the mount points are correct. Returns nil if there are no problems; returns
// an error if we had problems getting the backing devices or if the devices don't match.
func (dfs *DistributedFilesystem) VerifyTenantMounts(tenantID string) (error) {
	logger := plog.WithField("tenantid", tenantID)

	vol, err := dfs.disk.Get(tenantID)
	if err != nil {
		logger.WithError(err).Error("Could not get volume for tenant")
		return err
	}

	volumePath := vol.Path()
	exportPath := path.Join(dfs.net.ExportNamePath(), tenantID)

	logger.WithFields(logrus.Fields{
		"volumepath": volumePath,
		"exportPath": exportPath,
	}).Debug("Verifying tenant mounts")

	// Get the volume backing device..
	volumeDevice, err := dfs.net.GetDevice(volumePath)
	if err != nil {
		logger.WithError(err).Error("Unable to get backing device")
		return err
	}

	// Get the export backing device..
	exportDevice, err := dfs.net.GetDevice(exportPath)
	if err != nil {
		logger.WithError(err).Error("Unable to get backing device")
		return err
	}

	// Make sure the backing devices match.
	if volumeDevice != exportDevice {
		return ErrTenantMountsInvalid{volumePath: volumePath, exportPath: exportPath}
	}

	return nil
}