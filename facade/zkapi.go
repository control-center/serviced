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
	"errors"
	"fmt"
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
	zkimgregistry "github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	zkd "github.com/control-center/serviced/zzk/docker"
	zkr "github.com/control-center/serviced/zzk/registry"
	zks "github.com/control-center/serviced/zzk/service"
	zkvirtualip "github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

func getZZK() ZZK {
	return &zkf{NewServiceRegistryCache()}
}

type zkf struct {
	svcRegistry  *serviceRegistryCache
}

// UpdateService updates the service object and exposed public endpoints that
// are synced in zookeeper.
// TODO: we may want to combine these calls into a single transaction
func (zk *zkf) UpdateService(tenantID string, svc *service.Service, setLockOnCreate, setLockOnUpdate bool) error {
	logger := plog.WithFields(log.Fields{
		"tenantid":    tenantID,
		"serviceid":   svc.ID,
		"servicename": svc.Name,
		"poolid":      svc.PoolID,
	})

	if err := zk.SyncServiceRegistry(tenantID, svc); err != nil {
		logger.WithError(err).Debug("Could not sync public endpoints in zookeeper")
		return err
	}

	// get the pool-based connection to update the service
	poolconn, err := zzk.GetLocalConnection(path.Join("/pools", svc.PoolID))
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a pool-based connection to update the service in zookeeper")
		return err
	}

	if err := zks.UpdateService(poolconn, *svc, setLockOnCreate, setLockOnUpdate); err != nil {
		logger.WithError(err).Debug("Could not update the service in zookeeper")
		return err
	}
	logger.Debug("Updated the service in zookeeper")
	return nil
}

// RemoveService removes a service object from zookeeper
func (zk *zkf) RemoveService(poolID, serviceID string) error {
	logger := plog.WithFields(log.Fields{
		"poolid":    poolID,
		"serviceid": serviceID,
	})

	// get the root-based connection to delete the service
	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a root-based connection to delete the service in zookeeper")
		return err
	}

	if err := zks.RemoveService(rootconn, poolID, serviceID); err != nil {
		logger.WithError(err).Debug("Could not delete the service in zookeeper")
		return err
	}
	logger.Debug("Deleted the service in zookeeper")
	return nil
}

// RemoveServiceEndpoints removes all public endpoints of a service from
// zookeeper.
func (zk *zkf) RemoveServiceEndpoints(serviceID string) error {
	logger := plog.WithField("serviceid", serviceID)

	// get the root-based connection to delete the service endpoints
	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a root-based connection to delete the service's public endpoints in zookeeper")
		return err
	}

	zk.svcRegistry.Lock()
	defer zk.svcRegistry.Unlock()
	if zk.svcRegistry.registry == nil {
		err = zk.buildServiceRegistryCache(rootconn)
		if err != nil {
			logger.WithError(err).Debug("Could not build cache of the service registry")
			return err
		}
	}

	// delete the public endpoints for this service
	syncRequest := zkr.ServiceRegistrySyncRequest{
		ServiceID:       serviceID,
		PortsToPublish:  make(map[zkr.PublicPortKey]zkr.PublicPort),
		PortsToDelete:   []zkr.PublicPortKey{},
		VHostsToPublish: make(map[zkr.VHostKey]zkr.VHost),
		VHostsToDelete:  []zkr.VHostKey{},
	}

	serviceRegistry := zk.svcRegistry.GetRegistryForService(serviceID)
	for key, _ := range serviceRegistry.PublicPorts {
		syncRequest.PortsToDelete = append(syncRequest.PortsToDelete, key)
	}
	for key, _ := range serviceRegistry.VHosts {
		syncRequest.VHostsToDelete = append(syncRequest.VHostsToDelete, key)
	}

	// sync the registry
	if err := zkr.SyncServiceRegistry(rootconn, syncRequest); err != nil {
		logger.WithError(err).Debug("Could not delete the service's public endpoints in zookeeper")
		return err
	}

	// Update our local cache now that we know that ZK was updated successfully
	zk.svcRegistry.UpdateRegistry(serviceID, syncRequest.PortsToPublish, syncRequest.VHostsToPublish)
	logger.Debug("Deleted the service's public endpoints in zookeeper")
	return nil
}

// RemoveTenantExports removes all the exported endpoints for a given tenant
// from zookeeper.
func (zk *zkf) RemoveTenantExports(tenantID string) error {
	logger := plog.WithField("tenantid", tenantID)

	// get the root-based connection to delete the service endpoints
	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a root-based connection to delete the tenant's exported endpoints in zookeeper")
		return err
	}

	// delete the exports
	if err := zkr.DeleteExports(rootconn, tenantID); err != nil {
		logger.WithError(err).Debug("Could not delete the tenant's exported endpoints in zookeeper")
		return err
	}
	logger.Debug("Deleted the tenant's exported endpoints in zookeeper")
	return nil
}

