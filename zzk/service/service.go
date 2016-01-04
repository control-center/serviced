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

package service

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

const (
	zkService    = "/services"
	retryTimeout = time.Second
)

var ErrServiceIsRunning = errors.New("can only delete services in a stopped state")

type HasRunningInstances struct {
	ServiceID string
	Instances int
}

func (err HasRunningInstances) Error() string {
	return fmt.Sprintf("service %s has %d running instances", err.ServiceID, err.Instances)
}

func servicepath(nodes ...string) string {
	p := append([]string{zkService}, nodes...)
	return path.Join(p...)
}

type instances []dao.RunningService

func (inst instances) Len() int           { return len(inst) }
func (inst instances) Less(i, j int) bool { return inst[i].InstanceID < inst[j].InstanceID }
func (inst instances) Swap(i, j int)      { inst[i], inst[j] = inst[j], inst[i] }

// ServiceNode is the zookeeper client Node for services
type ServiceNode struct {
	*service.Service
	Locked  bool
	version interface{}
}

// Version implements client.Node
func (node *ServiceNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServiceNode) SetVersion(version interface{}) { node.version = version }

// ServiceHandler handles all non-zookeeper interactions required by the service
type ServiceHandler interface {
	SelectHost(*service.Service) (*host.Host, error)
}

// ServiceListener is the listener for /services
type ServiceListener struct {
	sync.Mutex
	conn    client.Connection
	handler ServiceHandler
}

// NewServiceListener instantiates a new ServiceListener
func NewServiceListener(handler ServiceHandler) *ServiceListener {
	return &ServiceListener{handler: handler}
}

// SetConnection implements zzk.Listener
func (l *ServiceListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath implements zzk.Listener
func (l *ServiceListener) GetPath(nodes ...string) string { return servicepath(nodes...) }

// Ready implements zzk.Listener
func (l *ServiceListener) Ready() (err error) { return }

// Done implements zzk.Listener
func (l *ServiceListener) Done() { return }

// PostProcess implements zzk.Listener
func (l *ServiceListener) PostProcess(p map[string]struct{}) {}

// Spawn watches a service and syncs the number of running instances
func (l *ServiceListener) Spawn(shutdown <-chan interface{}, serviceID string) {
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		var retry <-chan time.Time
		var err error

		var svcnode ServiceNode
		var svc service.Service
		svcnode.Service = &svc
		serviceEvent, err := l.conn.GetW(l.GetPath(serviceID), &svcnode, done)
		if err != nil {
			glog.Errorf("Could not load service %s: %s", serviceID, err)
			return
		}

		stateIDs, stateEvent, err := l.conn.ChildrenW(l.GetPath(serviceID), done)
		if err != nil {
			glog.Errorf("Could not load service states for %s: %s", serviceID, err)
			return
		}

		rss, err := l.getServiceStates(&svc, stateIDs)
		if err != nil {
			glog.Warningf("Could not get service states for service %s (%s): %s", svc.Name, svc.ID, err)
			retry = time.After(retryTimeout)
		} else {
			// Should the service be running at all?
			switch service.DesiredState(svc.DesiredState) {
			case service.SVCStop:
				l.stop(rss)
			case service.SVCRun:
				if !l.sync(svcnode.Locked, &svc, rss) {
					retry = time.After(retryTimeout)
				}
			case service.SVCPause:
				l.pause(rss)
			default:
				glog.Warningf("Unexpected desired state %d for service %s (%s)", svc.DesiredState, svc.Name, svc.ID)
			}
		}

		glog.V(2).Infof("Service %s (%s) waiting for event", svc.Name, svc.ID)

		select {
		case e := <-serviceEvent:
			if e.Type == client.EventNodeDeleted {
				glog.V(2).Infof("Shutting down service %s (%s) due to node delete", svc.Name, svc.ID)
				l.stop(rss)
				return
			}
			glog.V(2).Infof("Service %s (%s) received event: %v", svc.Name, svc.ID, e)
		case e := <-stateEvent:
			if e.Type == client.EventNodeDeleted {
				glog.V(2).Infof("Shutting down service %s (%s) due to node delete", svc.Name, svc.ID)
				l.stop(rss)
				return
			}
			glog.V(2).Infof("Service %s (%s) received event: %v", svc.Name, svc.ID, e)
		case <-retry:
			glog.Infof("Re-syncing service %s (%s)", svc.Name, svc.ID)
		case <-shutdown:
			glog.V(2).Infof("Leader stopping watch for %s (%s)", svc.Name, svc.ID)
			return
		}

		close(done)
		done = make(chan struct{})
	}
}

