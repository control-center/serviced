// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package service

import (
	"path"
	"sort"
	"sync"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/zzk"
)

const (
	zkService = "/services"
)

func servicepath(nodes ...string) string {
	p := append([]string{zkService}, nodes...)
	return path.Join(p...)
}

type instances []*dao.RunningService

func (inst instances) Len() int           { return len(inst) }
func (inst instances) Less(i, j int) bool { return inst[i].InstanceID < inst[j].InstanceID }
func (inst instances) Swap(i, j int)      { inst[i], inst[j] = inst[j], inst[i] }

// ServiceNode is the zookeeper client Node for services
type ServiceNode struct {
	*service.Service
	version interface{}
}

// ID implements zzk.Node
func (node *ServiceNode) ID() string {
	return node.ID
}

// Create implements zzk.Node
func (node *ServiceNode) Create(conn client.Connection) error {
	return UpdateService(conn, node.Service)
}

// Update implements zzk.Node
func (node *ServiceNode) Update(conn client.Connection) error {
	return UpdateService(conn, node.Service)
}

// Delete implements zzk.Node
func (node *ServiceNode) Delete(conn client.Connection) (err error) {
	var (
		cancel = make(chan interface{})
		done = make(chan interface{})
	)

	go func() {
		defer close(done) 
		err = RemoveService(cancel, conn, node.ID)
	}
	select {
	case <-time.After(45 * time.Second):
		close(cancel)
	case <-done:
	}
	<-done
	return err
}

// Version implements client.Node
func (node *ServiceNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServiceNode) SetVersion(version interface{}) { node.version = version }

// ServiceStateNode is the zookeeper client node for service states
type ServiceStateNode struct {
	ServiceState *servicestate.ServiceState
	version      interface{}
}

// Version implements client.Node
func (node *ServiceStateNode) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *ServiceStateNode) SetVersion(version interface{}) { node.version = version }

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
func NewServiceListener(conn client.Connection, handler ServiceHandler) *ServiceListener {
	return &ServiceListener{conn: conn, handler: handler}
}

// GetConnection implements zzk.Listener
func (l *ServiceListener) GetConnection() client.Connection { return l.conn }

// GetPath implements zzk.Listener
func (l *ServiceListener) GetPath(nodes ...string) string { return servicepath(nodes...) }

// Ready implements zzk.Listener
func (l *ServiceListener) Ready() (err error) { return }

// Done implements zzk.Listener
func (l *ServiceListener) Done() { return }

// Spawn watches a service and syncs the number of running instances
func (l *ServiceListener) Spawn(shutdown <-chan interface{}, serviceID string) {
	for {
		var svc service.Service
		event, err := l.conn.GetW(l.GetPath(serviceID), &ServiceNode{Service: &svc})
		if err != nil {
			glog.Errorf("Could not load service %s: %s", serviceID, err)
			return
		}

		rss, err := LoadRunningServicesByService(l.conn, svc.ID)
		if err != nil {
			glog.Errorf("Could not load states for service %s (%s): %s", svc.Name, svc.ID, err)
		}

		// Should the service be running at all?
		switch svc.DesiredState {
		case service.SVCStop:
			l.stop(rss)
		case service.SVCRun:
			l.sync(&svc, rss)
		default:
			glog.Warningf("Unexpected desired state %d for service %s (%s)", svc.DesiredState, svc.Name, svc.ID)
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				glog.V(2).Infof("Shutting down service %s (%s) due to node delete", svc.Name, svc.ID)
				l.stop(rss)
				return
			}
			glog.V(0).Infof("Service %s (%s) received event: %v", svc.Name, svc.ID, e)
		case <-shutdown:
			glog.V(2).Infof("Leader stopping watch for %s (%s)", svc.Name, svc.ID)
			l.stop(rss)
			return
		}
	}
}

func (l *ServiceListener) sync(svc *service.Service, rss []*dao.RunningService) {
	l.Lock()
	defer l.Unlock()
	sort.Sort(instances(rss))

	netInstances := svc.Instances - len(rss)
	if len(rss) > 0 && netInstances != 0 && utils.StringInSlice("restartAllOnInstanceChanged", svc.ChangeOptions) {
		netInstances = -len(rss)
	}

	if netInstances > 0 {
		glog.V(2).Infof("Starting %d instances of service %s (%s)", netInstances, svc.Name, svc.ID)
		var (
			last        = 0
			instanceIDs = make([]int, netInstances)
		)
		if count := len(rss); count > 0 {
			last = rss[count-1].InstanceID + 1
		}
		for i := 0; i < netInstances; i++ {
			instanceIDs[i] = last + i
		}
		l.start(svc, instanceIDs)
	} else if netInstances = -netInstances; netInstances > 0 {
		glog.V(2).Infof("Stopping %d instances of service %s (%s)", netInstances, svc.Name, svc.ID)
		l.stop(rss[:netInstances])
	}
}

