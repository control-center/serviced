package scheduler

import (
	"path"

	"github.com/zenoss/serviced/coordinator/client"
)

const (
	zkPool = "/pools"
)

func poolpath(nodes ...string) string {
	p := append([]string{zkPool}, nodes...)
	return path.Join(p...)
}

func AddResourcePool(conn client.Connection, poolID string) error {
	return conn.CreateDir(poolpath(poolID))
}

func RemoveResourcePool(conn client.Connection, poolID string) error {
	return conn.Delete(poolpath(poolID))
}