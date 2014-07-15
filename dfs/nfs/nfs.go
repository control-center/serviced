// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package nfs

import (
	"fmt"
	"os/exec"

	"github.com/zenoss/glog"
)

var nfsServiceName = "nfs-kernel-server"
var usrBinService = "/usr/sbin/service"

var start = startImpl
var reload = reloadImpl

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
