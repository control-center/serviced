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

package facade

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
	zkregistry "github.com/control-center/serviced/zzk/registry"
	zkhost "github.com/control-center/serviced/zzk/service"
	zkservice "github.com/control-center/serviced/zzk/service"
	zkvirtualip "github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

func getZKAPI(f *Facade) zkfuncs {
	return &zkf{f}
}

var zkAPI func(f *Facade) zkfuncs = getZKAPI

type zkfuncs interface {
	UpdateService(service *service.Service) error
	RemoveService(service *service.Service) error
	WaitService(service *service.Service, state service.DesiredState, cancel <-chan interface{}) error
	GetServiceStates(poolID string, states *[]servicestate.ServiceState, serviceIDs ...string) error
	StopServiceInstance(poolID, hostID, stateID string) error
	CheckRunningVHost(vhostName, serviceID string) error
	AddHost(host *host.Host) error
	UpdateHost(host *host.Host) error
	RemoveHost(host *host.Host) error
	GetActiveHosts(poolID string, hosts *[]string) error
	AddResourcePool(pool *pool.ResourcePool) error
	UpdateResourcePool(pool *pool.ResourcePool) error
	RemoveResourcePool(poolID string) error
	AddVirtualIP(virtualIP *pool.VirtualIP) error
	RemoveVirtualIP(virtualIP *pool.VirtualIP) error
}

type zkf struct {
	f *Facade
}

func (zk *zkf) UpdateService(service *service.Service) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(service.PoolID))
	if err != nil {
		return err
	}

	if err := zkservice.UpdateService(conn, service); err != nil {
		return err
	}

	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}

	return zkservice.UpdateServiceVhosts(rootconn, service)
}

func (zk *zkf) RemoveService(service *service.Service) error {
	// acquire the service lock to prevent that service from being scheduled
	// as it is being deleted
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(service.PoolID))
	if err != nil {
		return err
	}

	// FIXME: this may be a long-running operation, should we institute a timeout?
	mutex := zkservice.ServiceLock(conn)
	mutex.Lock()
	defer mutex.Unlock()
	return zkservice.RemoveService(conn, service.ID)
}

func (zk *zkf) WaitService(service *service.Service, state service.DesiredState, cancel <-chan interface{}) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(service.PoolID))
	if err != nil {
		return err
	}

	return zkservice.WaitService(cancel, conn, service.ID, state)
}

func (zk *zkf) GetServiceStates(poolID string, states *[]servicestate.ServiceState, serviceIDs ...string) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		return err
	}

	*states, err = zkservice.GetServiceStates(conn, serviceIDs...)
	return err
}

func (zk *zkf) StopServiceInstance(poolID, hostID, stateID string) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		return err
	}

	return zkservice.StopServiceInstance(conn, hostID, stateID)
}

func (z *zkf) CheckRunningVHost(vhostName, serviceID string) error {
	rootBasedConnection, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}

	vr, err := zkregistry.VHostRegistry(rootBasedConnection)
	if err != nil {
		glog.Errorf("Error getting vhost registry: %v", err)
		return err
	}

	vhostEphemeralNodes, err := vr.GetVHostKeyChildren(rootBasedConnection, vhostName)
	if err != nil {
		glog.Errorf("GetVHostKeyChildren failed %v: %v", vhostName, err)
		return err
	}

	if len(vhostEphemeralNodes) > 0 {
		if vhost := vhostEphemeralNodes[0]; vhost.ServiceID != serviceID {
			err := fmt.Errorf("virtual host %s is already running under service %s", vhostName, vhost.ServiceID)
			return err
		}
	}

	return nil
}

func (z *zkf) AddHost(host *host.Host) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(host.PoolID))
	if err != nil {
		return err
	}
	return zkhost.AddHost(conn, host)
}

func (z *zkf) UpdateHost(host *host.Host) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(host.PoolID))
	if err != nil {
		return err
	}
	return zkhost.UpdateHost(conn, host)
}

func (z *zkf) RemoveHost(host *host.Host) error {
	// acquire the service lock to prevent services from being scheduled
	// to that pool
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(host.PoolID))
	if err != nil {
		return err
	}

	// FIXME: this may be a long-running operation, should we institute a timeout?
	mutex := zkservice.ServiceLock(conn)
	mutex.Lock()
	defer mutex.Unlock()

	cancel := make(chan interface{})
	go func() {
		defer close(cancel)
		<-time.After(2 * time.Minute)
	}()

	return zkhost.RemoveHost(cancel, conn, host.ID)
}

func (z *zkf) GetActiveHosts(poolID string, hosts *[]string) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		return err
	}
	*hosts, err = zkhost.GetActiveHosts(conn, poolID)
	return err
}

func (z *zkf) AddResourcePool(pool *pool.ResourcePool) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zkservice.AddResourcePool(conn, pool)
}

func (z *zkf) UpdateResourcePool(pool *pool.ResourcePool) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zkservice.UpdateResourcePool(conn, pool)
}

func (z *zkf) RemoveResourcePool(poolID string) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zkservice.RemoveResourcePool(conn, poolID)
}

func (z *zkf) AddVirtualIP(virtualIP *pool.VirtualIP) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(virtualIP.PoolID))
	if err != nil {
		return err
	}
	return zkvirtualip.AddVirtualIP(conn, virtualIP)
}

func (z *zkf) RemoveVirtualIP(virtualIP *pool.VirtualIP) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(virtualIP.PoolID))
	if err != nil {
		return err
	}
	return zkvirtualip.RemoveVirtualIP(conn, virtualIP.IP)
}
