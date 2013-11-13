package rsched

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/zzk"
	"github.com/zenoss/serviced/dao"

	"time"
	"math/rand"
)

func Lead(dao dao.ControlPlane, conn *zk.Conn, zkEvent <-chan zk.Event) {
	shutdown_mode := false
	for {
		if shutdown_mode {
			glog.V(1).Info("Shutdown mode encountered.")
			break
		}
		time.Sleep(time.Second)
		func() error {
			select {
			case evt := <-zkEvent:
				// shut this thing down
				shutdown_mode = true
				glog.V(0).Info("Got a zkevent, leaving lead: ", evt)
				return nil
			default:
				glog.V(0).Info("Processing leader duties")
				// passthru
			}

			watchServices(dao, conn)
			return nil
		}()
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
		for _, serviceId := range serviceIds {
			if processing[serviceId] == nil {
				glog.V(2).Info("Leader starting goroutine to watch ", serviceId)
				serviceChannel := make(chan int)
				processing[serviceId] = serviceChannel
				go watchService(cpDao, conn, serviceChannel, sDone, serviceId)
			}
		}
		select {
		case evt := <-zkEvent:
			glog.V(1).Info("Leader event: ", evt)
		case serviceId := <-sDone:
			glog.V(1).Info("Leading cleaning up for service ", serviceId)
			delete(processing, serviceId)
		}
	}
}

func watchService(cpDao dao.ControlPlane, conn *zk.Conn, shutdown <-chan int, done chan<- string, serviceId string) {
	defer func() {
		glog.V(3).Info("Exiting function watchService ", serviceId)
		done <- serviceId 
	}()
	for {
		var service dao.Service
		_, zkEvent, err := zzk.LoadServiceW(conn, serviceId, &service)
		if err != nil {
			glog.Errorf("Unable to load service %s: %v", serviceId, err)
			return
		}
		_, _, childEvent, err := conn.ChildrenW(zzk.ServicePath(serviceId))

		glog.V(1).Info("Leader watching for changes to service ", service.Name)

		// check current state
		var serviceStates []*dao.ServiceState
		err = zzk.GetServiceStates(conn, &serviceStates, serviceId)
		if err != nil {
			glog.Error("Unable to retrieve running service states: ", err)
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
				glog.V(0).Info("Shutting down due to node delete ", serviceId)
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

		return startServiceInstances(conn, service, poolHosts, instancesToStart)

	} else if len(serviceStates) > service.Instances {
		instancesToKill := len(serviceStates) - service.Instances
		glog.V(2).Infof("updateServiceInstances wants to kill %d instances", instancesToKill)
		shutdownServiceInstances(conn, serviceStates, instancesToKill)
	}
	return nil

}

func startServiceInstances(conn *zk.Conn, service *dao.Service, pool_hosts []*dao.PoolHost, numToStart int) error {
	glog.V(1).Infof("Starting %d instances, choosing from %d hosts", numToStart, len(pool_hosts))
	for i := 0; i < numToStart; i++ {
		// randomly select host
		service_host := pool_hosts[rand.Intn(len(pool_hosts))]
		glog.V(2).Info("Selected host ", service_host)
		serviceState, err := service.NewServiceState(service_host.HostId)
		if err != nil {
			glog.Errorf("Error creating ServiceState instance: %v", err)
			return err
		}

		serviceState.HostIp = service_host.HostIp
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
