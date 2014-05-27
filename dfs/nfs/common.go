package nfs

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
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

	cmd := commandFactory("mount.nfs4", nfsPath, localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s := string(output)
		if !strings.Contains(s, "already mounted") {
			return fmt.Errorf(strings.TrimSpace(s))
		}
	}
	return nil
}
