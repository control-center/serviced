package nfs

import (
	"errors"
)

var etcHostsAllow = "/etc/hosts.allow"
var etcHostsDeny = "/etc/hosts.deny"
var etcFstab = "/etc/fstab"
var etcExports = "/etc/exports"
var exportsDir = "/exports"

var ErrUnimplemented = errors.New("unimplemented")
