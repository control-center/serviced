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
)

type ServiceSyncHandler interface {
	GetServicesByPool(poolID string) ([]*service.Service, error)
	AddOrUpdateService(service *service.Service) error
	RemoveService(serviceID string) error
}

type ServiceSyncListener struct {
	handler ServiceSyncHandler
	conn    client.Connection
	poolID  string
}

func NewServiceSyncListener(conn client.Connection, handler ServiceSyncHandler, poolID string) *zzk.SyncListener {
	svcSync := &ServiceSyncListener{handler, conn, poolID}
	return zzk.NewSyncListener(conn, svcSync)
}

func (l *ServiceSyncListener) GetPathBasedConnection(path string) (client.Connection, error) {
	return l.conn, nil
}

func (l *ServiceSyncListener) GetPath(nodes ...string) string { return servicepath(nodes...) }

func (l *ServiceSyncListener) GetAll() ([]zzk.Node, error) {
	svcs, err := l.handler.GetServicesByPool(l.poolID)
	if err != nil {
		return nil, err
	}

	nodes := make([]zzk.Node, len(svcs))
	for i, svc := range svcs {
		nodes[i] = &ServiceNode{svc, nil}
	}

	return nodes, nil
}

func (l *ServiceSyncListener) AddOrUpdate(id string, node zzk.Node) error {
	return l.handler.AddOrUpdateService(node.(*ServiceNode).Service)
}

func (l *ServiceSyncListener) Delete(id string) error {
	return l.handler.RemoveService(id)
}
