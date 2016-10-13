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

package facade

import (
	"errors"
	"fmt"
	"time"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/hostkey"
	"github.com/control-center/serviced/domain/service"
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

var (
	ErrHostDoesNotExist = errors.New("facade: host does not exist")
)

//---------------------------------------------------------------------------
// Host CRUD

// AddHost registers a host with serviced. Returns the host's private key.
// Returns an error if host already exists or if the host's IP is a virtual IP.
func (f *Facade) AddHost(ctx datastore.Context, entity *host.Host) ([]byte, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddHost"))
	glog.V(2).Infof("Facade.AddHost: %v", entity)
	if err := f.DFSLock(ctx).LockWithTimeout("add host", userLockTimeout); err != nil {
		glog.Warningf("Cannot add host: %s", err)
		return nil, err
	}
	defer f.DFSLock(ctx).Unlock()
	return f.addHost(ctx, entity)
}

func (f *Facade) addHost(ctx datastore.Context, entity *host.Host) ([]byte, error) {
	exists, err := f.GetHost(ctx, entity.ID)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, fmt.Errorf("host already exists: %s", entity.ID)
	}

	// validate Pool exists
	pool, err := f.GetResourcePool(ctx, entity.PoolID)
	if err != nil {
		return nil, fmt.Errorf("error verifying pool exists: %v", err)
	}
	if pool == nil {
		return nil, fmt.Errorf("error creating host, pool %s does not exists", entity.PoolID)
	}

	// verify that there are no virtual IPs with the given host IP(s)
	for _, ip := range entity.IPs {
		if exists, err := f.HasIP(ctx, pool.ID, ip.IPAddress); err != nil {
			return nil, fmt.Errorf("error verifying ip %s exists: %v", ip.IPAddress, err)
		} else if exists {
			return nil, fmt.Errorf("pool already has a virtual ip %s", ip.IPAddress)
		}
	}

	ec := newEventCtx()
	err = nil
	defer f.afterEvent(afterHostAdd, ec, entity, err)
	if err = f.beforeEvent(beforeHostAdd, ec, entity); err != nil {
		return nil, err
	}

	// Generate and store an RSA key for the host
	delegatePEMBlock, err := f.generateDelegateKey(ctx, entity)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now

	if err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity); err != nil {
		return nil, err
	}
	err = f.zzk.AddHost(entity)

	return delegatePEMBlock, nil
}

// Generate and store an RSA key for the host
func (f *Facade) generateDelegateKey(ctx datastore.Context, entity *host.Host) ([]byte, error) {
	// Generate new key
	delegateHeaders := map[string]string{
		"purpose": "delegate",
		"host_ip": entity.IPAddr,
		"host_id": entity.ID}
	publicPEM, privatePEM, err := auth.GenerateRSAKeyPairPEM(delegateHeaders)
	if err != nil {
		return nil, err
	}

	// Get the master public key
	masterHeaders := map[string]string{"purpose": "master"}
	masterPublicKey, err := auth.GetMasterPublicKey()
	if err != nil {
		return nil, err
	}
	masterPEM, err := auth.PEMFromRSAPublicKey(masterPublicKey, masterHeaders)
	if err != nil {
		return nil, err
	}

	// Store the key
	hostkeyEntity := hostkey.HostKey{PEM: string(publicPEM[:])}
	err = f.hostkeyStore.Put(ctx, entity.ID, &hostkeyEntity)
	if err != nil {
		return nil, err
	}

	// Concatenate and return keys
	delegatePEMBlock := append(privatePEM, masterPEM...)
	return delegatePEMBlock, nil
}

