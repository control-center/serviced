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

type ServiceNode struct {
	Service *service.Service
	version interface{}
}

func (node *ServiceNode) Version() interface{}           { return node.version }
func (node *ServiceNode) SetVersion(version interface{}) { node.version = version }

type ServiceStateNode struct {
	ServiceState *servicestate.ServiceState
	version      interface{}
}

func (node *ServiceStateNode) Version() interface{}           { return node.version }
func (node *ServiceStateNode) SetVersion(version interface{}) { node.version = version }