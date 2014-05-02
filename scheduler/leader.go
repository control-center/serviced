package scheduler

import (
	"fmt"
	"path"
	"time"

	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
)

type leader struct {
	facade  *facade.Facade
	dao     dao.ControlPlane
	conn    coordclient.Connection
	context datastore.Context
}

// Lead is executed by the "leader" of the control plane cluster to handle its management responsibilities of:
//    services
//    snapshots
//    virtual IPs
func Lead(facade *facade.Facade, dao dao.ControlPlane, conn coordclient.Connection, zkEvent <-chan coordclient.Event) {
	glog.V(0).Info("Entering Lead()!")
	defer glog.V(0).Info("Exiting Lead()!")
	shutdownmode := false
	leader := leader{facade: facade, dao: dao, conn: conn, context: datastore.Get()}
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

			go leader.watchSnapshotRequests()
			leader.watchServices()
			return nil
		}()
	}
}

func snapShotName(volumeName string) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return volumeName + "_" + utc.Format(format)
}

func (l *leader) watchSnapshotRequests() {
	glog.V(3).Info("started watchSnapshotRequestss")
	defer glog.V(3).Info("finished watchSnapshotRequestss")
	conn := l.conn
	cpDao := l.dao
	// make sure toplevel paths exist
	paths := []string{zzk.SNAPSHOT_PATH, zzk.SNAPSHOT_REQUEST_PATH}
	for _, path := range paths {
		exists, err := conn.Exists(path)
		if err != nil {
			if err == coordclient.ErrNoNode {
				if err := conn.CreateDir(path); err != nil {
					glog.Errorf("Leader unable to create znode:%s error: %s", path, err)
					return
				}
			} else {
				glog.Errorf("Leader unable to get status for znode:%s error: %s", path, err)
				return
			}
		}
		if !exists {
			if err := conn.CreateDir(path); err != nil {
				glog.Errorf("Leader unable to create znode:%s error: %s", path, err)
				return
			}
		}
	}

	// watch for snapshot requests and perform snapshots
	glog.V(0).Info("Leader watching for snapshot requests to ", zzk.SNAPSHOT_REQUEST_PATH)
	for {
		requestIds, zkEvent, err := conn.ChildrenW(zzk.SNAPSHOT_REQUEST_PATH)
		if err != nil {
			glog.Errorf("Leader unable to watch for snapshot requests to %s error: %s", zzk.SNAPSHOT_REQUEST_PATH, err)
			return
		}
		for _, requestID := range requestIds {
			snapshotRequest := dao.SnapshotRequest{}
			if err := zzk.LoadSnapshotRequest(conn, requestID, &snapshotRequest); err != nil {
				glog.Errorf("Leader unable to load snapshot request: %s  error: %s", requestID, err)
				snapshotRequest.SnapshotError = err.Error()
				zzk.UpdateSnapshotRequest(conn, &snapshotRequest)
				continue
			}
			if snapshotRequest.SnapshotLabel != "" {
				// already performed this request since SnapshotLabel is set
				continue
			}
			if snapshotRequest.SnapshotError != "" {
				// already performed this request since SnapshotError is set
				continue
			}

			glog.V(0).Infof("Leader starting snapshot for request: %+v", snapshotRequest)

			// TODO: perform snapshot request here
			snapLabel := ""
			if err := cpDao.LocalSnapshot(snapshotRequest.ServiceId, &snapLabel); err != nil {
				glog.V(0).Infof("watchSnapshotRequests: snaps.ExecuteSnapshot err=%s", err)
				snapshotRequest.SnapshotError = err.Error()
				snapshotRequest.SnapshotLabel = snapLabel
				zzk.UpdateSnapshotRequest(conn, &snapshotRequest)
				continue
			}

			snapshotRequest.SnapshotLabel = snapLabel
			if err := zzk.UpdateSnapshotRequest(conn, &snapshotRequest); err != nil {
				glog.Errorf("Leader unable to update snapshot request: %+v  error: %s", snapshotRequest, err)
				snapshotRequest.SnapshotError = err.Error()
				zzk.UpdateSnapshotRequest(conn, &snapshotRequest)
				continue
			}

			glog.V(0).Infof("Leader finished snapshot for request: %+v", snapshotRequest)
		}
		select {
		case evt := <-zkEvent:
			glog.V(2).Infof("Leader snapshot request watch event: %+v", evt)
		}
	}
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
		select {
		case evt := <-zkEvent:
			glog.V(1).Info("Leader event: ", evt)
			break
		case serviceID := <-sDone:
			glog.V(1).Info("Leading cleaning up for service ", serviceID)
			delete(processing, serviceID)
			break
		}

		/*
			for {
				select {
				case evt := <-zkEvent:
					glog.V(1).Info("Leader event: ", evt)
					break
				case serviceID := <-sDone:
					glog.V(1).Info("Leading cleaning up for service ", serviceID)
					delete(processing, serviceID)
					break
				case <-time.After(10 * time.Second):
					err := l.watchVirtualIPs()
					//err := watchVirtualIPs(l.context, l.facade)
					if err != nil {
						glog.Warningf("watchVirtualIPs: %v", err)
					}
				}
			}
		*/
	}
}

