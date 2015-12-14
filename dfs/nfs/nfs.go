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

	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

var nfsServiceName = determineNfsServiceName()
var usrBinService = determineServiceCommand()

var start = startImpl
var reload = reloadImpl
var restart = restartImpl
var stop = stopImpl

func determineServiceCommand() string {
	if utils.Platform == utils.Rhel {
		return "systemctl"
	} else {
		return "/usr/sbin/service"
	}
}

func determineNfsServiceName() string {
	// In RHEL-based releases, the 'nfs-server' service is used
	if utils.Platform == utils.Rhel {
		return "nfs-server"
	} else {
		return "nfs-kernel-server"
	}
}

// reload triggers the kernel to reread its NFS exports.
func reloadImpl() error {
	// FIXME: this does not return the proper exit code to see if nfs is running
	var cmd *exec.Cmd
	if utils.Platform == utils.Rhel {
		cmd = exec.Command(usrBinService, "reload", nfsServiceName)
	} else {
		cmd = exec.Command(usrBinService, nfsServiceName, "reload")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	glog.Infof("reloaded nfs server: %s", string(output))
	return nil
}

func startImpl() error {
	// FIXME: this does not return the proper exit code to see if nfs is running
	var cmd *exec.Cmd
	if utils.Platform == utils.Rhel {
		cmd = exec.Command(usrBinService, "reload-or-restart", nfsServiceName)
	} else {
		cmd = exec.Command(usrBinService, nfsServiceName, "start")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	glog.Infof("started nfs server: %s", string(output))
	return nil
}

func restartImpl() error {
	// FIXME: this does not return the proper exit code to see if nfs is running
	var cmd *exec.Cmd
	if utils.Platform == utils.Rhel {
		cmd = exec.Command(usrBinService, "restart", nfsServiceName)
	} else {
		cmd = exec.Command(usrBinService, nfsServiceName, "restart")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	glog.Infof("restarted nfs server: %s", string(output))
	return nil
}

func stopImpl() error {
	// FIXME: this does not return the proper exit code to see if nfs is running
	var cmd *exec.Cmd
	if utils.Platform == utils.Rhel {
		cmd = exec.Command(usrBinService, "stop", nfsServiceName)
	} else {
		cmd = exec.Command(usrBinService, nfsServiceName, "stop")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	glog.Infof("stopped nfs server: %s", string(output))
	return nil
}
