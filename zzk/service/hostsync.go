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
)

type HostSyncHandler interface {
	GetHostsByPool(poolID string) ([]*host.Host, error)
	AddOrUpdateHost(host *host.Host) error
	RemoveHost(hostID string) error
}

type HostSyncListener struct {
	handler HostSyncHandler
	conn    client.Connection
	poolID  string
}

func NewHostSyncListener(conn client.Connection, handler HostSyncHandler, poolID string) *zzk.SyncListener {
	hostSync := &HostSyncListener{handler, conn, poolID}
	return zzk.NewSyncListener(conn, hostSync)
}

func (l *HostSyncListener) GetPathBasedConnection(path string) (client.Connection, error) {
	return l.conn, nil
}

func (l *HostSyncListener) GetPath(nodes ...string) string { return servicepath(nodes...) }

func (l *HostSyncListener) GetAll() ([]zzk.Node, error) {
	hosts, err := l.handler.GetHostsByPool(l.poolID)
	if err != nil {
		return nil, err
	}

	nodes := make([]zzk.Node, len(hosts))
	for i, host := range hosts {
		nodes[i] = &HostNode{host, nil}
	}

	return nodes, nil
}

func (l *HostSyncListener) AddOrUpdate(id string, node zzk.Node) error {
	return l.handler.AddOrUpdateHost(node.(*HostNode).Host)
}

func (l *HostSyncListener) Delete(id string) error {
	return l.handler.RemoveHost(id)
}
