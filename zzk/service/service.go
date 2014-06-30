package service

import (
	"path"

	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
)

const (
	zkService = "/services"
)

func servicepath(nodes ...string) string {
	p := append([]string{zkService}, nodes...)
	return path.Join(p...)
}

// ServiceNode is the zookeeper client Node for services
type ServiceNode struct {
	Service *service.Service
	version interface{}
}

// Version implements client.Node
func (node *ServiceNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServiceNode) SetVersion(version interface{}) { node.version = version }

// ServiceStateNode is the zookeeper client node for service states
type ServiceStateNode struct {
	ServiceState *servicestate.ServiceState
	version      interface{}
}

// Version implements client.Node
func (node *ServiceStateNode) Version() interface{} { return node.version }

//SetVersion implements client.Node
func (node *ServiceStateNode) SetVersion(version interface{}) { node.version = version }