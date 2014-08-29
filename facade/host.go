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

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"

	"github.com/zenoss/glog"
)

const (
	beforeHostUpdate = beforeEvent("BeforeHostUpdate")
	afterHostUpdate  = afterEvent("AfterHostUpdate")
	beforeHostAdd    = beforeEvent("BeforeHostAdd")
	afterHostAdd     = afterEvent("AfterHostAdd")
	beforeHostDelete = beforeEvent("BeforeHostDelete")
	afterHostDelete  = afterEvent("AfterHostDelete")
)

//---------------------------------------------------------------------------
// Host CRUD
// AddHost register a host with serviced. Returns an error if host already exists
func (f *Facade) AddHost(ctx datastore.Context, entity *host.Host) (err error) {
	glog.V(2).Infof("Facade.AddHost: %v", entity)
	exists, err := f.GetHost(ctx, entity.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return fmt.Errorf("host already exists: %s", entity.ID)
	}
	// only allow hostid of master if SERVICED_REGISTRY is false
	if !docker.UseRegistry() {
		masterHostID, err := utils.HostID()
		if err != nil {
			return fmt.Errorf("unable to retrieve hostid %s: %s", entity.ID, err)
		}
		if entity.ID != masterHostID {
			return fmt.Errorf("SERVICED_REGISTRY is false and hostid %s does not match master %s", entity.ID, masterHostID)
		}
	}
	// validate Pool exists
	pool, err := f.GetResourcePool(ctx, entity.PoolID)
	if err != nil {
		return fmt.Errorf("error verifying pool exists: %v", err)
	}
	if pool == nil {
		return fmt.Errorf("error creating host, pool %s does not exists", entity.PoolID)
	}

	ec := newEventCtx()
	defer f.afterEvent(afterHostAdd, ec, entity, err)

	if err = f.beforeEvent(beforeHostAdd, ec, entity); err != nil {
		return err
	}

	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now
	if err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity); err != nil {
		return err
	} else if err = zkAPI(f).AddHost(entity); err != nil {
		return err
	}

	return nil
}

// UpdateHost information for a registered host
func (f *Facade) UpdateHost(ctx datastore.Context, entity *host.Host) (err error) {
	glog.V(2).Infof("Facade.UpdateHost: %+v", entity)
	//TODO: make sure pool exists
	ec := newEventCtx()
	defer f.afterEvent(afterHostAdd, ec, entity, err)

	if err = f.beforeEvent(beforeHostAdd, ec, entity); err != nil {
		return err
	}

	entity.UpdatedAt = time.Now()
	if err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity); err != nil {
		return err
	} else if err = zkAPI(f).UpdateHost(entity); err != nil {
		return err
	}

	return nil
}

// RemoveHost removes a Host from serviced
func (f *Facade) RemoveHost(ctx datastore.Context, hostID string) (err error) {
	glog.V(2).Infof("Facade.RemoveHost: %s", hostID)
	ec := newEventCtx()
	defer f.afterEvent(afterHostDelete, ec, hostID, err)
	if err = f.beforeEvent(beforeHostDelete, ec, hostID); err != nil {
		return err
	}

	//assert valid host
	var _host *host.Host
	if _host, err = f.GetHost(ctx, hostID); err != nil {
		return err
	} else if _host == nil {
		return nil
	}

	//grab all services that are address assigned this HostID
	query := []string{fmt.Sprintf("Endpoints.AddressAssignment.HostID:%s", hostID)}
	services, err := f.GetTaggedServices(ctx, query)
	if err != nil {
		glog.Errorf("Failed to grab servies with endpoints assigned to host %s: %s", _host.Name, err)
		return err
	}

	//remove all service endpoint address assignments to this host
	reassign := []string{}
	for i := range services {
		for j := range services[i].Endpoints {
			aa := services[i].Endpoints[j].AddressAssignment
			if aa.HostID == hostID && aa.AssignmentType == commons.STATIC {
				//remove the services address assignment
				if err = f.RemoveAddressAssignment(ctx, aa.ID); err != nil {
					glog.Warningf("Failed to remove service %s:%s address assignment to host %s", services[i].Name, services[i].ID, hostID)
				}
				reassign = append(reassign, services[i].ID)
			}
		}
	}

	//remove host from zookeeper
	if err = zkAPI(f).RemoveHost(_host); err != nil {
		return err
	}

	//remove host from datastore
	if err = f.hostStore.Delete(ctx, host.HostKey(hostID)); err != nil {
		return err
	}

	//update address assignments
	for i := range reassign {
		request := dao.AssignmentRequest{
			ServiceID:      reassign[i],
			IPAddress:      "",
			AutoAssignment: true,
		}
		if err = f.AssignIPs(ctx, request); err != nil {
			glog.Warningf("Failed assigning another ip to service %s: %s", reassign[i], err)
		}
	}

	return nil
}

// GetHost gets a host by id. Returns nil if host not found
func (f *Facade) GetHost(ctx datastore.Context, hostID string) (*host.Host, error) {
	glog.V(2).Infof("Facade.GetHost: id=%s", hostID)
	var value host.Host
	err := f.hostStore.Get(ctx, host.HostKey(hostID), &value)
	glog.V(4).Infof("Facade.GetHost: get error %v", err)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// GetHosts returns a list of all registered hosts
func (f *Facade) GetHosts(ctx datastore.Context) ([]*host.Host, error) {
	return f.hostStore.GetN(ctx, 10000)
}
func (f *Facade) GetActiveHostIDs(ctx datastore.Context) ([]string, error) {
	hostids := []string{}
	pools, err := f.GetResourcePools(ctx)
	if err != nil {
		glog.Errorf("Could not get resource pools: %v", err)
		return nil, err
	}
	for _, p := range pools {
		active, err := zkAPI(f).GetActiveHosts(p.ID)
		if err != nil {
			glog.Errorf("Could not get active host ids for pool: %v", err)
			return nil, err
		}
		hostids = append(hostids, active...)
	}
	return hostids, nil
}

// FindHostsInPool returns a list of all hosts with poolID
func (f *Facade) FindHostsInPool(ctx datastore.Context, poolID string) ([]*host.Host, error) {
	return f.hostStore.FindHostsWithPoolID(ctx, poolID)
}
