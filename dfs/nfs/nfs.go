// Copyright 2014 The Serviced Authors.
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

package nfs

import (
	"fmt"
	"os/exec"

	"github.com/zenoss/glog"
	"github.com/control-center/serviced/utils"
)

var nfsServiceName = determineNfsServiceName()
var usrBinService = "/usr/sbin/service"

var start = startImpl
var reload = reloadImpl

func determineNfsServiceName() string {
    // In RHEL-based releases, the 'nfs' service is used
    if utils.Platform == utils.Rhel {
        return "nfs"
    } else {
        return "nfs-kernel-server"
    }
}

// reload triggers the kernel to reread its NFS exports.
func reloadImpl() error {
	// FIXME: this does not return the proper exit code to see if nfs is running
	cmd := exec.Command(usrBinService, nfsServiceName, "reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

func startImpl() error {
	// FIXME: this does not return the proper exit code to see if nfs is running
	cmd := exec.Command(usrBinService, nfsServiceName, "start")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	glog.Infof("started nfs server: %s", string(output))
	return nil
}
