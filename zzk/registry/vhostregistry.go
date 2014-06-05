// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
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
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/utils"

	"github.com/zenoss/glog"
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

// Action is the request node for initialized a serviced action on a host
type VhostEndpoint struct {
	dao.ApplicationEndpoint
	version interface{}
}

// Version is an implementation of client.Node
func (v *VhostEndpoint) Version() interface{} { return v.version }

// SetVersion is an implementation of client.Node
func (v *VhostEndpoint) SetVersion(version interface{}) { v.version = version }

type VhostRegistry struct {
	registryType
}

func CreateVHostRegistry(conn client.Connection) (*VhostRegistry, error) {
	path := vhostPath()
	if err := conn.CreateDir(path); err != nil {
		return nil, err
	}
	return &VhostRegistry{registryType{vhostPath}}, nil
}

//AddItem adds VhostEndpoint to the key in registry.  Returns the path of the node in the registry
func (vr *VhostRegistry) AddItem(conn client.Connection, key string, node *VhostEndpoint) (string, error) {
	uuid, err := utils.NewUUID()
	if err != nil {
		return "", err
	}
	return vr.addItem(conn, key, uuid, node)
}

//GetItem gets  VhostEndpoint at the given path.
func (vr *VhostRegistry) GetItem(conn client.Connection, path string) (*VhostEndpoint, error) {
	var vep VhostEndpoint
	if err := conn.Get(path, &vep); err != nil {
		glog.V(1).Infof("Could not get vhost endpoint at %s: %s", path, err)
		return nil, err
	}
	return &vep, nil
}

//GetItem gets  VhostEndpoint at the given path.
func (vr *VhostRegistry) WatchVhostEnpoint(conn client.Connection, path string, processVhostEnpoint func(conn client.Connection,
	node *VhostEndpoint), errorHandler WatchError) error {

	processNode := func(conn client.Connection, node client.Node) {
		vhostEndpoint := node.(*VhostEndpoint)
		processVhostEnpoint(conn, vhostEndpoint)
	}

	var vep VhostEndpoint
	return vr.watchItem(conn, path, &vep, processNode, errorHandler)
}

func (r *registryType) watchItem(conn client.Connection, path string, nodeType client.Node, processNode func(conn client.Connection,
	node client.Node), errorHandler WatchError) error {
	for {
		event, err := conn.GetW(path, nodeType)
		if err != nil {
			glog.Errorf("Could not watch %s: %s", path, err)
			defer errorHandler(path, err)
			return err
		}
		processNode(conn, nodeType)
		//This blocks until a change happens under the key
		<-event
	}
	return nil

}