// getActiveHosts returns a map of all the available hosts
func (l *ServiceListener) getActiveHosts() (map[string]struct{}, error) {
	hosts, err := GetActiveHosts(l.conn)
	if err != nil {
		return nil, err
	}
	hostmap := make(map[string]struct{})
	for _, host := range hosts {
		hostmap[host] = struct{}{}
	}
	return hostmap, nil
}

// getServiceStates returns all the valid service states on a service
func (l *ServiceListener) getServiceStates(svc *service.Service, stateIDs []string) ([]dao.RunningService, error) {
	// figure out which hosts are still available
	hosts, err := l.getActiveHosts()
	if err != nil {
		return nil, err
	}
	var rss []dao.RunningService
	for _, stateID := range stateIDs {
		// get the service state
		var state servicestate.ServiceState
		if err := l.conn.Get(servicepath(svc.ID, stateID), &ServiceStateNode{ServiceState: &state}); err != nil {
			if err != client.ErrNoNode {
				glog.Errorf("Could not look up service instance %s for service %s (%s): %s", stateID, svc.Name, svc.ID, err)
				return nil, err
			}
			continue
		}
		// is the host currently active?
		var isActive bool
		if _, isActive = hosts[state.HostID]; isActive {
			if isActive, err = l.conn.Exists(hostpath(state.HostID, state.ID)); err != nil && err != client.ErrNoNode {
				glog.Errorf("Could not look up host instance %s on host %s for service %s: %s", state.ID, state.HostID, state.ServiceID, err)
				return nil, err
			}
		}
		if !isActive {
			// if the host is not active, remove the node
			glog.Infof("Service instance %s of service %s (%s) running on host %s (%s) is not active, rescheduling", state.ID, svc.Name, svc.ID, state.HostIP, state.HostID)
			if err := removeInstance(l.conn, state.ServiceID, state.HostID, state.ID); err != nil {
				glog.Errorf("Could not delete service instance %s for service %s: %s", state.ID, state.ServiceID, err)
				return nil, err
			}
		} else {
			rs, err := NewRunningService(svc, &state)
			if err != nil {
				glog.Errorf("Could not get service instance %s for service %s (%s): %s", state.ID, svc.Name, svc.ID, err)
				return nil, err
			}
			rss = append(rss, *rs)
		}
	}
	return rss, nil
}