func (l *leader) watchService(shutdown <-chan int, done chan<- string, serviceID string) {
	conn := l.conn
	defer func() {
		glog.V(3).Info("Exiting function watchService ", serviceID)
		done <- serviceID
	}()
	for {
		var service dao.Service
		zkEvent, err := zzk.LoadServiceW(conn, serviceID, &service)
		if err != nil {
			glog.Errorf("Unable to load service %s: %v", serviceID, err)
			return
		}
		_, childEvent, err := conn.ChildrenW(zzk.ServicePath(serviceID))

		glog.V(1).Info("Leader watching for changes to service ", service.Name)

		switch exists, err := conn.Exists(path.Join("/services", serviceID)); {
		case err != nil:
			glog.Errorf("conn.Exists failed (%v)", err)
			return
		case exists == false:
			glog.V(2).Infof("no /service node for: %s", serviceID)
			return
		}

		// check current state
		var serviceStates []*dao.ServiceState
		err = zzk.GetServiceStates(l.conn, &serviceStates, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return
		}

		// Is the service supposed to be running at all?
		switch {
		case service.DesiredState == dao.SVC_STOP:
			shutdownServiceInstances(l.conn, serviceStates, len(serviceStates))
		case service.DesiredState == dao.SVC_RUN:
			l.updateServiceInstances(&service, serviceStates)
		default:
			glog.Warningf("Unexpected desired state %d for service %s", service.DesiredState, service.Name)
		}

		select {
		case evt := <-zkEvent:
			if evt.Type == coordclient.EventNodeDeleted {
				glog.V(0).Info("Shutting down due to node delete ", serviceID)
				shutdownServiceInstances(l.conn, serviceStates, len(serviceStates))
				return
			}
			glog.V(1).Infof("Service %s received event: %v", service.Name, evt)
			continue

		case evt := <-childEvent:
			glog.V(1).Infof("Service %s received child event: %v", service.Name, evt)
			continue

		case <-shutdown:
			glog.V(1).Info("Leader stopping watch on ", service.Name)
			return

		}
	}

}

func (l *leader) updateServiceInstances(service *dao.Service, serviceStates []*dao.ServiceState) error {
	//	var err error
	// pick services instances to start
	if len(serviceStates) < service.Instances {
		instancesToStart := service.Instances - len(serviceStates)
		glog.V(2).Infof("updateServiceInstances wants to start %d instances", instancesToStart)
		hosts, err := l.facade.FindHostsInPool(l.context, service.PoolId)
		if err != nil {
			glog.Errorf("Leader unable to acquire hosts for pool %s: %v", service.PoolId, err)
			return err
		}
		if len(hosts) == 0 {
			glog.Warningf("Pool %s has no hosts", service.PoolId)
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

// getFreeInstanceIds looks up running instances of this service and returns n
// unused instance ids.
// Note: getFreeInstanceIds does NOT validate that instance ids do not exceed
// max number of instances for the service. We're already doing that check in
// another, better place. It is guaranteed that either nil or n ids will be
// returned.
func getFreeInstanceIds(conn coordclient.Connection, svc *dao.Service, n int) ([]int, error) {
	var (
		states []*dao.ServiceState
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
		used[s.InstanceId] = struct{}{}
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
func (l *leader) startServiceInstances(service *dao.Service, hosts []*host.Host, numToStart int) error {
	glog.V(1).Infof("Starting %d instances, choosing from %d hosts", numToStart, len(hosts))

	// Get numToStart free instance ids
	freeids, err := getFreeInstanceIds(l.conn, service, numToStart)
	if err != nil {
		return err
	}

	hostPolicy := NewServiceHostPolicy(service, l.dao)

	// Start up an instance per id
	for _, i := range freeids {
		servicehost, err := hostPolicy.SelectHost(hosts)
		if err != nil {
			return err
		}

		glog.V(2).Info("Selected host ", servicehost)
		serviceState, err := service.NewServiceState(servicehost.ID)
		if err != nil {
			glog.Errorf("Error creating ServiceState instance: %v", err)
			return err
		}

		serviceState.HostIp = servicehost.IPAddr
		serviceState.InstanceId = i
		err = zzk.AddServiceState(l.conn, serviceState)
		if err != nil {
			glog.Errorf("Leader unable to add service state: %v", err)
			return err
		}
		glog.V(2).Info("Started ", serviceState)
	}
	return nil
}

func shutdownServiceInstances(conn coordclient.Connection, serviceStates []*dao.ServiceState, numToKill int) {
	glog.V(1).Infof("Stopping %d instances from %d total", numToKill, len(serviceStates))
	for i := 0; i < numToKill; i++ {
		glog.V(2).Infof("Killing host service state %s:%s\n", serviceStates[i].HostId, serviceStates[i].Id)
		serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
		err := zzk.TerminateHostService(conn, serviceStates[i].HostId, serviceStates[i].Id)
		if err != nil {
			glog.Warningf("%s:%s wouldn't die", serviceStates[i].HostId, serviceStates[i].Id)
		}
	}
}

// selectPoolHostForService chooses a host from the pool for the specified service. If the service
// has an address assignment the host will already be selected. If not the host with the least amount
// of memory committed to running containers will be chosen.
func (l *leader) selectPoolHostForService(s *dao.Service, hosts []*host.Host) (*host.Host, error) {
	var hostid string
	for _, ep := range s.Endpoints {
		if ep.AddressAssignment != (servicedefinition.AddressAssignment{}) {
			hostid = ep.AddressAssignment.HostID
			break
		}
	}

	if hostid != "" {
		return poolHostFromAddressAssignments(hostid, hosts)
	}

	return NewServiceHostPolicy(s, l.dao).SelectHost(hosts)
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