// WaitService waits for all instances of a service to achieve a uniform state
// by monitoring zookeeper.
func (zk *zkf) WaitService(svc *service.Service, state service.DesiredState, cancel <-chan interface{}) error {
	logger := plog.WithFields(log.Fields{
		"serviceid":    svc.ID,
		"servicename":  svc.Name,
		"desiredstate": state,
		"poolid":       svc.PoolID,
	})

	// get the root-based connection to delete the service endpoints
	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a root-based connection to monitor the service's instances")
		return err
	}

	var checkCount func(int) bool
	var checkState func(*zks.State, bool) bool

	// set up the check calls
	switch state {
	case service.SVCStop:
		checkCount = func(count int) bool {
			return count <= svc.Instances
		}
		checkState = func(s *zks.State, exists bool) bool {
			return !exists || (s.DesiredState == service.SVCStop && s.Terminated.After(s.Started))
		}
	case service.SVCRun:
		checkCount = func(count int) bool {
			return count == svc.Instances
		}
		checkState = func(s *zks.State, exists bool) bool {
			return exists && s.DesiredState != service.SVCStop && s.Started.After(s.Terminated)
		}
	case service.SVCPause:
		checkCount = func(count int) bool {
			return count <= svc.Instances
		}
		checkState = func(s *zks.State, exists bool) bool {
			if exists {
				if s.DesiredState != service.SVCRun {
					return s.Paused || s.Terminated.After(s.Started)
				} else {
					return false
				}
			} else {
				return true
			}
		}
	default:
		return errors.New("invalid state")
	}

	// do wait
	errC := make(chan error)
	stop := make(chan struct{})
	go func() {
		errC <- zks.WaitService(stop, rootconn, svc.PoolID, svc.ID, checkCount, checkState)
	}()

	select {
	case err := <-errC:
		if err != nil {
			logger.WithError(err).Debug("Could not monitor the service's states in zookeeper")
			return err
		}
		return err
	case <-cancel:
		close(stop)
		return <-errC
	}
}

// GetPublicPort returns the service id and application using the public endpoint
func (z *zkf) GetPublicPort(portAddress string) (string, string, error) {
	logger := plog.WithField("portaddress", portAddress)

	// get the root-based connection to check the public port
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a root-based connection to look up a public endpoint port")
		return "", "", err
	}

	// look for the public port
	key := zkr.PublicPortKey{
		HostID:      "master",
		PortAddress: portAddress,
	}
	serviceID, application, err := zkr.GetPublicPort(conn, key)
	if err != nil {
		logger.WithError(err).Debug("Could not look up public endpoint port")
		return "", "", err
	}

	logger.Debug("Searched for public endpoint port")
	return serviceID, application, err
}

// GetVHost returns the service id using the vhost
func (z *zkf) GetVHost(subdomain string) (string, string, error) {
	logger := plog.WithField("subdomain", subdomain)

	// get the root-based connection to check the public port
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire root-based connection to look up a public endpoint vhost")
		return "", "", err
	}

	// look for the vhost
	key := zkr.VHostKey{
		HostID:    "master",
		Subdomain: subdomain,
	}
	serviceID, application, err := zkr.GetVHost(conn, key)
	if err != nil {
		logger.WithError(err).Debug("Could not look up public endpoint vhost")
		return "", "", err
	}

	logger.Debug("Searched for public endpoint vhost")
	return serviceID, application, err
}

func (z *zkf) AddHost(host *host.Host) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(host.PoolID))
	if err != nil {
		return err
	}
	return zks.AddHost(conn, *host)
}

func (z *zkf) UpdateHost(host *host.Host) error {
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(host.PoolID))
	if err != nil {
		return err
	}
	return zks.UpdateHost(conn, *host)
}

func (z *zkf) RemoveHost(host *host.Host) error {
	// acquire the service lock to prevent services from being scheduled
	// to that pool
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(host.PoolID))
	if err != nil {
		return err
	}
	locker, err := zks.ServiceLock(conn)
	if err != nil {
		glog.Errorf("Could not initialize service lock: %s", err)
		return err
	}
	if err := locker.Lock(); err != nil {
		glog.Errorf("Could not disable service scheduling for pool %s: %s", host.PoolID, err)
		return err
	}
	defer locker.Unlock()
	cancel := make(chan struct{})
	go func() {
		defer close(cancel)
		<-time.After(2 * time.Minute)
	}()
	return zks.RemoveHost(cancel, conn, "", host.ID)
}

func (z *zkf) GetActiveHosts(poolID string, hosts *[]string) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	*hosts, err = zks.GetCurrentHosts(conn, poolID)
	return err
}