// sync synchronizes the number of running instances for this service
func (l *ServiceListener) sync(locked bool, svc *service.Service, rss []dao.RunningService) bool {
	// sort running services by instance ID, so that you stop instances by the
	// lowest instance ID first and start instances with the greatest instance
	// ID last.
	sort.Sort(instances(rss))

	// resume any paused running services
	for _, state := range rss {
		// resumeInstance updates the service state ONLY if it has a PAUSED DesiredState
		if err := resumeInstance(l.conn, state.HostID, state.ID); err != nil {
			glog.Warningf("Could not resume paused service instance %s (%s) for service %s on host %s: %s", state.ID, state.Name, state.ServiceID, state.HostID, err)
			return false
		}
	}

	// if the service has a change option for restart all on changed, stop all
	// instances and wait for the nodes to stop.  Once all service instances
	// have been stopped (deleted), then go ahead and start the instances back
	// up.
	if count := len(rss); count > 0 && count != svc.Instances && utils.StringInSlice("restartAllOnInstanceChanged", svc.ChangeOptions) {
		svc.Instances = 0 // NOTE: this will not update the node in zk or elastic
	}

	// netInstances is the difference between the number of instances that
	// should be running, as described by the service from the number of
	// instances that are currently running
	netInstances := svc.Instances - len(rss)

	if netInstances > 0 {
		// If the service lock is enabled, do not try to start any service instances
		// This will prevent the retry restart from activating
		if locked {
			glog.Warningf("Could not start %d instances; service %s (%s) is locked", netInstances, svc.Name, svc.ID)
			return true
		}
		// the number of running instances is *less* than the number of
		// instances that need to be running, so schedule instances to start
		glog.V(2).Infof("Starting %d instances of service %s (%s)", netInstances, svc.Name, svc.ID)
		var (
			last        = 0
			instanceIDs = make([]int, netInstances)
		)

		// Find which instances IDs are being unused and add those instances
		// first.  All SERVICES must have an instance ID of 0, if instance ID
		// zero dies for whatever reason, then the service must schedule
		// another 0-id instance to take its place.
		j := 0
		for i := range instanceIDs {
			for j < len(rss) && last == rss[j].InstanceID {
				// if instance ID exists, then keep searching the list for
				// the next unique instance ID
				last += 1
				j += 1
			}
			instanceIDs[i] = last
			last += 1
		}

		return netInstances == l.start(svc, instanceIDs)
	} else if netInstances = -netInstances; netInstances > 0 {
		// the number of running instances is *greater* than the number of
		// instances that need to be running, so schedule instances to stop of
		// the highest instance IDs.
		glog.V(2).Infof("Stopping %d of %d instances of service %s (%s)", netInstances, len(rss), svc.Name, svc.ID)
		l.stop(rss[svc.Instances:])
	}

	return true
}

// start schedules the given service instances with the provided instance ID.
func (l *ServiceListener) start(svc *service.Service, instanceIDs []int) int {
	var i, id int

	for i, id = range instanceIDs {
		if success := func(instanceID int) bool {
			glog.V(2).Infof("Waiting to acquire scheduler lock for service %s (%s)", svc.Name, svc.ID)
			// only one service instance can be scheduled at a time
			l.Lock()
			defer l.Unlock()

			host, err := l.handler.SelectHost(svc)
			if err != nil {
				glog.Warningf("Could not assign a host to service %s (%s): %s", svc.Name, svc.ID, err)
				return false
			}

			glog.V(2).Infof("Host %s found, building service instance %d for %s (%s)", host.ID, id, svc.Name, svc.ID)

			state, err := servicestate.BuildFromService(svc, host.ID)
			if err != nil {
				glog.Warningf("Error creating service state for service %s (%s): %s", svc.Name, svc.ID, err)
				return false
			}

			state.HostIP = host.IPAddr
			state.InstanceID = instanceID
			if err := addInstance(l.conn, *state); err != nil {
				glog.Warningf("Could not add service instance %s for service %s (%s): %s", state.ID, svc.Name, svc.ID, err)
				return false
			}
			glog.V(2).Infof("Starting service instance %s for service %s (%s) on host %s", state.ID, svc.Name, svc.ID, host.ID)
			return true
		}(id); !success {
			// 'i' is the index of the unsuccessful instance id which should portray
			// the number of successful instances.  If you have 2 successful instances
			// started, then i = 2 because it attempted to create the third index and
			// failed
			glog.Warningf("Started %d of %d service instances for %s (%s)", i, len(instanceIDs), svc.Name, svc.ID)
			return i
		}
	}
	// add 1 because the index of the last instance 'i' would be len(instanceIDs) - 1
	return i + 1
}

// stop unschedules the provided service instances
func (l *ServiceListener) stop(rss []dao.RunningService) {
	for _, state := range rss {
		if err := StopServiceInstance(l.conn, state.HostID, state.ID); err != nil {
			glog.Warningf("Service instance %s (%s) from service %s won't die: %s", state.ID, state.Name, state.ServiceID, err)
			removeInstance(l.conn, state.ServiceID, state.HostID, state.ID)
			continue
		}
		glog.V(2).Infof("Stopping service instance %s (%s) for service %s on host %s", state.ID, state.Name, state.ServiceID, state.HostID)
	}
}

