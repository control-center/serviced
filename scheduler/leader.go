package scheduler

import (
	"fmt"
	"path"
	"time"

	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
	"github.com/zenoss/serviced/zzk/snapshot"
	"github.com/zenoss/serviced/zzk/virtualips"
)

type leader struct {
	facade  *facade.Facade
	dao     dao.ControlPlane
	conn    coordclient.Connection
	context datastore.Context
	poolID  string
}

// Lead is executed by the "leader" of the control plane cluster to handle its management responsibilities of:
//    services
//    snapshots
//    virtual IPs
func Lead(facade *facade.Facade, dao dao.ControlPlane, conn coordclient.Connection, zkEvent <-chan coordclient.Event) {
	glog.V(0).Info("Entering Lead()!")
	defer glog.V(0).Info("Exiting Lead()!")
	shutdownmode := false

	allPools, err := facade.GetResourcePools(datastore.Get())
	if err != nil {
		glog.Error(err)
		return
	} else if allPools == nil || len(allPools) == 0 {
		glog.Error("no resource pools found")
		return
	}

	for _, aPool := range allPools {
		// TODO: Support non default pools
		// Currently, only the default pool gets a leader
		if aPool.ID != "default" {
			glog.Warningf("Non default pool: %v (not currently supported)", aPool.ID)
			continue
		}

		leader := leader{facade: facade, dao: dao, conn: conn, context: datastore.Get(), poolID: aPool.ID}
		for {
			if shutdownmode {
				glog.V(1).Info("Shutdown mode encountered.")
				break
			}
			time.Sleep(time.Second)
			func() error {
				select {
				case evt := <-zkEvent:
					// shut this thing down
					shutdownmode = true
					glog.V(0).Info("Got a zkevent, leaving lead: ", evt)
					return nil
				default:
					glog.V(0).Info("Processing leader duties")
					// passthru
				}

				// creates a listener for snapshots with a function call to take snapshots
				// and return the label and error message
				go snapshot.Listen(conn, func(serviceID string) (string, error) {
					var label string
					err := dao.TakeSnapshot(serviceID, &label)
					return label, err
				})

				leader.watchServices()
				return nil
			}()
		}
	}
}

func snapShotName(volumeName string) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return volumeName + "_" + utc.Format(format)
}

func (l *leader) watchServices() {
	conn := l.conn
	processing := make(map[string]chan int)
	sDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		glog.V(0).Info("Leader shutting down child goroutines")
		for key, shutdown := range processing {
			glog.V(1).Info("Sending shutdown signal for ", key)
			shutdown <- 1
		}
	}()

	conn.CreateDir(zzk.SERVICE_PATH)

	for {
		glog.V(1).Info("Leader watching for changes to ", zzk.SERVICE_PATH)
		serviceIds, zkEvent, err := conn.ChildrenW(zzk.SERVICE_PATH)
		if err != nil {
			glog.Errorf("Leader unable to find any services: %s", err)
			return
		}
		for _, serviceID := range serviceIds {
			if processing[serviceID] == nil {
				glog.V(2).Info("Leader starting goroutine to watch ", serviceID)
				serviceChannel := make(chan int)
				processing[serviceID] = serviceChannel
				go l.watchService(serviceChannel, sDone, serviceID)
			}
		}

	VirtualIPWatching:
		for {
			select {
			case evt := <-zkEvent:
				glog.V(1).Info("Leader event: ", evt)
				break VirtualIPWatching
			case serviceID := <-sDone:
				glog.V(1).Info("Leading cleaning up for service ", serviceID)
				delete(processing, serviceID)
				break VirtualIPWatching
			case <-time.After(10 * time.Second):
				// every 10 seconds, sync the virtual IPs in the model to zookeeper nodes
				myPool, err := l.facade.GetResourcePool(l.context, l.poolID)
				if err != nil {
					glog.Errorf("Unable to load resource pool: %v", l.poolID)
				} else if myPool == nil {
					glog.Errorf("Pool ID: %v could not be found", l.poolID)
				}

				if err := virtualips.SyncVirtualIPs(l.conn, myPool.VirtualIPs); err != nil {
					glog.Warningf("SyncVirtualIPs: %v", err)
				}
			}
		}
	}
}

