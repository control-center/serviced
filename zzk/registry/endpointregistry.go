// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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

// GetID is an implementation of zzk.Node
func (v *EndpointNode) GetID() string { return hostContainerKey(v.HostID, v.ContainerID) }

// Create is an implementation of zzk.Node
func (v *EndpointNode) Create(conn client.Connection) error { return nil }

// Update is an implementation of zzk.Node
func (v *EndpointNode) Update(conn client.Connection) error { return nil }

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
	key, _ := ar.FindItem(conn, node.TenantID, node.EndpointID, node.HostID, node.ContainerID)
	if key == "" {
		key = hostContainerKey(node.HostID, node.ContainerID)
	}

	return ar.setItem(conn, TenantEndpointKey(node.TenantID, node.EndpointID), key, &node)
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

// GetItems gets all EndpointNodes at the given path
func (ar *EndpointRegistry) GetItems(conn client.Connection, parentPath string) ([]*EndpointNode, error) {
	nodeIDs, err := conn.Children(parentPath)
	if err != nil {
		glog.Errorf("Could not get Endpoints at %s: %s", parentPath, err)
		return nil, err
	}
	items := make([]*EndpointNode, len(nodeIDs))
	for i, nodeID := range nodeIDs {
		ep, err := ar.GetItem(conn, path.Join(parentPath, nodeID))
		if err != nil {
			glog.Errorf("Could not get endpoint at %s: %s", path.Join(parentPath, nodeID), err)
		}
		items[i] = ep
	}

	return items, nil
}

// FindItem returns the key of the EndpointNode
func (ar *EndpointRegistry) FindItem(conn client.Connection, tenantID, endpointID, hostID, containerID string) (string, error) {
	tenantEndpointKey := TenantEndpointKey(tenantID, endpointID)
	hostContainerKey := hostContainerKey(hostID, containerID)

	nodeIDs, err := conn.Children(zkEndpointsPath(tenantEndpointKey))
	if err != nil {
		glog.Errorf("Could not find nodes at key %s: %s", tenantEndpointKey, err)
		return "", err
	}
	for _, nodeID := range nodeIDs {
		if ep, err := ar.GetItem(conn, zkEndpointsPath(tenantEndpointKey, nodeID)); err != nil {
			glog.Errorf("Could not look up node %s: %s", hostContainerKey, err)
			return "", err
		} else if ep.GetID() == hostContainerKey {
			return nodeID, nil
		}
	}

	return "", client.ErrNoNode
}

// RemoveTenantEndpointKey removes a tenant endpoint key from the registry
func (ar *EndpointRegistry) RemoveTenantEndpointKey(conn client.Connection, tenantID, endpointID string) error {
	return ar.removeKey(conn, TenantEndpointKey(tenantID, endpointID))
}

// RemoveItem removes an item from the registry
func (ar *EndpointRegistry) RemoveItem(conn client.Connection, tenantID, endpointID, hostID, containerID string) error {
	key, err := ar.FindItem(conn, tenantID, endpointID, hostID, containerID)
	if err != nil {
		return err
	}
	return ar.removeItem(conn, TenantEndpointKey(tenantID, endpointID), key)
}

// WatchTenantEndpoint watches a tenant endpoint directory
func (ar *EndpointRegistry) WatchTenantEndpoint(conn client.Connection, tenantEndpointKey string,
	processChildren ProcessChildrenFunc, errorHandler WatchError) error {

	//TODO: Deal with cancel channel if this cares
	return ar.WatchKey(conn, tenantEndpointKey, make(<-chan bool), processChildren, errorHandler)
}
