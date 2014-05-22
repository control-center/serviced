package docker

import (
	"path"

	"github.com/zenoss/serviced/coordinator/client"
)

const (
	zkDocker = "/docker"
	zkAction = "/action"
	zkShell  = "/shell"

	nsInitRoot      = "/var/lib/docker/execdriver/native"
	urandomFilename = "/dev/urandom"
)