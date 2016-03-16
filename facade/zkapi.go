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

	zkimgregistry "github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk"
	zkregistry "github.com/control-center/serviced/zzk/registry"
	zkhost "github.com/control-center/serviced/zzk/service"
	zkservice "github.com/control-center/serviced/zzk/service"
	zkvirtualip "github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

func getZZK(f *Facade) ZZK {
	return &zkf{f}
}

type zkf struct {
	f *Facade
}

func (zk *zkf) UpdateService(svc *service.Service, setLockOnCreate, setLockOnUpdate bool) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
	if err != nil {
		return err
	}
	if err := zkservice.UpdateService(conn, *svc, setLockOnCreate, setLockOnUpdate); err != nil {
		return err
	}
	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zkservice.UpdateServicePublicEndpoints(rootconn, svc)
}

func (zk *zkf) RemoveService(svc *service.Service) error {
	// acquire the service lock to prevent that service from being scheduled
	// as it is being deleted
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
	if err != nil {
		return err
	}
	// remove the global list of all vhosts deployed
	if rootconn, err := zzk.GetLocalConnection("/"); err != nil {
		return err
	} else if err := zkservice.RemoveServicePublicEndpoints(rootconn, svc); err != nil {
		return err
	}
	return zkservice.RemoveService(conn, svc.ID)
}

func (zk *zkf) WaitService(svc *service.Service, state service.DesiredState, cancel <-chan interface{}) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
	if err != nil {
		return err
	}
	return zkservice.WaitService(cancel, conn, svc.ID, state)
}

func (zk *zkf) GetServiceStates(poolID string, states *[]servicestate.ServiceState, serviceIDs ...string) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		return err
	}
	*states, err = zkservice.GetServiceStates(conn, serviceIDs...)
	return err
}

func (zk *zkf) UpdateServiceState(poolID string, state *servicestate.ServiceState) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		return err
	}
	if err = zkservice.UpdateServiceState(conn, state); err != nil {
		return fmt.Errorf("unable to update service state: %s", err)
	}
	return nil
}

func (zk *zkf) StopServiceInstance(poolID, hostID, stateID string) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		return err
	}
	return zkservice.StopServiceInstance(conn, hostID, stateID)
}

func (z *zkf) CheckRunningPublicEndpoint(publicendpoint zkregistry.PublicEndpointKey, serviceID string) error {
	rootBasedConnection, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	per, err := zkregistry.PublicEndpointRegistry(rootBasedConnection)
	if err != nil {
		glog.Errorf("Error getting public endpoint registry: %v", err)
		return err
	}
	publicEndpointEphemeralNodes, err := per.GetPublicEndpointKeyChildren(rootBasedConnection, publicendpoint)
	if err != nil {
		glog.Errorf("GetPublicEndpointKeyChildren failed %v: %v", publicendpoint, err)
		return err
	}
	if len(publicEndpointEphemeralNodes) > 0 {
		if publicEndpoint := publicEndpointEphemeralNodes[0]; publicEndpoint.ServiceID != serviceID {
			err := fmt.Errorf("public end point %s is already running under service %s", publicendpoint, publicEndpoint.ServiceID)
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
	locker, err := zkservice.ServiceLock(conn)
	if err != nil {
		glog.Errorf("Could not initialize service lock: %s", err)
		return err
	}
	if err := locker.Lock(); err != nil {
		glog.Errorf("Could not disable service scheduling for pool %s: %s", host.PoolID, err)
		return err
	}
	defer locker.Unlock()
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
	*hosts, err = zkhost.GetActiveHosts(conn)
	return err
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

func (z *zkf) GetRegistryImage(id string) (*registry.Image, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return nil, err
	}
	return zkimgregistry.GetRegistryImage(conn, id)
}

func (z *zkf) SetRegistryImage(image *registry.Image) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}

	return zkimgregistry.SetRegistryImage(conn, *image)
}

func (z *zkf) DeleteRegistryImage(id string) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zkimgregistry.DeleteRegistryImage(conn, id)
}

func (z *zkf) DeleteRegistryLibrary(tenantID string) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zkimgregistry.DeleteRegistryImage(conn, tenantID)
}

func (z *zkf) LockServices(svcs []service.Service) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	nodes := make([]zkservice.ServiceLockNode, len(svcs))
	for i, svc := range svcs {
		nodes[i] = zkservice.ServiceLockNode{
			PoolID:    svc.PoolID,
			ServiceID: svc.ID,
		}
	}
	return zkservice.LockServices(conn, nodes)
}

func (z *zkf) UnlockServices(svcs []service.Service) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	nodes := make([]zkservice.ServiceLockNode, len(svcs))
	for i, svc := range svcs {
		nodes[i] = zkservice.ServiceLockNode{
			PoolID:    svc.PoolID,
			ServiceID: svc.ID,
		}
	}
	return zkservice.UnlockServices(conn, nodes)
}

// Get a list of exported endpoints for the specified service from the Zookeeper namespace
func (zk *zkf) GetServiceEndpoints(tenantID, serviceID string, result *[]applicationendpoint.ApplicationEndpoint) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("Could get connection to zookeeper: %s", err)
		return err
	}

	endpointRegisty, err := zkregistry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Error getting endpoint registry: %s", err)
		return err
	}

	serviceEndpoints, err := endpointRegisty.GetServiceEndpoints(conn, tenantID, serviceID)
	if err != nil {
		glog.Errorf("Error getting endpoints: %s", err)
		return err
	}

	for _, endpoint := range serviceEndpoints {
		*result = append(*result, endpoint.ApplicationEndpoint)
	}
	return nil
}