func (z *zkf) UpdateResourcePool(pool *pool.ResourcePool) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zks.UpdateResourcePool(conn, *pool)
}

func (z *zkf) RemoveResourcePool(poolID string) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	return zks.RemoveResourcePool(conn, poolID)
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

func (z *zkf) GetVirtualIPHostID(poolID, ip string) (string, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return "", err
	}
	return zkvirtualip.GetHostID(conn, poolID, ip)
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
	nodes := make([]zks.ServiceLockNode, len(svcs))
	for i, svc := range svcs {
		nodes[i] = zks.ServiceLockNode{
			PoolID:    svc.PoolID,
			ServiceID: svc.ID,
		}
	}
	return zks.LockServices(conn, nodes)
}

func (z *zkf) UnlockServices(svcs []service.Service) error {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}
	nodes := make([]zks.ServiceLockNode, len(svcs))
	for i, svc := range svcs {
		nodes[i] = zks.ServiceLockNode{
			PoolID:    svc.PoolID,
			ServiceID: svc.ID,
		}
	}
	return zks.UnlockServices(conn, nodes)
}

// GetServiceStates returns all running instances for a service
func (zk *zkf) GetServiceStates(poolID, serviceID string) ([]zks.State, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("Could not get connection to zookeeper: %s", err)
		return nil, err
	}

	return zks.GetServiceStates(conn, poolID, serviceID)
}

// GetHostStates returns all running instances for a host
func (zk *zkf) GetHostStates(poolID, hostID string) ([]zks.State, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("Could not get connection to zookeeper: %s", err)
		return nil, err
	}

	return zks.GetHostStates(conn, poolID, hostID)
}

