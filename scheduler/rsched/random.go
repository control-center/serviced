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
			break
		}
		time.Sleep(time.Second)
		func() error {
			select {
			case evt := <-zkEvent:
				// shut this thing down
				shutdown_mode = true
				glog.Errorf("Got a zkevent, leaving lead: %v", evt)
				return nil
			default:
				glog.Info("Processing leader duties")
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
		for _, shutdown := range processing {
			shutdown <- 1
		}
	}()

	for {
		glog.Infof("Leader watching for changes to %s", zzk.SERVICE_PATH)
		serviceIds, _, zkEvent, err := conn.ChildrenW(zzk.SERVICE_PATH)
		if err != nil {
			glog.Errorf("Leader unable to find any services: %v", err)
			return
		}
		for _, serviceId := range serviceIds {
			if processing[serviceId] == nil {
				serviceChannel := make(chan int)
				processing[serviceId] = serviceChannel
				go watchService(cpDao, conn, serviceChannel, sDone, serviceId)
			}
		}
		select {
		case evt := <-zkEvent:
			glog.Infof("Leader event: %v", evt)
		case serviceId := <-sDone:
			glog.Infof("Leading cleaning up for service %s", serviceId)
			delete(processing, serviceId)
		}
	}
}

func watchService(cpDao dao.ControlPlane, conn *zk.Conn, shutdown <-chan int, done chan<- string, serviceId string) {
	defer func() { done <- serviceId }()
	for {
		var service dao.Service
		_, zkEvent, err := zzk.LoadServiceW(conn, serviceId, &service)
		if err != nil {
			glog.Errorf("Unable to load service %s: %v", serviceId, err)
			return
		}

		glog.Infof("Leader watching for changes to service %s", service.Name)

		// check current state
		var serviceStates []*dao.ServiceState
		err = zzk.GetServiceStates(conn, &serviceStates, serviceId)
		if err != nil {
			glog.Errorf("Unable to retrieve running service states: %v", err)
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
				glog.Infof("Shutting down due to node delete %s", serviceId)
				shutdownServiceInstances(conn, serviceStates, len(serviceStates))
				return
			}
			glog.Infof("Service %s received event %v", service.Name, evt)
			continue

		case <-shutdown:
			glog.Info("Leader stopping watch on %s", service.Name)
			return

		}
	}

}

func updateServiceInstances(cpDao dao.ControlPlane, conn *zk.Conn, service *dao.Service, serviceStates []*dao.ServiceState) error {
	var err error
	// pick services instances to start
	if len(serviceStates) < service.Instances {
		instancesToStart := service.Instances - len(serviceStates)
		var poolHosts []*dao.PoolHost
		err = cpDao.GetHostsForResourcePool(service.PoolId, &poolHosts)
		if err != nil {
			glog.Errorf("Leader unable to acquire hosts for pool %s", service.PoolId)
			return err
		}
		if len(poolHosts) == 0 {
			glog.Warningf("Pool %s has no hosts", service.PoolId)
			return nil
		}

		return startServiceInstances(conn, service, poolHosts, instancesToStart)

	} else if len(serviceStates) > service.Instances {
		instancesToKill := len(serviceStates) - service.Instances
		shutdownServiceInstances(conn, serviceStates, instancesToKill)
	}
	return nil

}

func startServiceInstances(conn *zk.Conn, service *dao.Service, pool_hosts []*dao.PoolHost, numToStart int) error {
	for i := 0; i < numToStart; i++ {
		// randomly select host
		service_host := pool_hosts[rand.Intn(len(pool_hosts))]
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
		glog.Infof("cp: serviceState %s", serviceState.Started)
	}
	return nil
}

func shutdownServiceInstances(conn *zk.Conn, serviceStates []*dao.ServiceState, numToKill int) {
	for i := 0; i < numToKill; i++ {
		glog.Infof("Killing host service state %s:%s\n", serviceStates[i].HostId, serviceStates[i].Id)
		serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
		err := zzk.TerminateHostService(conn, serviceStates[i].HostId, serviceStates[i].Id)
		if err != nil {
			glog.Warningf("%s:%s wouldn't die", serviceStates[i].HostId, serviceStates[i].Id)
		}
	}
}
