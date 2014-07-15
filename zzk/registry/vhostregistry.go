// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// vhostregistry is used for storing a list of vhost endpoints under a vhost key.
// The zookeeper structurs is:
//    /vhosts
//      /<vhost key 1>
//         |--<VhostEndpoint>
//         |--<VhostEndpoint>
//      /<vhost key 2>

package registry

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/validation"
	"github.com/zenoss/serviced/zzk/utils"

	"fmt"
	"path"
)

const (
	zkVhosts = "/vhosts"
)

func vhostPath(nodes ...string) string {
	p := []string{zkVhosts}
	p = append(p, nodes...)
	return path.Join(p...)
}

// NewVhostEndpoint creates a new VhostEndpoint
func NewVhostEndpoint(endpointName string, appEndpoint dao.ApplicationEndpoint) VhostEndpoint {
	return VhostEndpoint{ApplicationEndpoint: appEndpoint, EndpointName: endpointName}
}

// VhostEndpoint contains information about a vhost
type VhostEndpoint struct {
	dao.ApplicationEndpoint
	EndpointName string
	version      interface{}
}

// Version is an implementation of client.Node
func (v *VhostEndpoint) Version() interface{} { return v.version }

// SetVersion is an implementation of client.Node
func (v *VhostEndpoint) SetVersion(version interface{}) { v.version = version }

// VhostRegistry is a specific registryType for vhosts
type VhostRegistry struct {
	registryType
}

// VHostRegistry ensures the vhost registry and returns the VhostRegistry type
func VHostRegistry(conn client.Connection) (*VhostRegistry, error) {
	path := vhostPath()
	if exists, err := utils.PathExists(conn, path); err != nil {
		return nil, err
	} else if !exists {
		if err := conn.CreateDir(path); err != nil {
			glog.Errorf("error with CreateDir(%s) %+v", path, err)
			return nil, err
		}
	}
	return &VhostRegistry{registryType{getPath: vhostPath, ephemeral: true}}, nil
}

//SetItem adds or replaces the  VhostEndpoint to the key in registry.  Returns the path of the node in the registry
func (vr *VhostRegistry) SetItem(conn client.Connection, key string, node VhostEndpoint) (string, error) {
	verr := validation.NewValidationError()

	verr.Add(validation.NotEmpty("ServiceID", node.ServiceID))
	verr.Add(validation.NotEmpty("EndpointName", node.EndpointName))
	if verr.HasError() {
		return "", verr
	}

	nodeID := fmt.Sprintf("%s_%s", node.ServiceID, node.EndpointName)
	return vr.setItem(conn, key, nodeID, &node)
}

//GetItem gets  VhostEndpoint at the given path.
func (vr *VhostRegistry) GetItem(conn client.Connection, path string) (*VhostEndpoint, error) {
	var vep VhostEndpoint
	if err := conn.Get(path, &vep); err != nil {
		glog.Infof("Could not get vhost endpoint at %s: %s", path, err)
		return nil, err
	}
	return &vep, nil
}

//WatchVhostEndpoint watch a specific VhostEnpoint
func (vr *VhostRegistry) WatchVhostEndpoint(conn client.Connection, path string, cancel <-chan bool, processVhostEdnpoint func(conn client.Connection,
	node *VhostEndpoint), errorHandler WatchError) error {

	processNode := func(conn client.Connection, node client.Node) {
		vhostEndpoint := node.(*VhostEndpoint)
		processVhostEdnpoint(conn, vhostEndpoint)
	}

	var vep VhostEndpoint
	return vr.watchItem(conn, path, &vep, cancel, processNode, errorHandler)
}
