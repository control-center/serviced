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
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

// ServiceSyncHandler is the handler for local Service data
type ServiceSyncHandler interface {
	// GetServicesByPool gets all the services for a pool ID
	GetServicesByPool(string) ([]service.Service, error)
	// AddUpdateService adds or updates a service
	AddUpdateService(*service.Service) error
	// RemoveService removes an existing service
	RemoveService(string) error
}

// ServiceSynchonizer is the Synchronizer for Service data
type ServiceSynchronizer struct {
	handler ServiceSyncHandler
	poolID  string
}

// NewServiceSynchronizer initializes a new Synchronizer for Service data
func NewServiceSynchronizer(handler ServiceSyncHandler, poolID string) *zzk.Synchronizer {
	sSync := &ServiceSynchronizer{handler, poolID}
	return zzk.NewSynchronizer(sSync)
}

// Allocate implements zzk.SyncHandler
func (l *ServiceSynchronizer) Allocate() zzk.Node { return &ServiceNode{} }

// GetConnection implements zzk.SyncHandler
func (l *ServiceSynchronizer) GetConnection(path string) (client.Connection, error) { return nil, nil }

// GetPath implements zzk.SyncHandler
func (l *ServiceSynchronizer) GetPath(nodes ...string) string { return servicepath(nodes...) }

// Ready implements zzk.SyncHandler
func (l *ServiceSynchronizer) Ready() error { return nil }

// Done implements zzk.SyncHandler
func (l *ServiceSynchronizer) Done() {}

// GetAll implements zzk.SyncHandler
func (l *ServiceSynchronizer) GetAll() ([]zzk.Node, error) {
	services, err := l.handler.GetServicesByPool(l.poolID)
	if err != nil {
		return nil, err
	}

	nodes := make([]zzk.Node, len(services))
	for i, service := range services {
		nodes[i] = &ServiceNode{Service: &service}
	}

	return nodes, nil
}

// AddUpdate implements zzk.SyncHandler
func (l *ServiceSynchronizer) AddUpdate(id string, node zzk.Node) (string, error) {
	if snode, ok := node.(*ServiceNode); !ok {
		glog.Errorf("Could not extract service node data for %s", id)
		return "", zzk.ErrInvalidType
	} else if svc := snode.Service; svc == nil {
		glog.Errorf("Service is nil for %s", id)
		return "", zzk.ErrInvalidType
	} else if err := l.handler.AddUpdateService(svc); err != nil {
		return "", err
	}
	return node.GetID(), nil
}

// Delete implements zzk.SyncHandler
func (l *ServiceSynchronizer) Delete(id string) error {
	return l.handler.RemoveService(id)
}