func (l *ServiceListener) start(svc *service.Service, instanceIDs []int) {
	for _, i := range instanceIDs {
		host, err := l.handler.SelectHost(svc)
		if err != nil {
			glog.Warningf("Could not assign a host to service %s (%s): %s", svc.Name, svc.ID, err)
			continue
		}
		state, err := servicestate.BuildFromService(svc, host.ID)
		if err != nil {
			glog.Warningf("Error creating service state for service %s (%s): %s", svc.Name, svc.ID, err)
			continue
		}
		state.HostIP = host.IPAddr
		state.InstanceID = i
		if err := addInstance(l.conn, state); err != nil {
			glog.Warningf("Could not add service instance %s for service %s (%s): %s", state.ID, svc.Name, svc.ID, err)
			continue
		}
		glog.V(2).Infof("Starting service instance %s for service %s (%s) on host %s", state.ID, svc.Name, svc.ID, host.ID)
	}
}

func (l *ServiceListener) stop(rss []*dao.RunningService) {
	for _, state := range rss {
		if err := StopServiceInstance(l.conn, state.HostID, state.ID); err != nil {
			glog.Warningf("Service instance %s from %s (%s) won't die: %s", state.ID, state.Name, state.ServiceID, err)
			continue
		}
		glog.V(2).Infof("Stopping service instance %s for service %s on host %s", state.ID, state.ServiceID, state.HostID)
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
	node.Service.DesiredState = service.SVCRun
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
	node.Service.DesiredState = service.SVCStop
	return conn.Set(path, &node)
}

// SyncServices synchronizes all services into zookeeper
func SyncServices(conn client.Connection, services []*service.Service) error {
	nodes := make([]*ServiceNode, len(services))
	for i := range services {
		nodes[i] = &ServiceNode{Service:services[i]}
	}
	return zzk.Sync(conn, nodes, servicepath())
}

// UpdateService updates a service node if it exists, otherwise creates it
func UpdateService(conn client.Connection, svc *service.Service) error {
	var node ServiceNode
	spath := servicepath(svc.ID)

	// For some reason you can't just create the node with the service data
	// already set.  Trust me, I tried.  It was very aggravating.
	if err := conn.Get(spath, &node); err != nil {
		conn.Create(spath, &node)
	}
	node.Service = svc
	return conn.Set(spath, &node)
}

// RemoveServices stop any running services and deletes an existing service
func RemoveService(cancel <-chan interface{}, conn client.Connection, serviceID string) error {
	// Check if the path exists
	if exists, err := zzk.PathExists(conn, servicepath(serviceID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	// If it exists, stop the service
	if err := StopService(conn, serviceID); err != nil {
		return err
	}

	// Wait for there to be no running states
	for {
		children, event, err := conn.ChildrenW(servicepath(serviceID))
		if err != nil {
			return err
		}

		if len(children) == 0 {
			break
		}

		select {
		case <-event:
			// pass
		case <-cancel:
			glog.Infof("Gave up deleting service %s with %d children", serviceID, len(children))
			return zzk.ErrShutdown
		}
	}

	// Delete the service
	return conn.Delete(servicepath(serviceID))
}

// GetServiceState gets a service state
func GetServiceState(conn client.Connection, state *servicestate.ServiceState, serviceID string, stateID string) error {
	return conn.Get(servicepath(serviceID, stateID), &ServiceStateNode{ServiceState: state})
}

// GetServiceStates gets all service states for a particular service
func GetServiceStates(conn client.Connection, serviceIDs ...string) (states []*servicestate.ServiceState, err error) {
	for _, serviceID := range serviceIDs {
		stateIDs, err := conn.Children(servicepath(serviceID))
		if err != nil {
			return nil, err
		}

		for _, stateID := range stateIDs {
			var state servicestate.ServiceState
			if err := GetServiceState(conn, &state, serviceID, stateID); err != nil {
				return nil, err
			}
			states = append(states, &state)
		}
	}
	return states, nil
}

// UpdateServiceState updates a particular service state
func UpdateServiceState(conn client.Connection, state *servicestate.ServiceState) error {
	var node ServiceStateNode
	path := servicepath(state.ServiceID, state.ID)
	if err := conn.Get(path, &node); err != nil {
		return err
	}
	node.ServiceState = state
	return conn.Set(path, &node)
}
