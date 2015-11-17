// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volume

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zenoss/glog"
)

func DetectDriverType(root string) (DriverType, error) {
	// Check to see if the directory even exists. If not, no driver has been initialized.
	glog.V(2).Infof("Detecting driver type under %s", root)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			glog.V(2).Infof("Root does not exist; no driver has been initialized")
			return "", ErrDriverNotInit
		}
		return "", err
	}
	for _, drivertype := range []DriverType{DriverTypeBtrFS, DriverTypeRsync, DriverTypeDeviceMapper} {
		dirname := filepath.Join(root, fmt.Sprintf(".%s", drivertype))
		flagfile := FlagFilePath(dirname)
		if fi, err := os.Stat(flagfile); !os.IsNotExist(err) && fi != nil {
			glog.V(2).Infof("Found %s file; returning %s", dirname, drivertype)
			return drivertype, nil
		}
	}
	return "", ErrDriverNotInit
}