func (l *leader) watchService(shutdown <-chan int, done chan<- string, serviceID string) {
	conn := l.conn
	defer func() {
		glog.V(3).Info("Exiting function watchService ", serviceID)
		done <- serviceID
	}()
	for {
		var svc service.Service
		zkEvent, err := zzk.LoadServiceW(conn, serviceID, &svc)
		if err != nil {
			glog.Errorf("Unable to load service %s: %v", serviceID, err)
			return
		}
		_, childEvent, err := conn.ChildrenW(zzk.ServicePath(serviceID))

		glog.V(1).Info("Leader watching for changes to service ", svc.Name)

		switch exists, err := conn.Exists(path.Join("/services", serviceID)); {
		case err != nil:
			glog.Errorf("conn.Exists failed (%v)", err)
			return
		case exists == false:
			glog.V(2).Infof("no /service node for: %s", serviceID)
			return
		}

		// check current state
		var serviceStates []*servicestate.ServiceState
		err = zzk.GetServiceStates(l.conn, &serviceStates, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return
		}

		// Is the service supposed to be running at all?
		switch {
		case svc.DesiredState == service.SVCStop:
			shutdownServiceInstances(l.conn, serviceStates, len(serviceStates))
		case svc.DesiredState == service.SVCRun:
			if err := l.updateServiceInstances(&svc, serviceStates); err != nil {
				glog.Errorf("%v", err)
			}
		default:
			glog.Warningf("Unexpected desired state %d for service %s", svc.DesiredState, svc.Name)
		}

		select {
		case evt := <-zkEvent:
			if evt.Type == coordclient.EventNodeDeleted {
				glog.V(0).Info("Shutting down due to node delete ", serviceID)
				shutdownServiceInstances(l.conn, serviceStates, len(serviceStates))
				return
			}
			glog.V(1).Infof("Service %s received event: %v", svc.Name, evt)
			continue

		case evt := <-childEvent:
			glog.V(1).Infof("Service %s received child event: %v", svc.Name, evt)
			continue

		case <-shutdown:
			glog.V(1).Info("Leader stopping watch on ", svc.Name)
			return
		}
	}
}

func (l *leader) updateServiceInstances(service *service.Service, serviceStates []*servicestate.ServiceState) error {
	//	var err error
	// pick services instances to start
	if len(serviceStates) < service.Instances {
		instancesToStart := service.Instances - len(serviceStates)
		glog.V(2).Infof("updateServiceInstances wants to start %d instances", instancesToStart)
		hosts, err := l.facade.FindHostsInPool(l.context, service.PoolID)
		if err != nil {
			glog.Errorf("Leader unable to acquire hosts for pool %s: %v", service.PoolID, err)
			return err
		}
		if len(hosts) == 0 {
			glog.Warningf("Pool %s has no hosts", service.PoolID)
			return nil
		}

		return l.startServiceInstances(service, hosts, instancesToStart)

	} else if len(serviceStates) > service.Instances {
		instancesToKill := len(serviceStates) - service.Instances
		glog.V(2).Infof("updateServiceInstances wants to kill %d instances", instancesToKill)
		shutdownServiceInstances(l.conn, serviceStates, instancesToKill)
	}
	return nil

}