// pause updates the state of the given service instance to paused
func (l *ServiceListener) pause(rss []dao.RunningService) {
	for _, state := range rss {
		// pauseInstance updates the service state ONLY if it has a RUN DesiredState
		if err := pauseInstance(l.conn, state.HostID, state.ID); err != nil {
			glog.Warningf("Could not pause service instance %s (%s) for service %s: %s", state.ID, state.Name, state.ServiceID, err)
			continue
		}
		glog.V(2).Infof("Pausing service instance %s (%s) for service %s on host %s", state.ID, state.Name, state.ServiceID, state.HostID)
	}
}

// StartService schedules a service to start
func StartService(conn client.Connection, serviceID string) error {
	glog.Infof("Scheduling service %s to start", serviceID)
	var node ServiceNode
	path := servicepath(serviceID)

	if err := conn.Get(path, &node); err != nil {
		return err
	}
	node.Service.DesiredState = int(service.SVCRun)
	return conn.Set(path, &node)
}

// StopService schedules a service to stop
func StopService(conn client.Connection, serviceID string) error {
	glog.Infof("Scheduling service %s to stop", serviceID)
	var node ServiceNode
	path := servicepath(serviceID)

	if err := conn.Get(path, &node); err != nil {
		return err
	}
	node.Service.DesiredState = int(service.SVCStop)
	return conn.Set(path, &node)
}

// SyncServices synchronizes all services into zookeeper
func SyncServices(conn client.Connection, svcs []service.Service) error {
	// Make sure service path exists (during upgrades, sometimes it disappears: CC-1691)
	if err := conn.CreateDir(servicepath()); err != nil && err != client.ErrNodeExists {
		glog.Errorf("Cannot create service path %s: %s", servicepath(), err)
		return err
	}

	svcNodes, err := conn.Children(servicepath())
	if err != nil {
		glog.Errorf("Can not look up services: %s", err)
		return err
	}
	svcNodeMap := make(map[string]struct{})
	for _, svcNode := range svcNodes {
		svcNodeMap[svcNode] = struct{}{}
	}
	for _, svc := range svcs {
		if _, ok := svcNodeMap[svc.ID]; ok {
			delete(svcNodeMap, svc.ID)
		}
		if err := UpdateService(conn, svc, false, false); err != nil {
			glog.Errorf("Could not sync service %s: %s", svc.ID, err)
			return err
		}
	}
	for svcID := range svcNodeMap {
		if err := conn.Delete(servicepath(svcID)); err != nil {
			glog.Errorf("Could not remove service %s: %s", svcID, err)
			return err
		}
	}
	return nil
}

