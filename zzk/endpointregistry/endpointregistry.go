package endpointregistry

import (
	"path"

	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
)

const (
	zkEndpointsRoot = "/endpointsobsolete"
)

// EndpointsPath returns the path to container endpoint
func EndpointsPath(nodes ...string) string {
	p := []string{zkEndpointsRoot}
	p = append(p, nodes...)
	return path.Join(p...)
}

func appKey(tenantID, endpointID string) string {
	return tenantID + "_" + endpointID
}

// EndpointRegistry is a registry for endpoints
type EndpointRegistry struct {
	client *coordclient.Client
}

// NewEndpointRegistry creates a new EndpointRegistry
func NewEndpointRegistry(client *coordclient.Client) *EndpointRegistry {
	return &EndpointRegistry{
		client: client,
	}
}

// EndpointNode is a node for the container endpoint
type EndpointNode struct {
	Endpoint *dao.ApplicationEndpoint
	version  interface{}
}

// Version returns the EndpointNode version
func (e *EndpointNode) Version() interface{} { return e.version }

// SetVersion sets the EndpointNode version
func (e *EndpointNode) SetVersion(version interface{}) { e.version = version }

// AddEndpoint adds a container endpoint
func (er *EndpointRegistry) AddEndpoint(tenantID, endpointID, containerID string, endpoint *dao.ApplicationEndpoint) error {
	conn, err := er.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return AddEndpoint(conn, tenantID, endpointID, containerID, endpoint)
}

// AddEndpoint adds a container endpoint
func AddEndpoint(conn coordclient.Connection, tenantID, endpointID, containerID string, endpoint *dao.ApplicationEndpoint) error {
	glog.V(0).Infof("Adding new registry endpoint %s", endpoint.ServiceID)

	// /endpoints
	//     /tenantID_rabbitmq
	//        /containerID   -> ApplicationEndPoint {ServiceIP, ContainerPort:5672,  HostPort, HostIP, ContainerIP, Protocol, }
	//        /containerID   -> ApplicationEndPoint {ServiceIP, ContainerPort:15672, HostPort, HostIP, ContainerIP, Protocol, }

	// make sure toplevel paths exist
	appPath := EndpointsPath(appKey(tenantID, endpointID))
	paths := []string{zkEndpointsRoot, appPath}
	for _, path := range paths {
		exists, err := conn.Exists(path)
		if err != nil {
			if err == coordclient.ErrNoNode {
				if err := conn.CreateDir(path); err != nil && err != coordclient.ErrNodeExists {
					return err
				}
			}
		}
		if !exists {
			if err := conn.CreateDir(path); err != nil && err != coordclient.ErrNodeExists {
				return err
			}
		}
	}

	// add the endpoint to the root
	en := EndpointNode{Endpoint: endpoint}
	endpointPath := EndpointsPath(appKey(tenantID, endpointID), containerID)
	if err := conn.Create(endpointPath, &en); err != nil {
		glog.Errorf("Unable to create endpoint registry node %s: %v", endpointPath, err)
		return err
	}

	glog.Infof("Successfully created endpoint registry node %s", endpointPath)
	return nil
}

// LoadEndpoint loads a container endpoint
func (er *EndpointRegistry) LoadEndpoint(tenantID, endpointID, containerID string, endpoint *dao.ApplicationEndpoint) error {
	conn, err := er.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return LoadEndpoint(conn, tenantID, endpointID, containerID, endpoint)
}

// LoadEndpoint loads a container endpoint
func LoadEndpoint(conn coordclient.Connection, tenantID, endpointID, containerID string, endpoint *dao.ApplicationEndpoint) error {
	en := EndpointNode{}
	endpointPath := EndpointsPath(appKey(tenantID, endpointID), containerID)
	err := conn.Get(endpointPath, &en)
	if err != nil {
		glog.Errorf("Unable to retrieve endpoint %s: %v", endpointPath, err)
		return err
	}
	*endpoint = *en.Endpoint
	return nil
}

// RemoveEndpoint removes a container endpoint
func (er *EndpointRegistry) RemoveEndpoint(tenantID, endpointID, containerID string) error {
	conn, err := er.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return RemoveEndpoint(conn, tenantID, endpointID, containerID)
}

// RemoveEndpoint removes a container endpoint
func RemoveEndpoint(conn coordclient.Connection, tenantID, endpointID, containerID string) error {
	endpointPath := EndpointsPath(appKey(tenantID, endpointID), containerID)
	if err := conn.Delete(endpointPath); err != nil {
		glog.Errorf("Unable to delete endpoint %s: %v", endpointPath, err)
		return err
	}

	return nil
}