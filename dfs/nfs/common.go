
package nfs

import (
	"errors"
)

var etcHostsAllow = "/etc/hosts.allow"
var etcHostsDeny = "/etc/hosts.deny"
var etcFstab = "/etc/fstab"
var etcExports = "/etc/exports"

var ErrUnimplemented = errors.New("unimplemented")

// Version returns the NFS version on this machine. It does
// this by running nfsstat
func Version() (string, err) {
	return "", ErrUnimplemented
}