// UpdateService updates a service node if it exists, otherwise creates it
func UpdateService(conn client.Connection, svcData service.Service, setLockOnCreate, setLockOnUpdate bool) error {
	// svc is the service to be marshalled into zookeeper
	svc := &service.Service{
		ID:              svcData.ID,
		Name:            svcData.Name,
		Startup:         svcData.Startup,
		Instances:       svcData.Instances,
		ChangeOptions:   svcData.ChangeOptions,
		ImageID:         svcData.ImageID,
		LogConfigs:      svcData.LogConfigs,
		DesiredState:    svcData.DesiredState,
		HostPolicy:      svcData.HostPolicy,
		Privileged:      svcData.Privileged,
		Endpoints:       svcData.Endpoints,
		Volumes:         svcData.Volumes,
		Snapshot:        svcData.Snapshot,
		RAMCommitment:   svcData.RAMCommitment,
		CPUCommitment:   svcData.CPUCommitment,
		HealthChecks:    svcData.HealthChecks,
		MemoryLimit:     svcData.MemoryLimit,
		CPUShares:       svcData.CPUShares,
		ParentServiceID: svcData.ParentServiceID,
		Hostname:        svcData.Hostname,
	}
	svcNodePath := servicepath(svc.ID)
	svcNode := ServiceNode{Service: &service.Service{}}
	if err := conn.Get(svcNodePath, &svcNode); err == client.ErrNoNode {
		// the node does not exist, so create it
		// setLockOnCreate sets the lock as the node is created.
		svcNode.Locked = setLockOnCreate
		svcNode.Service = svc
		if err := conn.Create(svcNodePath, &svcNode); err != nil {
			glog.Errorf("Could not create node at %s: %s", svcNodePath, err)
			return err
		}
	} else if err != nil {
		glog.Errorf("Could not look up node for service %s: %s", svc.ID, err)
		return err
	}
	// setLockOnUpdate sets the lock to true if enabled, otherwise it uses
	// whatever existing value was previously set on the node.
	if setLockOnUpdate {
		svcNode.Locked = true
	}
	svcNode.Service = svc
	if err := conn.Set(svcNodePath, &svcNode); err != nil {
		glog.Errorf("Could not update node for service %s: %s", svc.ID, err)
		return err
	}
	return nil
}

// RemoveService deletes a service
func RemoveService(conn client.Connection, serviceID string) error {
	// Check if the path exists
	if exists, err := zzk.PathExists(conn, servicepath(serviceID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	// If the service has any children, do not delete
	if states, err := conn.Children(servicepath(serviceID)); err != nil {
		return err
	} else if instances := len(states); instances > 0 {
		return fmt.Errorf("service %s has %d running instances", serviceID, instances)
	}
	// Delete the service
	return conn.Delete(servicepath(serviceID))
}

// WaitService waits for a particular service's instances to reach a particular state
func WaitService(shutdown <-chan interface{}, conn client.Connection, serviceID string, desiredState service.DesiredState) error {
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		// Get the list of service states
		stateIDs, event, err := conn.ChildrenW(servicepath(serviceID), done)
		if err != nil {
			return err
		}
		count := len(stateIDs)

		switch desiredState {
		case service.SVCStop:
			// if there are no instances, then the service is stopped
			if count == 0 {
				return nil
			}
		case service.SVCRun, service.SVCRestart:
			// figure out which service instances are actively running and decrement non-running instances
			for _, stateID := range stateIDs {
				var state ServiceStateNode
				if err := conn.Get(servicepath(serviceID, stateID), &state); err == client.ErrNoNode {
					// if the instance does not exist, then that instance is no running
					count--
				} else if err != nil {
					return err
				} else if !state.IsRunning() {
					count--
				}
			}

			// Get the service node and verify that the number of running instances meets or exceeds the number
			// of instances required by the service
			var node ServiceNode
			node.Service = &service.Service{}
			if err := conn.Get(servicepath(serviceID), &node); err != nil {
				return err
			} else if count >= node.Instances {
				return nil
			}
		case service.SVCPause:
			// figure out which services have stopped or paused
			for _, stateID := range stateIDs {
				var state ServiceStateNode
				if err := conn.Get(servicepath(serviceID, stateID), &state); err == client.ErrNoNode {
					// if the instance does not exist, then it is not runng (so it is paused)
					count--
				} else if err != nil {
					return err
				} else if state.IsPaused() {
					count--
				}
			}
			// no instances should be running for all instances to be considered paused
			if count == 0 {
				return nil
			}
		default:
			return fmt.Errorf("invalid desired state")
		}

		if len(stateIDs) > 0 {
			// wait for each instance to reach the desired state
			for _, stateID := range stateIDs {
				if err := wait(shutdown, conn, serviceID, stateID, desiredState); err != nil {
					return err
				}
			}
			select {
			case <-shutdown:
				return zzk.ErrShutdown
			default:
			}
		} else {
			// otherwise, wait for a change in the number of children
			select {
			case <-event:
			case <-shutdown:
				return zzk.ErrShutdown
			}
		}

		close(done)
		done = make(chan struct{})
	}
}
