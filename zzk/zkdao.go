package zzk

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"encoding/json"
	"time"
)

type ZkDao struct {
	Zookeepers []string
}

type ZkConn struct {
	Conn *zk.Conn
}

type HostServiceState struct {
	HostId         string
	ServiceId      string
	ServiceStateId string
	DesiredState   int
}

// Communicates to the agent that this service instance should stop
func (this *ZkDao) TerminateHostService(hostId string, serviceStateId string) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	hostServiceStatePath := SCHEDULER_PATH + "/" + hostId + "/" + serviceStateId
	hostStateNode, stats, err := conn.Get(hostServiceStatePath)
	if err != nil {
		// Should it really be an error if we can't find anything?
		glog.Infof("Unable to find data at %s", hostServiceStatePath)
		return err
	}
	var hostServiceState HostServiceState
	err = json.Unmarshal(hostStateNode, &hostServiceState)
	if err != nil {
		return err
	}
	hostServiceState.DesiredState = dao.SVC_STOP
	hssBytes, err := json.Marshal(hostServiceState)
	if err != nil {
		return err
	}
	_, err = conn.Set(hostServiceStatePath, hssBytes, stats.Version)
	return err
}

func (this *ZkDao) AddService(service *dao.Service) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	servicePath := ServicePath(service.Id)
	sBytes, err := json.Marshal(service)
	if err != nil {
		glog.Errorf("Unable to marshal data for %s", servicePath)
		return err
	}
	_, err = conn.Create(servicePath, sBytes, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		glog.Errorf("Unable to create service for %s: %v", servicePath, err)
	}
	return err
}

func (this *ZkDao) AddServiceState(state *dao.ServiceState) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	serviceStatePath := ServiceStatePath(state.ServiceId, state.Id)
	ssBytes, err := json.Marshal(state)
	if err != nil {
		glog.Errorf("Unable to marshal data for %s", serviceStatePath)
		return err
	}
	_, err = conn.Create(serviceStatePath, ssBytes, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		glog.Errorf("Unable to create path %s because %v", serviceStatePath, err)
		return err
	}
	hostServicePath := HostServiceStatePath(state.HostId, state.Id)
	hssBytes, err := json.Marshal(ssToHss(state))
	if err != nil {
		glog.Errorf("Unable to marshal data for %s", hostServicePath)
		return err
	}
	_, err = conn.Create(hostServicePath, hssBytes, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		glog.Errorf("Unable to create path %s because %v", hostServicePath, err)
		return err
	}
	return err
}

func ssToHss(ss *dao.ServiceState) *HostServiceState {
	return &HostServiceState{ss.HostId, ss.ServiceId, ss.Id, dao.SVC_RUN}
}

func (this *ZkDao) UpdateServiceState(state *dao.ServiceState) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	ssBytes, err := json.Marshal(state)
	if err != nil {
		return err
	}

	serviceStatePath := ServiceStatePath(state.ServiceId, state.Id)
	_, stats, err := conn.Get(serviceStatePath)
	if err != nil {
		return err
	}
	_, err = conn.Set(serviceStatePath, ssBytes, stats.Version)
	return err
}

func (this *ZkDao) UpdateService(service *dao.Service) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	servicePath := ServicePath(service.Id)

	sBytes, err := json.Marshal(service)
	if err != nil {
		return err
	}

	_, stats, err := conn.Get(servicePath)
	if err != nil {
		return err
	}
	_, err = conn.Set(servicePath, sBytes, stats.Version)
	return err
}

func (this *ZkDao) GetServiceState(serviceState *dao.ServiceState, serviceId string, serviceStateId string) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()
	serviceStateNode, _, err := conn.Get(ServiceStatePath(serviceId, serviceStateId))
	if err != nil {
		return err
	}
	return json.Unmarshal(serviceStateNode, serviceState)
}

func (this *ZkDao) GetServiceStates(serviceStates *[]*dao.ServiceState, serviceIds ...string) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, serviceId := range serviceIds {
		appendServiceStates(conn, serviceId, serviceStates)
	}
	return err
}

func (this *ZkDao) GetAllRunningServices(running *[]*dao.RunningService) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	serviceIds, _, err := conn.Children(SERVICE_PATH)
	glog.Infof("Found %d services", len(serviceIds))
	if err != nil {
		glog.Errorf("Unable to acquire list of services")
		return err
	}
	for _, serviceId := range serviceIds {
		serviceNode, _, err := conn.Get(ServicePath(serviceId))
		if err != nil {
			glog.Errorf("Unable to retrieve service %s: %v", serviceId, err)
			return err
		}
		var s dao.Service
		err = json.Unmarshal(serviceNode, &s)
		if err != nil {
			glog.Errorf("Unable to unmarshal service %s: %v", serviceId, err)
			return err
		}

		servicePath := ServicePath(serviceId)
		childNodes, _, err := conn.Children(servicePath)
		if err != nil {
			return err
		}

		_ss := make([]*dao.RunningService, len(childNodes))
		for i, childId := range childNodes {
			childPath := servicePath + "/" + childId
			ssNode, _, err := conn.Get(childPath)
			if err != nil {
				glog.Errorf("Got error for %s: %v", childId, err)
				return err
			}
			var ss dao.ServiceState
			err = json.Unmarshal(ssNode, &ss)
			if err != nil {
				glog.Errorf("Unable to unmarshal %s", childId)
				return err
			}
			_ss[i] = &dao.RunningService{}
			_ss[i].Id = ss.Id
			_ss[i].ServiceId = ss.ServiceId
			_ss[i].StartedAt = ss.Started
			_ss[i].HostId = ss.HostId
			_ss[i].Startup = s.Startup
			_ss[i].Name = s.Name
			_ss[i].Description = s.Description
			_ss[i].Instances = s.Instances
			_ss[i].PoolId = s.PoolId
			_ss[i].ImageId = s.ImageId
			_ss[i].DesiredState = s.DesiredState
			_ss[i].ParentServiceId = s.ParentServiceId

		}
		*running = append(*running, _ss...)
	}
	glog.Infof("Total running: %d", len(*running))
	return err
}

func appendServiceStates(conn *zk.Conn, serviceId string, serviceStates *[]*dao.ServiceState) error {
	servicePath := ServicePath(serviceId)
	childNodes, _, err := conn.Children(servicePath)
	if err != nil {
		return err
	}
	_ss := make([]*dao.ServiceState, len(childNodes))
	for i, childId := range childNodes {
		childPath := servicePath + "/" + childId
		serviceStateNode, _, err := conn.Get(childPath)
		if err != nil {
			glog.Errorf("Got error for %s: %v", childId, err)
			return err
		}
		var serviceState dao.ServiceState
		err = json.Unmarshal(serviceStateNode, &serviceState)
		if err != nil {
			glog.Errorf("Unable to unmarshal %s", childId)
			return err
		}
		_ss[i] = &serviceState
	}
	*serviceStates = append(*serviceStates, _ss...)
	return err
}

func HostPath(hostId string) string {
	return SCHEDULER_PATH + "/" + hostId
}

func ServicePath(serviceId string) string {
	return SERVICE_PATH + "/" + serviceId
}

func ServiceStatePath(serviceId string, serviceStateId string) string {
	return SERVICE_PATH + "/" + serviceId + "/" + serviceStateId
}

func HostServiceStatePath(hostId string, serviceStateId string) string {
	return SCHEDULER_PATH + "/" + hostId + "/" + serviceStateId
}
