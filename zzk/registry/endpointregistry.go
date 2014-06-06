// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
//
// endpointregistry is used for storing a list of application endpoints
// under an endpoint key.
//
// The zookeeper structures is:
//     /endpoints
//
//         /<endpoint key>                      "tenantID_zope"
//             /<hostID_containerID:zope_Inst1>
//                 |--<ApplicationEndpoint>         {tcp/9080, ...}
//             /<hostID_containerID:zope_Inst2>
//                 |--<ApplicationEndpoint>         {tcp/9080, ...}
//
//         /<endpoint key>                      "tenantID_localhost_zenhubPB"
//             /<hostID_containerID:zenhub>
//                 |--<ApplicationEndpoint>         {tcp/8789}
//
//         /<endpoint key>                      "tenantID_localhost_zenhubXMLRpc"
//             /<hostID_containerID:zenhub>
//                 |--<ApplicationEndpoint>         {tcp/8781}
//
//         /<endpoint key>                      "tenantID_zodb_mysql"
//             /<hostID_containerID:mysql>
//                 |--<ApplicationEndpoint>         {tcp/3306}
//
//         /<endpoint key>                      "tenantID_zodb_impact"
//             /<hostID_containerID:impact>
//                 |--<ApplicationEndpoint>         {tcp/8083}

package registry

import (
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/validation"
	"path"
)

const (
	zkEndpoints = "/endpoints"
)

func zkEndpointsPath(nodes ...string) string {
	p := []string{zkEndpoints}
	p = append(p, nodes...)
	return path.Join(p...)
}

// NewEndpointNode returns a new EndpointNode given tenantID, endpointID, hostID, containerID, ApplicationEndpoint
func NewEndpointNode(tenantID, endpointID, hostID, containerID string, endpoint dao.ApplicationEndpoint) EndpointNode {
	return EndpointNode{
		ApplicationEndpoint: endpoint,
		TenantID:            tenantID,
		EndpointID:          endpointID,
		HostID:              hostID,
		ContainerID:         containerID,
	}
}

// EndpointNode is a node for the exported ApplicationEndpoint
type EndpointNode struct {
	dao.ApplicationEndpoint
	TenantID    string
	EndpointID  string
	HostID      string
	ContainerID string
	version     interface{}
}

// Version is an implementation of client.Node
func (v *EndpointNode) Version() interface{} { return v.version }

// SetVersion is an implementation of client.Node
func (v *EndpointNode) SetVersion(version interface{}) { v.version = version }

// EndpointRegistry holds exported ApplicationEndpoint in EndpointNode nodes
type EndpointRegistry struct {
	registryType
}

// CreateEndpointRegistry creates the endpoint registry
func CreateEndpointRegistry(conn client.Connection) (*EndpointRegistry, error) {
	path := zkEndpointsPath()
	if exists, err := conn.Exists(path); err != nil {
		return nil, err
	} else if !exists {
		if err := conn.CreateDir(path); err != nil {
			return nil, err
		}
	}
	return &EndpointRegistry{registryType{zkEndpointsPath}}, nil
}

// TenantEndpointKey generates the key for the application endpoint
func TenantEndpointKey(tenantID, endpointID string) string {
	return tenantID + "_" + endpointID
}

// hostContainerKey generates the key for the container
func hostContainerKey(hostID, containerID string) string {
	return hostID + "_" + containerID
}

// validateEndpointNode validates EndpointNode
func validateEndpointNode(node EndpointNode) error {
	verr := validation.NewValidationError()

	verr.Add(validation.NotEmpty("ServiceID", node.ServiceID))
	verr.Add(validation.NotEmpty("TenantID", node.TenantID))
	verr.Add(validation.NotEmpty("EndpointID", node.EndpointID))
	verr.Add(validation.NotEmpty("ContainerID", node.ContainerID))
	verr.Add(validation.NotEmpty("HostID", node.HostID))
	if verr.HasError() {
		return verr
	}

	return nil
}

// SetItem sets EndpointNode to the key in registry.  Returns the path of the node in the registry
func (ar *EndpointRegistry) SetItem(conn client.Connection, tenantID, endpointID, hostID, containerID string, node EndpointNode) (string, error) {
	if err := validateEndpointNode(node); err != nil {
		return "", err
	}
	return ar.setItem(conn, TenantEndpointKey(tenantID, endpointID), hostContainerKey(hostID, containerID), &node)
}

// GetItem gets EndpointNode at the given path.
func (ar *EndpointRegistry) GetItem(conn client.Connection, path string) (*EndpointNode, error) {
	var ep EndpointNode
	if err := conn.Get(path, &ep); err != nil {
		glog.Errorf("Could not get EndpointNode at %s: %s", path, err)
		return nil, err
	}
	return &ep, nil
}

// WatchTenantEndpoint watches a tenant endpoint directory
func (ar *EndpointRegistry) WatchTenantEndpoint(conn client.Connection, tenantID, endpointID string,
	processChildren processChildrenFunc, errorHandler WatchError) error {

	key := TenantEndpointKey(tenantID, endpointID)
	return ar.WatchKey(conn, key, processChildren, errorHandler)
}

// WatchApplicationEndpoint watches a specific application endpoint node
func (ar *EndpointRegistry) WatchApplicationEndpoint(conn client.Connection, tenantID, endpointID string,
	processEndpoint func(conn client.Connection, node *EndpointNode), errorHandler WatchError) error {

	processNode := func(conn client.Connection, node client.Node) {
		endpoint := node.(*EndpointNode)
		processEndpoint(conn, endpoint)
	}

	var ep EndpointNode
	path := zkEndpointsPath(TenantEndpointKey(tenantID, endpointID))
	return ar.watchItem(conn, path, &ep, processNode, errorHandler)
}
