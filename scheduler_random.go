package serviced

import (
	"container/heap"
	"fmt"
	"path"
	"sync"

	"time"

	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/zzk"
)

// Lead is executed by the "leader" of the control plane cluster to handle its
// service/snapshot management responsibilities.
func Lead(dao dao.ControlPlane, conn *zk.Conn, zkEvent <-chan zk.Event) {
	shutdownmode := false
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

			go watchSnapshotRequests(dao, conn)
			watchServices(dao, conn)
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

func watchSnapshotRequests(cpDao dao.ControlPlane, conn *zk.Conn) {
	glog.V(3).Info("started watchSnapshotRequestss")
	defer glog.V(3).Info("finished watchSnapshotRequestss")

	// make sure toplevel paths exist
	paths := []string{zzk.SNAPSHOT_PATH, zzk.SNAPSHOT_REQUEST_PATH}
	for _, path := range paths {
		exists, _, _, err := conn.ExistsW(path)
		if err != nil {
			if err == zk.ErrNoNode {
				if err := zzk.CreateNode(path, conn); err != nil {
					glog.Errorf("Leader unable to create znode:%s error: %s", path, err)
					return
				}
			} else {
				glog.Errorf("Leader unable to get status for znode:%s error: %s", path, err)
				return
			}
		}
		if !exists {
			if err := zzk.CreateNode(path, conn); err != nil {
				glog.Errorf("Leader unable to create znode:%s error: %s", path, err)
				return
			}
		}
	}

	// watch for snapshot requests and perform snapshots
	glog.V(0).Info("Leader watching for snapshot requests to ", zzk.SNAPSHOT_REQUEST_PATH)
	for {
		requestIds, _, zkEvent, err := conn.ChildrenW(zzk.SNAPSHOT_REQUEST_PATH)
		if err != nil {
			glog.Errorf("Leader unable to watch for snapshot requests to %s error: %s", zzk.SNAPSHOT_REQUEST_PATH, err)
			return
		}
		for _, requestID := range requestIds {
			snapshotRequest := dao.SnapshotRequest{}
			if _, err := zzk.LoadSnapshotRequest(conn, requestID, &snapshotRequest); err != nil {
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

func watchServices(cpDao dao.ControlPlane, conn *zk.Conn) {
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

	for {
		glog.V(1).Info("Leader watching for changes to ", zzk.SERVICE_PATH)
		serviceIds, _, zkEvent, err := conn.ChildrenW(zzk.SERVICE_PATH)
		if err != nil {
			glog.Errorf("Leader unable to find any services: ", err)
			return
		}
		for _, serviceID := range serviceIds {
			if processing[serviceID] == nil {
				glog.V(2).Info("Leader starting goroutine to watch ", serviceID)
				serviceChannel := make(chan int)
				processing[serviceID] = serviceChannel
				go watchService(cpDao, conn, serviceChannel, sDone, serviceID)
			}
		}
		select {
		case evt := <-zkEvent:
			glog.V(1).Info("Leader event: ", evt)
		case serviceID := <-sDone:
			glog.V(1).Info("Leading cleaning up for service ", serviceID)
			delete(processing, serviceID)
		}
	}
}

func watchService(cpDao dao.ControlPlane, conn *zk.Conn, shutdown <-chan int, done chan<- string, serviceID string) {
	defer func() {
		glog.V(3).Info("Exiting function watchService ", serviceID)
		done <- serviceID
	}()
	for {
		var service dao.Service
		_, zkEvent, err := zzk.LoadServiceW(conn, serviceID, &service)
		if err != nil {
			glog.Errorf("Unable to load service %s: %v", serviceID, err)
			return
		}
		_, _, childEvent, err := conn.ChildrenW(zzk.ServicePath(serviceID))

		glog.V(1).Info("Leader watching for changes to service ", service.Name)

		switch exists, _, err := conn.Exists(path.Join("/services", serviceID)); {
		case err != nil:
			glog.Errorf("conn.Exists failed (%v)", err)
			return
		case exists == false:
			glog.V(2).Infof("no /service node for: %s", serviceID)
			return
		}

		// check current state
		var serviceStates []*dao.ServiceState
		err = zzk.GetServiceStates(conn, &serviceStates, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return
		}

		// Is the service supposed to be running at all?
		switch {
		case service.DesiredState == dao.SVC_STOP:
			shutdownServiceInstances(conn, serviceStates, len(serviceStates))
		case service.DesiredState == dao.SVC_RUN:
			updateServiceInstances(cpDao, conn, &service, serviceStates)
		default:
			glog.Warningf("Unexpected desired state %d for service %s", service.DesiredState, service.Name)
		}

		select {
		case evt := <-zkEvent:
			if evt.Type == zk.EventNodeDeleted {
				glog.V(0).Info("Shutting down due to node delete ", serviceID)
				shutdownServiceInstances(conn, serviceStates, len(serviceStates))
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

func updateServiceInstances(cpDao dao.ControlPlane, conn *zk.Conn, service *dao.Service, serviceStates []*dao.ServiceState) error {
	var err error
	// pick services instances to start
	if len(serviceStates) < service.Instances {
		instancesToStart := service.Instances - len(serviceStates)
		glog.V(2).Infof("updateServiceInstances wants to start %d instances", instancesToStart)
		var poolHosts []*dao.PoolHost
		err = cpDao.GetHostsForResourcePool(service.PoolId, &poolHosts)
		if err != nil {
			glog.Errorf("Leader unable to acquire hosts for pool %s: %v", service.PoolId, err)
			return err
		}
		if len(poolHosts) == 0 {
			glog.Warningf("Pool %s has no hosts", service.PoolId)
			return nil
		}

		return startServiceInstances(cpDao, conn, service, poolHosts, instancesToStart)

	} else if len(serviceStates) > service.Instances {
		instancesToKill := len(serviceStates) - service.Instances
		glog.V(2).Infof("updateServiceInstances wants to kill %d instances", instancesToKill)
		shutdownServiceInstances(conn, serviceStates, instancesToKill)
	}
	return nil

}

func startServiceInstances(cpDao dao.ControlPlane, conn *zk.Conn, service *dao.Service, poolhosts []*dao.PoolHost, numToStart int) error {
	glog.V(1).Infof("Starting %d instances, choosing from %d hosts", numToStart, len(poolhosts))
	for i := 0; i < numToStart; i++ {
		servicehost, err := selectPoolHostForService(cpDao, service, poolhosts)
		if err != nil {
			return err
		}

		glog.V(2).Info("Selected host ", servicehost)
		serviceState, err := service.NewServiceState(servicehost.HostId)
		if err != nil {
			glog.Errorf("Error creating ServiceState instance: %v", err)
			return err
		}

		serviceState.HostIp = servicehost.HostIp
		err = zzk.AddServiceState(conn, serviceState)
		if err != nil {
			glog.Errorf("Leader unable to add service state: %v", err)
			return err
		}
		glog.V(2).Info("Started ", serviceState)
	}
	return nil
}

func shutdownServiceInstances(conn *zk.Conn, serviceStates []*dao.ServiceState, numToKill int) {
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
func selectPoolHostForService(cp dao.ControlPlane, s *dao.Service, pool []*dao.PoolHost) (*dao.PoolHost, error) {
	var hostid string
	for _, ep := range s.Endpoints {
		if ep.AddressAssignment != (dao.AddressAssignment{}) {
			hostid = ep.AddressAssignment.HostId
			break
		}
	}

	if hostid != "" {
		return poolHostFromAddressAssignments(hostid, pool)
	}

	return selectLeastCommittedHost(cp, pool)
}

// poolHostFromAddressAssignments determines the pool host for the service from its address assignment(s).
func poolHostFromAddressAssignments(hostid string, pool []*dao.PoolHost) (*dao.PoolHost, error) {
	// ensure the assigned host is in the pool
	for _, ph := range pool {
		if ph.HostId == hostid {
			return ph, nil
		}
	}

	return nil, fmt.Errorf("assigned host is not in pool")
}

// hostitem is what is stored in the least commited RAM scheduler's priority queue
type hostitem struct {
	host     *dao.PoolHost
	priority uint64 // the host's available RAM
	index    int    // the index of the hostitem in the heap
}

// priorityqueue implements the heap.Interface and holds hostitems
type priorityqueue []*hostitem

// selectLeastCommittedHost choses the host with the least RAM commited to running containers. It
// uses a two stage pipeline that first calculates the available RAM per host and then adds each
// host to a queue prioritized by available RAM. The selected host is the one with the top
// priority.
func selectLeastCommittedHost(cp dao.ControlPlane, hosts []*dao.PoolHost) (*dao.PoolHost, error) {
	var wg sync.WaitGroup

	done := make(chan bool)
	defer close(done)

	hic := make(chan *hostitem)

	// fan-out available RAM computation for each host
	for _, h := range hosts {
		wg.Add(1)
		go func(poolhost *dao.PoolHost) {
			availableRAM(cp, poolhost, hic, done)
			wg.Done()
		}(h)
	}

	// close the hostitem channel when all the calculation is finished
	go func() {
		wg.Wait()
		close(hic)
	}()

	pq := &priorityqueue{}
	heap.Init(pq)

	// fan-in all the available RAM computations
	for hi := range hic {
		heap.Push(pq, hi)
	}

	// select the highest priority (most available RAM) host
	if pq.Len() <= 0 {
		return nil, fmt.Errorf("unable to find a host to schedule")
	}
	return heap.Pop(pq).(*hostitem).host, nil
}

// availableRAM computes the amount of RAM available on a given host by subtracting the sum of the
// RAM commitments of each of its running services from its total memory.
func availableRAM(cp dao.ControlPlane, host *dao.PoolHost, result chan *hostitem, done <-chan bool) {
	rss := []*dao.RunningService{}
	if err := cp.GetRunningServicesForHost(host.HostId, &rss); err != nil {
		glog.Errorf("cannot retrieve running services for host: %s (%v)", host.HostId, err)
		return // this host won't be scheduled
	}

	var cr uint64

	for i := range rss {
		s := dao.Service{}
		if err := cp.GetService(rss[i].ServiceId, &s); err != nil {
			glog.Errorf("cannot retrieve service information for running service (%v)", err)
			return // this host won't be scheduled
		}

		cr += s.RAMCommitment
	}

	h := dao.Host{}
	if err := cp.GetHost(host.HostId, &h); err != nil {
		glog.Errorf("cannot retrieve host information for pool host %s (%v)", host.HostId, err)
		return // this host won't be scheduled
	}

	result <- &hostitem{host, h.Memory - cr, -1}
}

/*
PriorityQueue implementation take from golang std library container/heap documentation example
*/

// Len is the number of elements in the collection.
func (pq priorityqueue) Len() int {
	return len(pq)
}

// Less reports whether the element with index i should sort before the element with index j.
func (pq priorityqueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

// Swap swaps the elements with indexes i and j.
func (pq priorityqueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push pushes the hostitem onto the heap.
func (pq *priorityqueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*hostitem)
	item.index = n
	*pq = append(*pq, item)
}

// Pop removes the minimum element (according to Less) from the heap and returns it.
func (pq *priorityqueue) Pop() interface{} {
	opq := *pq
	n := len(opq)
	item := opq[n-1]
	item.index = -1 // mark it as removed, just in case
	*pq = opq[0 : n-1]
	return item
}
