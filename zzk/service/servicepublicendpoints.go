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
	"strconv"
	"strings"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/zenoss/glog"
)

const (
	// ZKServicePublicEndpoints is the zk node name for public endpoint data
	ZKServicePublicEndpoints = "/servicepublicendpoints"
	offPrefix                = ":peOff:" // serviceIDs and endpoints can't have ":"
	onPrefix                 = ":peOn:"  // serviceIDs and endpoints can't have ":"
)

func servicePublicEndpointPath(serviceID, endpointname string, enabled bool, pepType registry.PublicEndpointType) string {
	state := offPrefix
	if enabled {
		state = onPrefix
	}
	p := append([]string{ZKServicePublicEndpoints}, fmt.Sprintf("%s_%s_%s_%d", state, serviceID, endpointname, pepType))
	return path.Join(p...)
}

func servicePublicEndpointKeyPath(key string) string {
	p := append([]string{ZKServicePublicEndpoints}, key)
	return path.Join(p...)
}

// ServicePublicEndpointNode is the zookeeper client Node for public endpoints
type ServicePublicEndpointNode struct {
	ServiceID string
	Name      string
	Enabled   bool
	Type      registry.PublicEndpointType
	version   interface{}
}

// GetID implements zzk.Node
func (node *ServicePublicEndpointNode) GetID() string {
	return fmt.Sprintf("%s_%s", node.ServiceID, node.Name)
}

// Create implements zzk.Node
func (node *ServicePublicEndpointNode) Create(conn client.Connection) error {
	return updateServicePublicEndpoint(conn, node.ServiceID, node.Name, node.Enabled, node.Type)
}

// Update implements zzk.Node
func (node *ServicePublicEndpointNode) Update(conn client.Connection) error {
	return updateServicePublicEndpoint(conn, node.ServiceID, node.Name, node.Enabled, node.Type)
}

// Version implements client.Node
func (node *ServicePublicEndpointNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServicePublicEndpointNode) SetVersion(version interface{}) { node.version = version }

// PublicEndpointKey format is enabled_serviceid_name_type
type PublicEndpointKey string

func (v PublicEndpointKey) hasStatePrefix() bool {
	if strings.HasPrefix(string(v), offPrefix) || strings.HasPrefix(string(v), onPrefix) {
		return true
	}
	return false
}

// IsEnabled returns the enabled state of the node
func (v PublicEndpointKey) IsEnabled() bool {
	if strings.HasPrefix(string(v), offPrefix) {
		return false
	}
	//it either has the on prefix or no prefix. No prefix is enabled for backwards compatability
	return true
}

// ServiceID returns the service id for the node
func (v PublicEndpointKey) ServiceID() string {
	parts := strings.SplitN(string(v), "_", 4)
	if v.hasStatePrefix() {
		return parts[1]
	}
	//no state prefix means the first part is the service ID
	return parts[0]
}

// Name returns the public endpoint name (either vhost or port number)
func (v PublicEndpointKey) Name() string {
	if v.hasStatePrefix() {
		parts := strings.SplitN(string(v), "_", 4)
		return parts[2]
	}
	// no state prefix means the second item is the vhost name
	parts := strings.SplitN(string(v), "_", 3)
	return parts[1]
}

// Type identifies whether this public endpoint is a vhost or port type
func (v PublicEndpointKey) Type() registry.PublicEndpointType {
	if v.hasStatePrefix() {
		parts := strings.SplitN(string(v), "_", 4)
		return parsedType(parts[3])
	}

	// no state prefix means the third item is the node type
	parts := strings.SplitN(string(v), "_", 3)
	return parsedType(parts[2])
}

// parsedType returns the string as a registry.PublicEndpointType.  If this isn't
// a valid uint8 it returns registry.EPTypeVHost
func parsedType(arg string) registry.PublicEndpointType {
	// Validate the port number
	pepType, err := strconv.Atoi(arg)
	if err != nil {
		return registry.EPTypeVHost
	}

	if pepType == int(registry.EPTypePort) {
		return registry.EPTypePort
	}
	return registry.EPTypeVHost

}

func newPublicEndpointKey(serviceID string, epName string, enabled bool, pepType registry.PublicEndpointType) PublicEndpointKey {
	state := offPrefix
	if enabled {
		state = onPrefix
	}
	return PublicEndpointKey(fmt.Sprintf("%s_%s_%s_%d", state, serviceID, epName, pepType))
}

func UpdateServicesPublicEndpoints(conn client.Connection, svcs []service.Service) error {
	for _, svc := range svcs {
		if err := UpdateServicePublicEndpoints(conn, &svc); err != nil {
			glog.Errorf("Error Updating ServicePublicEndpoints for Service %s: %s", svc.Name, err)
			return err
		}
	}
	return nil
}

