package service

import (
	"fmt"
	"path"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	zkutils "github.com/zenoss/serviced/zzk/utils"
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

// SetVersion implements client.Node
func (node *ServiceStateNode) SetVersion(version interface{}) { node.version = version }

// UpdateService updates a service node if it exists, otherwise it creates it
func UpdateService(conn client.Connection, svc *service.Service) error {
	if svc.ID == "" {
		return fmt.Errorf("service id required")
	}

	var (
		spath = servicepath(svc.ID)
		node  = &ServiceNode{Service: svc}
	)

	if exists, err := zkutils.PathExists(conn, spath); err != nil {
		return err
	} else if !exists {
		return conn.Create(spath, node)
	}
	return conn.Set(spath, node)
}