// UpdateHost information for a registered host
func (f *Facade) UpdateHost(ctx datastore.Context, entity *host.Host) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("UpdateHost"))
	glog.V(2).Infof("Facade.UpdateHost: %+v", entity)
	if err := f.DFSLock(ctx).LockWithTimeout("update host", userLockTimeout); err != nil {
		glog.Warningf("Cannot update host: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	// validate the host exists
	if host, err := f.GetHost(ctx, entity.ID); err != nil {
		return err
	} else if host == nil {
		return fmt.Errorf("host does not exist: %s", entity.ID)
	}

	// validate the pool exists
	if pool, err := f.GetResourcePool(ctx, entity.PoolID); err != nil {
		return err
	} else if pool == nil {
		return fmt.Errorf("pool does not exist: %s", entity.PoolID)
	}

	var err error
	ec := newEventCtx()
	defer f.afterEvent(afterHostAdd, ec, entity, err)

	if err = f.beforeEvent(beforeHostAdd, ec, entity); err != nil {
		return err
	}

	entity.UpdatedAt = time.Now()
	if err = f.hostStore.Put(ctx, host.HostKey(entity.ID), entity); err != nil {
		return err
	}

	err = f.zzk.UpdateHost(entity)
	return err
}

// RemoveHost removes a Host from serviced
func (f *Facade) RemoveHost(ctx datastore.Context, hostID string) (err error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("RemoveHost"))
	glog.V(2).Infof("Facade.RemoveHost: %s", hostID)
	if err := f.DFSLock(ctx).LockWithTimeout("remove host", userLockTimeout); err != nil {
		glog.Warningf("Cannot remove host: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	//assert valid host
	var _host *host.Host
	if _host, err = f.GetHost(ctx, hostID); err != nil {
		return err
	} else if _host == nil {
		return fmt.Errorf("HostID %s does not exist", hostID)
	}

	ec := newEventCtx()
	defer f.afterEvent(afterHostDelete, ec, hostID, err)
	if err = f.beforeEvent(beforeHostDelete, ec, hostID); err != nil {
		return err
	}

	//remove host from zookeeper
	if err = f.zzk.RemoveHost(_host); err != nil {
		return err
	}

	// remove host from hostkey datastore
	if err = f.hostkeyStore.Delete(ctx, _host.ID); err != nil {
		return err
	}

	//remove host from datastore
	if err = f.hostStore.Delete(ctx, host.HostKey(hostID)); err != nil {
		return err
	}

	//grab all services that are address assigned the host's IPs
	var services []service.Service
	for _, ip := range _host.IPs {
		query := []string{fmt.Sprintf("Endpoints.AddressAssignment.IPAddr:%s", ip.IPAddress)}
		svcs, err := f.GetTaggedServices(ctx, query)
		if err != nil {
			glog.Errorf("Failed to grab services with endpoints assigned to ip %s on host %s: %s", ip.IPAddress, _host.Name, err)
			return err
		}
		services = append(services, svcs...)
	}

	// update address assignments
	for _, svc := range services {
		request := addressassignment.AssignmentRequest{
			ServiceID:      svc.ID,
			IPAddress:      "",
			AutoAssignment: true,
		}
		if err = f.AssignIPs(ctx, request); err != nil {
			glog.Warningf("Failed assigning another ip to service %s: %s", svc.ID, err)
		}
	}

	return nil
}

// GetHost gets a host by id. Returns nil if host not found
func (f *Facade) GetHost(ctx datastore.Context, hostID string) (*host.Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetHost"))
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

// GetHostKey gets a host key by id. Returns nil if host not found
func (f *Facade) GetHostKey(ctx datastore.Context, hostID string) ([]byte, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetHostKey"))
	glog.V(2).Infof("Facade.GetHostKey: id=%s", hostID)

	if key, err := f.hostkeyStore.Get(ctx, hostID); err != nil {
		return nil, err
	} else {
		return []byte(key.PEM), nil
	}
}

// ResetHostKey generates and returns a host key by id. Returns nil if host not found
func (f *Facade) ResetHostKey(ctx datastore.Context, hostID string) ([]byte, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ResetHostKey"))
	glog.V(2).Infof("Facade.ResetHostKey: id=%s", hostID)

	var value host.Host
	if err := f.hostStore.Get(ctx, host.HostKey(hostID), &value); err != nil {
		return nil, err
	}
	return f.generateDelegateKey(ctx, &value)
}

// SetHostExpiration sets a host's auth token
// expiration time in the HostExpirationRegistry
func (f *Facade) SetHostExpiration(ctx datastore.Context, hostid string, expiration int64) {
	f.hostRegistry.Set(hostid, expiration)
}

// RemoveHostExpiration removes a host from the
// HostExpirationRegistry
func (f *Facade) RemoveHostExpiration(ctx datastore.Context, hostid string) {
	f.hostRegistry.Remove(hostid)
}

// GetHosts returns a list of all registered hosts
func (f *Facade) GetHosts(ctx datastore.Context) ([]host.Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetHosts"))
	return f.hostStore.GetN(ctx, 10000)
}

// GetActiveHostIDs returns a list of active host ids
func (f *Facade) GetActiveHostIDs(ctx datastore.Context) ([]string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetActiveHostIDs"))
	hostids := []string{}
	pools, err := f.GetResourcePools(ctx)
	if err != nil {
		glog.Errorf("Could not get resource pools: %v", err)
		return nil, err
	}
	for _, p := range pools {
		var active []string
		if err := f.zzk.GetActiveHosts(p.ID, &active); err != nil {
			glog.Errorf("Could not get active host ids for pool: %v", err)
			return nil, err
		}
		hostids = append(hostids, active...)
	}
	return hostids, nil
}

// FindHostsInPool returns a list of all hosts with poolID
func (f *Facade) FindHostsInPool(ctx datastore.Context, poolID string) ([]host.Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("FindHostsInPool"))
	return f.hostStore.FindHostsWithPoolID(ctx, poolID)
}

// GetHostByIP returns the host by IP address
func (f *Facade) GetHostByIP(ctx datastore.Context, hostIP string) (*host.Host, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetHostByIP"))
	return f.hostStore.GetHostByIP(ctx, hostIP)
}

// GetReadHosts returns list of all hosts using a minimal representation of a host
func (f *Facade) GetReadHosts(ctx datastore.Context) ([]host.ReadHost, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetReadHosts"))
	hosts, err := f.hostStore.GetN(ctx, 20000)
	if err != nil {
		return nil, err
	}

	return toReadHosts(hosts), nil
}

// FindReadHostsInPool returns list of all hosts for a pool using a minimal representation of a host
func (f *Facade) FindReadHostsInPool(ctx datastore.Context, poolID string) ([]host.ReadHost, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("FindReadHostsInPool"))
	hosts, err := f.hostStore.FindHostsWithPoolID(ctx, poolID)
	if err != nil {
		return nil, err
	}

	return toReadHosts(hosts), nil
}

// GetHostStatuses returns the memory usage and whether or not a host is active
func (f *Facade) GetHostStatuses(ctx datastore.Context, hostIDs []string, since time.Time) ([]host.HostStatus, error) {
	if hostIDs == nil {
		return []host.HostStatus{}, nil
	}

	statuses := []host.HostStatus{}
	for _, id := range hostIDs {
		h, err := f.GetHost(ctx, id)
		if err != nil {
			continue
		}

		status := host.HostStatus{HostID: id, MemoryUsage: service.Usage{}}
		active, err := f.zzk.IsHostActive(h.PoolID, h.ID)
		if err != nil {
			continue
		}
		status.Active = active

		expired, _ := f.hostRegistry.IsExpired(h.ID)
		status.Authenticated = !expired

		instances, err := f.GetHostInstances(ctx, since, id)
		if err != nil {
			continue
		}

		for _, i := range instances {
			status.MemoryUsage.Cur += i.MemoryUsage.Cur
			status.MemoryUsage.Max += i.MemoryUsage.Max
			status.MemoryUsage.Avg += i.MemoryUsage.Avg
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

func toReadHosts(hosts []host.Host) []host.ReadHost {
	readHosts := []host.ReadHost{}
	for _, h := range hosts {
		readHosts = append(readHosts, toReadHost(h))
	}

	return readHosts
}

func toReadHost(h host.Host) host.ReadHost {
	return host.ReadHost{
		ID:            h.ID,
		Name:          h.Name,
		PoolID:        h.PoolID,
		Cores:         h.Cores,
		Memory:        h.Memory,
		RAMLimit:      h.RAMLimit,
		KernelVersion: h.KernelVersion,
		KernelRelease: h.KernelRelease,
		ServiceD: host.ReadServiced{
			Version: h.ServiceD.Version,
			Date:    h.ServiceD.Date,
			Release: h.ServiceD.Release,
		},
		CreatedAt: h.CreatedAt,
		UpdatedAt: h.UpdatedAt,
	}
}
