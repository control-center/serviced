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
//             /<containerID:zope_Instance1>
//                 |--<ApplicationEndpoint>         {tcp/9080, ...}
//             /<containerID:zope_Instance2>
//                 |--<ApplicationEndpoint>         {tcp/9080, ...}
//
//         /<endpoint key>                      "tenantID_localhost_zenhubPB"
//             /<containerID:zenhub>
//                 |--<ApplicationEndpoint>         {tcp/8789}
//
//         /<endpoint key>                      "tenantID_localhost_zenhubXMLRpc"
//             /<containerID:zenhub>
//                 |--<ApplicationEndpoint>         {tcp/8781}
//
//         /<endpoint key>                      "tenantID_zodb_mysql"
//             /<containerID:mysql>
//                 |--<ApplicationEndpoint>         {tcp/3306}
//
//         /<endpoint key>                      "tenantID_zodb_impact"
//             /<containerID:impact>
//                 |--<ApplicationEndpoint>         {tcp/8083}

package registry

import (
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"

	"github.com/zenoss/glog"
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

// EndpointNode is a node for the exported endpoint endpoint

type EndpointNode struct {
	dao.ApplicationEndpoint
	tenantID    string
	endpointID  string
	containerID string
	version     interface{}
}

// Version is an implementation of client.Node
func (v *EndpointNode) Version() interface{} { return v.version }

// SetVersion is an implementation of client.Node
func (v *EndpointNode) SetVersion(version interface{}) { v.version = version }

// NewEndpointNode returns a new EndpointNode given ApplicationEndpoint, tenantID, endpointID, containerID
func NewEndpointNode(endpoint *dao.ApplicationEndpoint, tenantID, endpointID, containerID string) *EndpointNode {
	en := EndpointNode{
		*endpoint,
		tenantID,
		endpointID,
		containerID,
		"1.0",
	}
	glog.Info("NewEndpointNode: %+v", en)
	return &en
}

// EndpointRegistry holds exported ApplicationEndpoint in EndpointNode nodes
type EndpointRegistry struct {
	registryType
}

// CreateEndpointRegistry creates the endpoint registry
func CreateEndpointRegistry(conn client.Connection) (*EndpointRegistry, error) {
	path := zkEndpointsPath()
	if err := conn.CreateDir(path); err != nil && err != client.ErrNodeExists {
		glog.Errorf("Could not create EndpointRegistry at %s: %s", path, err)
		return nil, err
	}

	return &EndpointRegistry{registryType{zkEndpointsPath}}, nil
}

// appKey generates the key for the application endpoint
func appKey(tenantID, endpointID string) string {
	return tenantID + "_" + endpointID
}

// AddItem adds EndpointNode to the key in registry.  Returns the path of the node in the registry
func (ar *EndpointRegistry) AddItem(conn client.Connection, tenantID, endpointID, containerID string, node *EndpointNode) (string, error) {
	return ar.addItem(conn, appKey(tenantID, endpointID), containerID, node)
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

// WatchApplicationEndpoint watches a specific EndpointNode
func (ar *EndpointRegistry) WatchApplicationEndpoint(conn client.Connection, path string,
	processEndpoint func(conn client.Connection, node *EndpointNode), errorHandler WatchError) error {

	processNode := func(conn client.Connection, node client.Node) {
		endpoint := node.(*EndpointNode)
		processEndpoint(conn, endpoint)
	}

	var ep EndpointNode
	return ar.watchItem(conn, path, &ep, processNode, errorHandler)
}
