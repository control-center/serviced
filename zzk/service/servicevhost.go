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

package service

import (
	"fmt"
	"path"
	"strings"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

const (
	zkServiceVhosts = "/servicevhosts"
)

func servicevhostpath(serviceID, vhost string, enabled bool) string {
	state := "off"
	if enabled {
		state = "on"
	}
	p := append([]string{zkServiceVhosts}, fmt.Sprintf("%s_%s_%s", state, serviceID, vhost))
	return path.Join(p...)
}

// ServiceVhostNode is the zookeeper client Node for service vhosts
type ServiceVhostNode struct {
	ServiceID string
	Vhost     string
	Enabled   bool
	version   interface{}
}

// GetID implements zzk.Node
func (node *ServiceVhostNode) GetID() string {
	return fmt.Sprintf("%s_%s", node.ServiceID, node.Vhost)
}

// Create implements zzk.Node
func (node *ServiceVhostNode) Create(conn client.Connection) error {
	return updateServiceVhost(conn, node.ServiceID, node.Vhost, node.Enabled)
}

// Update implements zzk.Node
func (node *ServiceVhostNode) Update(conn client.Connection) error {
	return updateServiceVhost(conn, node.ServiceID, node.Vhost, node.Enabled)
}

// Version implements client.Node
func (node *ServiceVhostNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServiceVhostNode) SetVersion(version interface{}) { node.version = version }

//format is enabled_serviceid_vhost
type VHostKey string

func (v VHostKey) IsEnabled() bool {
	if v.Enabled() == "on" {
		return true
	}
	return false
}

func (v VHostKey) Enabled() string {
	parts := strings.SplitN(v, "_", 2)
	return parts[0]
}

func (v VHostKey) ServiceID() string {
	parts := strings.SplitN(v, "_", 3)
	return parts[1]
}

func (v VHostKey) VHost() string {
	parts := strings.SplitN(v, "_", 3)
	return parts[2]
}

func newVhostKey(serviceID string, vhostName string, enabled bool) string {
	state := "off"
	if enabled {
		state = "on"
	}
	return fmt.Sprintf("%s_%s_%s", state, serviceID, vhostName)
}

// UpdateServiceVhosts updates vhosts of a service
func UpdateServiceVhosts(conn client.Connection, svc *service.Service) error {
	glog.V(2).Infof("UpdateServiceVhosts for ID:%s Name:%s", svc.ID, svc.Name)

	// generate map of current vhosts
	currentvhosts := make(map[VHostKey]struct{})
	if svcvhosts, err := conn.Children(zkServiceVhosts); err == client.ErrNoNode {
		/*
			// do not do this, otherwise, nodes aren't deleted when calling RemoveServiceVhost

			if exists, err := zzk.PathExists(conn, zkServiceVhosts); err != nil {
				return err
			} else if !exists {
				err := conn.CreateDir(zkServiceVhosts)
				if err != client.ErrNodeExists && err != nil {
					return err
				}
			}
		*/
	} else if err != nil {
		glog.Errorf("UpdateServiceVhosts unable to retrieve vhost children at path %s %s", zkServiceVhosts, err)
		return err
	} else {
		for _, svcvhost := range svcvhosts {
			vhKey := VHostKey(svcvhost)
			currentvhosts[vhKey] = struct{}{}
		}
	}
	glog.V(2).Infof("  currentvhosts %+v", currentvhosts)

	// generate map of enabled vhosts in the service
	svcvhosts := make(map[VHostKey]struct{})
	for _, ep := range svc.GetServiceVHosts() {
		for _, vhost := range ep.VHostList {
			svcvhosts[newVhostKey(svc.ID, vhost.Name, vhost.Enabled)] = vhost
		}
	}
	glog.V(2).Infof("  svcvhosts %+v", svcvhosts)

	// remove vhosts if current not in svc that match serviceid
	for key, _ := range currentvhosts {
		if key.ServiceID() != svc.ID {
			continue
		}

		if _, ok := svcvhosts[key]; !ok {
			if err := removeServiceVhost(conn, svc.ID, key.VHost(), key.IsEnabled()); err != nil {
				return err
			}
		}
	}

	// add vhosts from svc not in current
	for sv, _ := range svcvhosts {
		if _, ok := currentvhosts[sv]; !ok {
			if err := updateServiceVhost(conn, svc.ID, sv.VHost(), sv.IsEnabled()); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateServiceVhost updates a service vhost node if it exists, otherwise creates it
func updateServiceVhost(conn client.Connection, serviceID, vhostname string, enabled bool) error {
	glog.V(2).Infof("UpdateServiceVhost serviceID:%s vhostname:%s", serviceID, vhostname)
	var node ServiceVhostNode
	spath := servicevhostpath(serviceID, vhostname, enabled)

	// For some reason you can't just create the node with the service data
	// already set.  Trust me, I tried.  It was very aggravating.
	if err := conn.Get(spath, &node); err != nil {
		if err := conn.Create(spath, &node); err != nil {
			glog.Errorf("Error trying to create node at %s: %s", spath, err)
		}
	}
	node.ServiceID = serviceID
	node.Vhost = vhostname
	node.Enabled = enabled
	glog.V(2).Infof("Adding service vhost at path:%s %+v", spath, node)
	return conn.Set(spath, &node)
}

// RemoveServiceVhosts removes vhosts of a service
func RemoveServiceVhosts(conn client.Connection, svc *service.Service) error {
	glog.V(2).Infof("RemoveServiceVhosts for ID:%s Name:%s", svc.ID, svc.Name)

	// generate map of current vhosts
	if svcvhosts, err := conn.Children(zkServiceVhosts); err == client.ErrNoNode {
	} else if err != nil {
		glog.Errorf("UpdateServiceVhosts unable to retrieve vhost children at path %s %s", zkServiceVhosts, err)
		return err
	} else {
		glog.V(2).Infof("RemoveServiceVhosts for svc.ID:%s from children:%+v", svc.ID, svcvhosts)
		for _, svcvhost := range svcvhosts {
			vhkey := VHostKey(svcvhost)
			if vhkey.ServiceID() == svc.ID {
				if err := removeServiceVhost(conn, vhkey.ServiceID(), vhkey.VHost(), vhkey.IsEnabled()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// removeServiceVhost deletes a service vhost
func removeServiceVhost(conn client.Connection, serviceID, vhostname string, enabled bool) error {
	glog.V(2).Infof("RemoveServiceVhost serviceID:%s vhostname:%s", serviceID, vhostname)
	// Check if the path exists
	spath := servicevhostpath( serviceID, vhostname, enabled)
	if exists, err := zzk.PathExists(conn, spath); err != nil {
		glog.Errorf("unable to determine whether removal path exists %s %s", spath, err)
		return err
	} else if !exists {
		glog.Errorf("service vhost removal path does not exist %s", spath)
		return nil
	}

	// Delete the service vhost
	glog.V(2).Infof("Deleting service vhost at path:%s", spath)
	return conn.Delete(spath)
}
