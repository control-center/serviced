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
	"strings"
	"time"

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

	if mountInstance, err := getMount(localPath); err == nil {
		if mountInstance.Type == "nfs4" {
			glog.Infof("%s is already mounted", localPath)
			return nil
		}
		return fmt.Errorf("%s not mounted nfs4, %s instead", localPath, mountInstance.Type)
	}

	// try mounting via in interuptable mount
	cmd := commandFactory("mount.nfs4", "-o", "intr", nfsPath, localPath)
	ret := make(chan error)
	go func() {
		output, err := cmd.CombinedOutput()
		s := string(output)
		switch {
		case err != nil && !strings.Contains(err.Error(), "status 32"):
			ret <- fmt.Errorf(strings.TrimSpace(s))
		case strings.Contains(s, "already mounted") || len(strings.TrimSpace(s)) == 0:
			ret <- nil
		default:
			ret <- nil
		}
		close(ret)
	}()
	select {
	case <-time.After(time.Second * 30):
		if execCmd, ok := cmd.(*exec.Cmd); ok {
			execCmd.Process.Kill()
		}
		glog.Errorf("timed out waiting for nfs mount")
		return fmt.Errorf("timeout waiting for nfs mount")
	case err, ok := <-ret:
		if ok {
			return err
		}
	}
	return nil
}