// GetServiceState returns the state of a service from its service and instance
// id.
func (zk *zkf) GetServiceState(poolID, serviceID string, instanceID int) (*zks.State, error) {
	logger := plog.WithFields(log.Fields{
		"poolid":     poolID,
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	// get the root-based connection to find the service instance
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire root-based connection")
		return nil, err
	}

	// get the state host id
	hostID, err := zks.GetServiceStateHostID(conn, poolID, serviceID, instanceID)
	if err != nil {
		logger.WithError(err).Debug("Could not get host id for state")
		return nil, err
	}

	logger = logger.WithField("hostid", hostID)
	logger.Debug("Found service state on host")

	// set up the request
	req := zks.StateRequest{
		PoolID:     poolID,
		HostID:     hostID,
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}

	state, err := zks.GetState(conn, req)
	if err != nil {
		logger.WithError(err).Debug("Could not get state information")
		return nil, err
	}

	logger.Debug("Loaded state information")
	return state, nil
}

// StopServiceInstance2 stops an instance of a service
func (zk *zkf) StopServiceInstance(poolID, serviceID string, instanceID int) error {
	logger := plog.WithFields(log.Fields{
		"poolid":     poolID,
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	// get the root-based connection to stop the service instance
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire root-based connection")
		return err
	}

	// get the state host id
	hostID, err := zks.GetServiceStateHostID(conn, poolID, serviceID, instanceID)
	if err != nil {
		logger.WithError(err).Debug("Could not get host id for state")
		return err
	}

	logger = logger.WithField("hostid", hostID)

	// check if the host is online
	isOnline, err := zks.IsHostOnline(conn, poolID, hostID)
	if err != nil {
		logger.WithError(err).Debug("Could not check the online status of the host")
		return err
	}

	// set up the request
	req := zks.StateRequest{
		PoolID:     poolID,
		HostID:     hostID,
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}

	// if the host is online, schedule the service to stop, otherwise delete the
	// service state.
	if isOnline {
		if err := zks.UpdateState(conn, req, func(s *zks.State) bool {
			if s.DesiredState != service.SVCStop {
				s.DesiredState = service.SVCStop
				return true
			}
			return false
		}); err != nil {
			logger.WithError(err).Debug("Could not schedule to stop service instance")
			return err
		}
		logger.Debug("Service instance scheduled to stop")
	} else {
		if err := zks.DeleteState(conn, req); err != nil {
			logger.WithError(err).Debug("Could not delete service instance from offline host")
			return err
		}
		logger.Debug("Service instance deleted from offline host")
	}
	return nil
}

// StopServiceInstances stops all instances for a service
func (zk *zkf) StopServiceInstances(poolID, serviceID string) error {
	logger := plog.WithFields(log.Fields{
		"poolid":    poolID,
		"serviceid": serviceID,
	})

	// get the root-based connection to stop the service instance
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire root-based connection")
		return err
	}

	// keep track of host online states
	onlineHosts := make(map[string]bool)

	// get all the state ids of the service
	reqs, err := zks.GetServiceStateIDs(conn, poolID, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not get service state ids")
		return err
	}

	// for each state, if the host is online, stop the service;
	// if the host is offline delete the service.
	for _, req := range reqs {
		st8log := logger.WithField("instanceid", req.InstanceID)

		// check the host
		isOnline, ok := onlineHosts[req.HostID]
		if !ok {
			isOnline, err = zks.IsHostOnline(conn, poolID, req.HostID)
			if err != nil {
				logger.WithField("hostid", req.HostID).WithError(err).Warn("Could not check if host is online")
				continue
			}
			onlineHosts[req.HostID] = isOnline
		}

		// manage the service
		if isOnline {
			if err := zks.UpdateState(conn, req, func(s *zks.State) bool {
				if s.DesiredState != service.SVCStop {
					s.DesiredState = service.SVCStop
					return true
				}
				return false
			}); err != nil {
				st8log.WithError(err).Warn("Could not stop service instance")
				continue
			}
			st8log.Debug("Set service instance to stopped")
		} else {
			if err := zks.DeleteState(conn, req); err != nil {
				st8log.WithError(err).Warn("Could not delete service instance")
				continue
			}
			st8log.Debug("Deleted service instance")
		}
	}

	return nil
}

// SendDockerAction submits an action to the docker queue
func (zk *zkf) SendDockerAction(poolID, serviceID string, instanceID int, command string, args []string) error {
	logger := plog.WithFields(log.Fields{
		"poolid":     poolID,
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	// get the pool-based connection to send the docker action
	conn, err := zzk.GetLocalConnection(path.Join("/pools", poolID))
	if err != nil {
		logger.WithError(err).Debug("Could not acquire pool-based connection")
		return err
	}

	// get the state host id
	hostID, err := zks.GetServiceStateHostID(conn, "", serviceID, instanceID)
	if err != nil {
		logger.WithError(err).Debug("Could not get host id for state")
		return err
	}

	logger = logger.WithField("hostid", hostID)

	// set up the action
	req := zkd.Action{
		HostID:   hostID,
		DockerID: fmt.Sprintf("%s-%d", serviceID, instanceID),
		Command:  append([]string{command}, args...),
	}

	// send the action
	if _, err := zkd.SendAction(conn, &req); err != nil {
		logger.WithError(err).Debug("Could not send docker action")
		return err
	}

	logger.Debug("Submitted docker action")
	return nil
}

// GetServiceStateIDs returns the ids of all the states running on a given
// service
func (zk *zkf) GetServiceStateIDs(poolID, serviceID string) ([]zks.StateRequest, error) {
	logger := plog.WithFields(log.Fields{
		"poolid":    poolID,
		"serviceid": serviceID,
	})

	// get the root-based connection to look up the state ids for a service
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire root-based connection")
		return nil, err
	}

	return zks.GetServiceStateIDs(conn, poolID, serviceID)
}

func (zk *zkf) SyncServiceRegistry(tenantID string, svc *service.Service) error {
	logger := plog.WithFields(log.Fields{
		"tenantid":    tenantID,
		"serviceid":   svc.ID,
		"servicename": svc.Name,
		"poolid":      svc.PoolID,
	})

	// get the root-based connection to update the service's endpoints
	rootconn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not acquire a root-based connection to update the service's public endpoints in zookeeper")
		return err
	}

	zk.svcRegistry.Lock()
	defer zk.svcRegistry.Unlock()
	if zk.svcRegistry == nil {
		err = zk.buildServiceRegistryCache(rootconn)
		if err != nil {
			logger.WithError(err).Debug("Could not build cache of the service registry")
			return err
		}
	}

	// sync the registry
	syncRequest := zk.svcRegistry.BuildSyncRequest(tenantID, svc)
	if err := zkr.SyncServiceRegistry(rootconn, syncRequest); err != nil {
		logger.WithError(err).Debug("Could not update the service's public endpoints in zookeeper")
		return err
	}

	// Update our local cache now that we know that ZK was updated successfully
	 zk.svcRegistry.UpdateRegistry(syncRequest.ServiceID, syncRequest.PortsToPublish, syncRequest.VHostsToPublish)

	logger.Debug("Updated the service's public endpoints in zookeeper")
	return nil
}

// buildServiceRegistryCache builds the cache of public endpoints based on data in zookeeper
func (zk *zkf) buildServiceRegistryCache(conn client.Connection) error {
	publicPorts, err := zkr.GetPublicPorts(conn)
	if err != nil {
		return err
	}
	vhosts, err := zkr.GetVHosts(conn)
	if err != nil {
		return err
	}
	zk.svcRegistry.BuildCache(publicPorts, vhosts)
	return nil
}