// UpdateServicePublicEndpoints updates vhosts of a service
func UpdateServicePublicEndpoints(conn client.Connection, svc *service.Service) error {
	glog.V(2).Infof("UpdateServicePublicEndpoints for ID:%s Name:%s", svc.ID, svc.Name)

	// generate map of current public endpoints
	currentpublicendpoints := make(map[PublicEndpointKey]struct{})
	if svcpublicEndpoints, err := conn.Children(ZKServicePublicEndpoints); err == client.ErrNoNode {
		/*
			// do not do this, otherwise, nodes aren't deleted when calling RemoveServiceVhost

			if exists, err := zzk.PathExists(conn, ZKServicePublicEndpoints); err != nil {
				return err
			} else if !exists {
				err := conn.CreateDir(ZKServicePublicEndpoints)
				if err != client.ErrNodeExists && err != nil {
					return err
				}
			}
		*/
	} else if err != nil {
		glog.Errorf("UpdateServicePublicEndpoints unable to retrieve public endpoint children at path %s %s", ZKServicePublicEndpoints, err)
		return err
	} else {
		for _, svcvhost := range svcpublicEndpoints {
			peKey := PublicEndpointKey(svcvhost)
			currentpublicendpoints[peKey] = struct{}{}
		}
	}
	glog.V(2).Infof("  currentpublicendpoints %+v", currentpublicendpoints)

	// generate map of enabled public endpoints in the service
	svcpublicendpoints := make(map[PublicEndpointKey]struct{})
	// Add the VHost entries.
	for _, ep := range svc.GetServiceVHosts() {
		for _, vhost := range ep.VHostList {
			svcpublicendpoints[newPublicEndpointKey(svc.ID, vhost.Name, vhost.Enabled, registry.EPTypeVHost)] = struct{}{}
		}
	}
	// Add the Port entries.
	for _, ep := range svc.GetServicePorts() {
		for _, port := range ep.PortList {
			svcpublicendpoints[newPublicEndpointKey(svc.ID, fmt.Sprintf("%s", port.PortAddr), port.Enabled, registry.EPTypePort)] = struct{}{}
		}
	}
	glog.V(2).Infof("  svcpublicendpoints %+v", svcpublicendpoints)

	// remove public endpoints if current not in svc that match serviceid
	for key := range currentpublicendpoints {
		if key.ServiceID() != svc.ID {
			continue
		}

		if _, ok := svcpublicendpoints[key]; !ok {
			if err := removeServicePublicEndpoint(conn, string(key)); err != nil {
				return err
			}
		}
	}

	// add vhosts from svc not in current
	for sv := range svcpublicendpoints {
		if _, ok := currentpublicendpoints[sv]; !ok {
			if err := updateServicePublicEndpoint(conn, svc.ID, sv.Name(), sv.IsEnabled(), sv.Type()); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateServicePublicEndpoint updates a service vhost node if it exists, otherwise creates it
func updateServicePublicEndpoint(conn client.Connection, serviceID, endpointname string, enabled bool, pepType registry.PublicEndpointType) error {
	glog.V(2).Infof("updateServicePublicEndpoint serviceID:%s vhostname:%s", serviceID, endpointname)
	var node ServicePublicEndpointNode
	spath := servicePublicEndpointPath(serviceID, endpointname, enabled, pepType)

	// For some reason you can't just create the node with the service data
	// already set.  Trust me, I tried.  It was very aggravating.
	if err := conn.Get(spath, &node); err != nil {
		if err := conn.Create(spath, &node); err != nil {
			glog.Errorf("Error trying to create node at %s: %s", spath, err)
		}
	}
	node.ServiceID = serviceID
	node.Name = endpointname
	node.Enabled = enabled
	node.Type = pepType
	glog.V(2).Infof("Adding service public endpoint at path:%s %+v", spath, node)
	return conn.Set(spath, &node)
}

// RemoveServicePublicEndpoints removes the public endpoints (vhost or port) of a service
func RemoveServicePublicEndpoints(conn client.Connection, svc *service.Service) error {
	glog.V(2).Infof("RemoveServicePublicEndpoints for ID:%s Name:%s", svc.ID, svc.Name)

	// generate map of current public endpoints
	if svcpublicendpoints, err := conn.Children(ZKServicePublicEndpoints); err == client.ErrNoNode {
	} else if err != nil {
		glog.Errorf("RemoveServicePublicEndpoints unable to retrieve service endpoint children at path %s %s", ZKServicePublicEndpoints, err)
		return err
	} else {
		glog.V(2).Infof("RemoveServicePublicEndpoints for svc.ID:%s from children:%+v", svc.ID, svcpublicendpoints)
		for _, svcpublicendpoint := range svcpublicendpoints {
			pepkey := PublicEndpointKey(svcpublicendpoint)
			if pepkey.ServiceID() == svc.ID {
				if err := removeServicePublicEndpoint(conn, string(pepkey)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// removeServicePublicEndpoint deletes a service public endpoint (either vhost or port)
func removeServicePublicEndpoint(conn client.Connection, key string) error {
	glog.V(2).Infof("removeServicePublicEndpoint %s", key)
	// Check if the path exists
	spath := servicePublicEndpointKeyPath(key)
	if exists, err := zzk.PathExists(conn, spath); err != nil {
		glog.Errorf("unable to determine whether removal path exists %s %s", spath, err)
		return err
	} else if !exists {
		glog.Errorf("service public endpoint removal path does not exist %s", spath)
		return nil
	}

	// Delete the service vhost
	glog.V(2).Infof("Deleting service public endpoint at path:%s", spath)
	return conn.Delete(spath)
}
