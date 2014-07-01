package service

import (
	"path"

	"github.com/zenoss/serviced/coordinator/client"
)

const (
	zkHost = "/hosts"
)

func hostpath(nodes ...string) string {
	p := append([]string{zkHost}, nodes...)
	return path.Join(p...)
}

// HostState is the zookeeper node for storing service instance information
// per host
type HostState struct {
	HostID         string
	ServiceID      string
	ServiceStateID string
	DesiredState   int
	version        interface{}
}

// Version inplements client.Node
func (node *HostState) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *HostState) SetVersion(version interface{}) {
	node.version = version
}

func removeInstance(conn client.Connection, hostID, ssID string) error {
	var hs HostState
	if err := conn.Get(hostpath(hostID, ssID), &hs); err != nil {
		return err
	} else if err := conn.Delete(hostpath(hostID, ssID)); err != nil {
		return err
	} else if err := conn.Delete(servicepath(hs.ServiceID, hs.ServiceStateID)); err != nil {
		return err
	}
	return nil
}