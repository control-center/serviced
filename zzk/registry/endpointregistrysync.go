// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package registry

import (
	"strings"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
)

type EndpointSyncListener struct {
	conn     client.Connection
	registry *EndpointRegistry
	key      string
}

func NewEndpointSyncListener(remote, local client.Connection, registry *EndpointRegistry, key string) *zzk.SyncListener {
	epSync := &EndpointSyncListener{local, registry, key}
	return zzk.NewSyncListener(remote, epSync)
}

func (l *EndpointSyncListener) GetPathBasedConnection(path string) (client.Connection, error) {
	return l.conn, nil
}

func (l *EndpointSyncListener) GetPath(nodes ...string) string {
	return zkEndpointsPath(append([]string{l.key}, nodes...)...)
}

func (l *EndpointSyncListener) GetAll() ([]zzk.Node, error) {
	endpoints, err := l.registry.GetItems(l.conn, l.key)
	if err != nil {
		return nil, err
	}
	nodes := make([]zzk.Node, len(endpoints))
	for i, endpoint := range endpoints {
		nodes[i] = endpoint
	}

	return nodes, nil
}

func (l *EndpointSyncListener) AddOrUpdate(id string, node zzk.Node) error {
	_, err := l.registry.SetItem(l.conn, *(node.(*EndpointNode)))
	return err
}

func (l *EndpointSyncListener) Delete(key string) error {
	parts := strings.Split(l.key, "_")
	tenantID, endpointID := parts[0], parts[1]
	parts = strings.Split(key, "_")
	hostID, containerID := parts[0], parts[1]
	return l.registry.RemoveItem(l.conn, tenantID, endpointID, hostID, containerID)
}

type EndpointRegistrySyncListener struct {
	conn     client.Connection
	registry *EndpointRegistry
}

func NewEndpointRegistrySyncListener(remote, local client.Connection, registry *EndpointRegistry) *zzk.SyncListener {
	erSync := &EndpointRegistrySyncListener{local, registry}
	return zzk.NewSyncListener(remote, erSync)
}

func (l *EndpointRegistrySyncListener) GetPathBasedConnection(path string) (client.Connection, error) {
	return l.conn, nil
}

func (l *EndpointRegistrySyncListener) GetPath(nodes ...string) string {
	return zkEndpointsPath(nodes...)
}

func (l *EndpointRegistrySyncListener) GetAll() ([]zzk.Node, error) {
	return []zzk.Node{}, nil
}

func (l *EndpointRegistrySyncListener) AddOrUpdate(key string, node zzk.Node) error {
	_, err := l.registry.EnsureKey(l.conn, key)
	return err
}

func (l *EndpointRegistrySyncListener) Delete(key string) error {
	parts := strings.Split(key, "_")
	tenantID, endpointID := parts[0], parts[1]
	return l.registry.RemoveTenantEndpointKey(l.conn, tenantID, endpointID)
}
