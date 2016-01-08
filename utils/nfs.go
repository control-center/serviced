package utils

import (
	"fmt"
	"os/exec"

	"github.com/zenoss/glog"
)

func IsNFSMountStale(mountpoint string) bool {
	// See http://stackoverflow.com/questions/17612004/linux-shell-script-how-to-detect-nfs-mount-point-or-the-server-is-dead
	// for explanation of the following command.
	if err := exec.Command("/bin/bash", "-c", fmt.Sprintf("read -t1 < <(stat -t '%s' 2>&-)", mountpoint)).Run(); err != nil {
		if err.Error() == "wait: no child processes" {
			glog.V(2).Infof("Distributed storage check hit probably spurious ECHILD. Ignoring.")
			return false
		}
		status, iscode := GetExitStatus(err)
		if iscode {
			if status == 142 {
				// EREMDEV; read timed out, wait for NFS to come back.
				glog.Infof("Distributed storage temporarily unavailable (EREMDEV). Waiting for it to return.")
				return false
			}
			if status == 10 {
				// ECHILD: No child processes, not sure what causes this, but appears to be spurious.
				glog.V(2).Infof("Distributed storage check hit probably spurious ECHILD. Ignoring.")
				return false
			}
		}
		glog.Errorf("Mount point %s check had error (%d, %s); considering stale.", mountpoint, status, err)
		return true
	}
	return false
}
