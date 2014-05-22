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

func mkdir(conn client.Connection, dirpath string) error {
	if exists, err := conn.Exists(dirpath); err != nil && err != client.ErrNoNode {
		return err
	} else if exists {
		return nil
	} else if err := mkdir(conn, path.Dir(dirpath)); err != nil {
		return err
	}
	return conn.CreateDir(dirpath)
}