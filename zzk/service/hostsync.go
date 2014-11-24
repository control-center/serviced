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
package service

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

// HostSyncHandler is the handler for synchronizing local host data
type HostSyncHandler interface {
	// GetHostsByPool looks up all host given the pool ID
	GetHostsByPool(string) ([]host.Host, error)
	// AddUpdateHost adds or updates a host
	AddUpdateHost(*host.Host) error
	// RemoteHost removes an existing host
	RemoveHost(string) error
}

// HostSynchronizer is the synchronizer for Host data
type HostSynchronizer struct {
	handler HostSyncHandler
	poolID  string
}

// NewHostSynchronizer instantiates a new Synchronizer for host data
func NewHostSynchronizer(handler HostSyncHandler, poolID string) *zzk.Synchronizer {
	hSync := &HostSynchronizer{handler, poolID}
	return zzk.NewSynchronizer(hSync)
}

// Allocate implements zzk.SyncHandler
func (l *HostSynchronizer) Allocate() zzk.Node { return &HostNode{} }

// GetConnection implements zzk.SyncHandler
func (l *HostSynchronizer) GetConnection(path string) (client.Connection, error) { return nil, nil }

// GetPath implements zzk.SyncHandler
func (l *HostSynchronizer) GetPath(nodes ...string) string { return hostpath(nodes...) }

// Ready implements zzk.SyncHandler
func (l *HostSynchronizer) Ready() error { return nil }

// Done implements zzk.Done
func (l *HostSynchronizer) Done() {}

// GetAll implements zzk.SyncHandler
func (l *HostSynchronizer) GetAll() ([]zzk.Node, error) {
	hosts, err := l.handler.GetHostsByPool(l.poolID)
	if err != nil {
		return nil, err
	}

	nodes := make([]zzk.Node, len(hosts))
	for i := range hosts {
		nodes[i] = &HostNode{Host: &hosts[i]}
	}
	return nodes, nil
}

// AddUpdate implements zzk.SyncHandler
func (l *HostSynchronizer) AddUpdate(id string, node zzk.Node) (string, error) {
	if hnode, ok := node.(*HostNode); !ok {
		glog.Errorf("Could not extract host node data for %s", id)
		return "", zzk.ErrInvalidType
	} else if host := hnode.Host; host == nil {
		glog.Errorf("Host is nil for %s", id)
		return "", zzk.ErrInvalidType
	} else if err := l.handler.AddUpdateHost(host); err != nil {
		return "", err
	}

	return node.GetID(), nil
}

// Delete implements zzk.SyncHandler
func (l *HostSynchronizer) Delete(id string) error {
	return l.handler.RemoveHost(id)
}
