package service

import (
	"fmt"
	"path"
	"sort"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	zkutils "github.com/zenoss/serviced/zzk/utils"
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
	Service *service.Service
	version interface{}
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
	conn    client.Connection
	handler ServiceHandler
}

// NewServiceListener instantiates a new ServiceListener
func NewServiceListener(conn client.Connection, handler ServiceHandler) *ServiceListener {
	return &ServiceListener{conn, handler}
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

		rss, err := LoadRunningServicesByService(l.conn, serviceID)
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
	sort.Sort(instances(rss))
	netInstances := svc.Instances - len(rss)
	if netInstances > 0 {
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
	} else if netInstances < 0 {
		l.stop(rss[:-netInstances])
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

// UpdateService updates a service node if it exists, otherwise creates it
func UpdateService(conn client.Connection, svc *service.Service) error {
	if svc.ID == "" {
		return fmt.Errorf("service id required")
	}

	var (
		spath = servicepath(svc.ID)
		node  = &ServiceNode{Service: svc}
	)

	if exists, err := zkutils.PathExists(conn, spath); err != nil {
		return err
	} else if !exists {
		return conn.Create(spath, node)
	}
	return conn.Set(spath, node)
}
