// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
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
//                 |--<ApplicationEndpoint>         {tcp/8081}
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
	"path"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/validation"
	"github.com/control-center/serviced/zzk"
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

// CreateEndpointRegistry creates the endpoint registry and returns the EndpointRegistry type
// This is created in the leader, most other calls will just get that one
func CreateEndpointRegistry(conn client.Connection) (*EndpointRegistry, error) {
	path := zkEndpointsPath()
	if exists, err := zzk.PathExists(conn, path); err != nil {
		return nil, err
	} else if !exists {
		if err := conn.CreateDir(path); err != nil {
			glog.Errorf("error with CreateDir(%s) %+v", path, err)
			return nil, err
		}
	}
	return &EndpointRegistry{registryType{getPath: zkEndpointsPath, ephemeral: true}}, nil
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
func (ar *EndpointRegistry) SetItem(conn client.Connection, node EndpointNode) (string, error) {
	if err := validateEndpointNode(node); err != nil {
		return "", err
	}
	return ar.setItem(conn, TenantEndpointKey(node.TenantID, node.EndpointID), hostContainerKey(node.HostID, node.ContainerID), &node)
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

// RemoveTenantEndpointKey removes a tenant endpoint key from the registry
func (ar *EndpointRegistry) RemoveTenantEndpointKey(conn client.Connection, tenantID, endpointID string) error {
	return ar.removeKey(conn, TenantEndpointKey(tenantID, endpointID))
}

// RemoveItem removes an item from the registry
func (ar *EndpointRegistry) RemoveItem(conn client.Connection, tenantID, endpointID, hostID, containerID string) error {
	return ar.removeItem(conn, TenantEndpointKey(tenantID, endpointID), hostContainerKey(hostID, containerID))
}

// WatchTenantEndpoint watches a tenant endpoint directory
func (ar *EndpointRegistry) WatchTenantEndpoint(conn client.Connection, tenantEndpointKey string,
	processChildren ProcessChildrenFunc, errorHandler WatchError) error {

	//TODO: Deal with cancel channel if this cares
	return ar.WatchKey(conn, tenantEndpointKey, make(<-chan bool), processChildren, errorHandler)
}