// getFreeInstanceIDs looks up running instances of this service and returns n
// unused instance ids.
// Note: getFreeInstanceIDs does NOT validate that instance ids do not exceed
// max number of instances for the service. We're already doing that check in
// another, better place. It is guaranteed that either nil or n ids will be
// returned.
func getFreeInstanceIDs(conn coordclient.Connection, svc *service.Service, n int) ([]int, error) {
	var (
		states []*servicestate.ServiceState
		ids    []int
	)
	// Look up existing instances
	err := zzk.GetServiceStates(conn, &states, svc.Id)
	if err != nil {
		return nil, err
	}
	// Populate the used set
	used := make(map[int]struct{})
	for _, s := range states {
		used[s.InstanceID] = struct{}{}
	}
	// Find n unused ids
	for i := 0; len(ids) < n; i++ {
		if _, ok := used[i]; !ok {
			// Id is unused
			ids = append(ids, i)
		}
	}
	return ids, nil
}
func (l *leader) startServiceInstances(svc *service.Service, hosts []*host.Host, numToStart int) error {
	glog.V(1).Infof("Starting %d instances, choosing from %d hosts", numToStart, len(hosts))

	// Get numToStart free instance ids
	freeids, err := getFreeInstanceIDs(l.conn, svc, numToStart)
	if err != nil {
		return err
	}

	hostPolicy := NewServiceHostPolicy(svc, l.dao)

	// Start up an instance per id
	for _, i := range freeids {
		servicehost, err := l.selectPoolHostForService(svc, hosts, hostPolicy)
		if err != nil {
			return err
		}

		glog.V(2).Info("Selected host ", servicehost)
		serviceState, err := servicestate.BuildFromService(svc, servicehost.ID)
		if err != nil {
			glog.Errorf("Error creating ServiceState instance: %v", err)
			return err
		}

		serviceState.HostIP = servicehost.IPAddr
		serviceState.InstanceID = i
		err = zzk.AddServiceState(l.conn, serviceState)
		if err != nil {
			glog.Errorf("Leader unable to add service state: %v", err)
			return err
		}
		glog.V(2).Info("Started ", serviceState)
	}
	return nil
}

func shutdownServiceInstances(conn coordclient.Connection, serviceStates []*servicestate.ServiceState, numToKill int) {
	glog.V(1).Infof("Stopping %d instances from %d total", numToKill, len(serviceStates))
	for i := 0; i < numToKill; i++ {
		glog.V(2).Infof("Killing host service state %s:%s\n", serviceStates[i].HostID, serviceStates[i].Id)
		serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
		err := zzk.TerminateHostService(conn, serviceStates[i].HostID, serviceStates[i].Id)
		if err != nil {
			glog.Warningf("%s:%s wouldn't die", serviceStates[i].HostID, serviceStates[i].Id)
		}
	}
}

// selectPoolHostForService chooses a host from the pool for the specified service. If the service
// has an address assignment the host will already be selected. If not the host with the least amount
// of memory committed to running containers will be chosen.
func (l *leader) selectPoolHostForService(s *service.Service, hosts []*host.Host, policy *ServiceHostPolicy) (*host.Host, error) {
	var assignmentType string
	var ipAddr string
	var hostid string
	for _, ep := range s.Endpoints {
		if ep.AddressAssignment != (addressassignment.AddressAssignment{}) {
			assignmentType = ep.AddressAssignment.AssignmentType
			ipAddr = ep.AddressAssignment.IPAddr
			hostid = ep.AddressAssignment.HostID
			break
		}
	}

	if assignmentType == "virtual" {
		// populate hostid
		if err := virtualips.GetVirtualIPHostID(l.conn, ipAddr, &hostid); err != nil {
			return nil, err
		}
		glog.Infof("Service: %v has been assigned virtual IP: %v which has been locked and configured on host %s", s.Name, ipAddr, hostid)
	}

	if hostid != "" {
		return poolHostFromAddressAssignments(hostid, hosts)
	}

	return policy.SelectHost(hosts)
}

// poolHostFromAddressAssignments determines the pool host for the service from its address assignment(s).
func poolHostFromAddressAssignments(hostid string, hosts []*host.Host) (*host.Host, error) {
	// ensure the assigned host is in the pool
	for _, h := range hosts {
		if h.ID == hostid {
			return h, nil
		}
	}

	return nil, fmt.Errorf("assigned host is not in pool")
}
