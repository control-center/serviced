package nfs

import (
	"fmt"
	"os/exec"
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
	return nil
}
