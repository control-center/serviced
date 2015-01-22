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
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/commons/proc"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"
)

var etcHostsAllow = "/etc/hosts.allow"
var etcHostsDeny = "/etc/hosts.deny"
var etcFstab = "/etc/fstab"
var etcExports = "/etc/exports"
var exportsDir = "/exports"
var lookPath = exec.LookPath

const mountNfs4 = "/sbin/mount.nfs4"

// ErrMalformedNFSMountpoint is returned when the nfs mountpoint string is malformed
var ErrMalformedNFSMountpoint = errors.New("malformed nfs mountpoint")

// ErrNfsMountingUnsupported is returned when the mount.nfs4 binary is not found
var ErrNfsMountingUnsupported = errors.New("nfs mounting not supported; install nfs-common")

// exec.Command interface (for mocking)
type commandFactoryT func(string, ...string) command

// locally plugable command interface
var commandFactory = func(name string, args ...string) command {
	return exec.Command(name, args...)
}

// exec.Cmd interface subset we need
type command interface {
	CombinedOutput() ([]byte, error)
}

// Mount attempts to mount the nfsPath to the localPath
func Mount(nfsPath, localPath string) error {

	if _, err := lookPath(mountNfs4); err != nil {
		return ErrNfsMountingUnsupported
	}
	parts := strings.Split(nfsPath, ":")
	if len(parts) != 2 {
		return ErrMalformedNFSMountpoint
	}
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return ErrMalformedNFSMountpoint
	}
	if len(parts[1]) < 2 || !strings.HasPrefix(parts[1], "/") {
		return ErrMalformedNFSMountpoint
	}

	// Is the localpath already mounted?
	mountInfo, mountError := proc.GetNFSVolumeInfo(localPath)
	if mountError == proc.ErrMountPointNotFound {
		// the mountpoint is not found so try to mount
		// try mounting via in interuptable mount
		glog.Infof("%s not mounted; mounting...", localPath)
		cmd := commandFactory("mount.nfs4", "-o", "intr", nfsPath, localPath)
		errC := make(chan error, 1)
		go func() {
			output, err := cmd.CombinedOutput()
			glog.V(1).Infof("Mount %s to %s: %s (%s)", nfsPath, localPath, string(output), err)

			exitCode, ok := utils.GetExitStatus(err)
			if exitCode == 32 || !ok {
				err := fmt.Errorf("%s (%s)", string(output), err)
				glog.Errorf("Could not mount %s to %s: %s", nfsPath, localPath, err)
				errC <- err
			} else {
				errC <- nil
			}
		}()

		select {
		case <-time.After(time.Second * 30):
			if execCmd, ok := cmd.(*exec.Cmd); ok {
				execCmd.Process.Kill()
			}
			glog.Errorf("timed out waiting for nfs mount")
			return fmt.Errorf("timeout waiting for nfs mount")
		case err := <-errC:
			if err != nil {
				return err
			}
		}

		// get the mount point
		mountInfo, mountError = proc.GetNFSVolumeInfo(localPath)
	}

	if mountError != nil {
		// we should have a mountpoint by now or bust
		glog.Errorf("Could not get volume info for %s (%s): %s", localPath, nfsPath, mountError)
		return mountError
	}

	// Validate mount info
	glog.Infof("Mount Info: %+v", mountInfo)
	verr := validation.NewValidationError()
	verr.Add(validation.StringsEqual(nfsPath, mountInfo.RemotePath, ""))
	verr.Add(validation.StringsEqual("v4", mountInfo.Version, fmt.Sprintf("%s not mounted nfs4, %s instead", localPath, mountInfo.Version)))
	verr.Add(func(fsid string) error {
		if fsiduint, err := strconv.ParseUint(fsid, 16, 64); err != nil || fsiduint == 0 {
			return fmt.Errorf("invalid fsid: %s", fsid)
		}
		return nil
	}(mountInfo.FSID))

	if verr.HasError() {
		// the mountpoint is stale or wrong, so unmount
		glog.Warningf("Stale mount point; unmounting...")
		cmd := commandFactory("umount", "-f", localPath)
		output, err := cmd.CombinedOutput()
		glog.Infof("Unmount %s: %s (%s)", localPath, string(output), err)
		return verr
	}

	return nil
}
