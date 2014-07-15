package service

import (
	"github.com/zenoss/serviced/coordinator/client"
)

const (
	zkScheduler = "/scheduler"
)

type PoolLeader struct {
	HostID  string
	version interface{}
}

func (node *PoolLeader) Version() interface{}           { return node.version }
func (node *PoolLeader) SetVersion(version interface{}) { node.version = version }

func NewLeader(conn client.Connection, hostID string) client.Leader {
	return conn.NewLeader(zkScheduler, &PoolLeader{HostID: hostID})
}