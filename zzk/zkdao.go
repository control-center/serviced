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
func TerminateHostService(conn *zk.Conn, hostId string, serviceStateId string) error {
	return loadAndUpdateHss(conn, hostId, serviceStateId, func(hss *HostServiceState) {
		hss.DesiredState = dao.SVC_STOP
	})
}

func ResetServiceState(conn *zk.Conn, serviceId string, serviceStateId string) error {
	return LoadAndUpdateServiceState(conn, serviceId, serviceStateId, func(ss *dao.ServiceState) {
		ss.Terminated = time.Now()
	})
}

// Communicates to the agent that this service instance should stop
func (this *ZkDao) TerminateHostService(hostId string, serviceStateId string) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		glog.Errorf("Unable to connect to zookeeper: %v", err)
		return err
	}
	defer conn.Close()

	return TerminateHostService(conn, hostId, serviceStateId)
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

	return AddServiceState(conn, state)
}

func AddServiceState(conn *zk.Conn, state *dao.ServiceState) error {
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
	hssBytes, err := json.Marshal(SsToHss(state))
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
	return GetServiceState(conn, serviceState, serviceId, serviceStateId)
}

func GetServiceState(conn *zk.Conn, serviceState *dao.ServiceState, serviceId string, serviceStateId string) error {
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

	return GetServiceStates(conn, serviceStates, serviceIds...)
}

func GetServiceStates(conn *zk.Conn, serviceStates *[]*dao.ServiceState, serviceIds ...string) error {
	for _, serviceId := range serviceIds {
		err := appendServiceStates(conn, serviceId, serviceStates)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *ZkDao) GetRunningService(serviceId string, serviceStateId string, running *dao.RunningService) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	var s dao.Service
	_, err = LoadService(conn, serviceId, &s)
	if err != nil {
		return err
	}

	var ss dao.ServiceState
	_, err = LoadServiceState(conn, serviceId, serviceStateId, &ss)
	if err != nil {
		return err
	}
	*running = *sssToRs(&s, &ss)
	return nil
}

func (this *ZkDao) GetRunningServicesForHost(hostId string, running *[]*dao.RunningService) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	serviceStateIds, _, err := conn.Children(HostPath(hostId))
	if err != nil {
		glog.Errorf("Unable to acquire list of services")
		return err
	}

	_ss := make([]*dao.RunningService, len(serviceStateIds))
	for i, hssId := range serviceStateIds {

		var hss HostServiceState
		_, err = LoadHostServiceState(conn, hostId, hssId, &hss)
		if err != nil {
			return err
		}

		var s dao.Service
		_, err = LoadService(conn, hss.ServiceId, &s)
		if err != nil {
			return err
		}

		var ss dao.ServiceState
		_, err = LoadServiceState(conn, hss.ServiceId, hss.ServiceStateId, &ss)
		if err != nil {
			return err
		}
		_ss[i] = sssToRs(&s, &ss)
	}
	*running = append(*running, _ss...)
	return nil
}

func (this *ZkDao) GetRunningServicesForService(serviceId string, running *[]*dao.RunningService) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	return LoadRunningServices(conn, running, serviceId)
}

func (this *ZkDao) GetAllRunningServices(running *[]*dao.RunningService) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	serviceIds, _, err := conn.Children(SERVICE_PATH)
	if err != nil {
		glog.Errorf("Unable to acquire list of services")
		return err
	}
	return LoadRunningServices(conn, running, serviceIds...)
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

func LoadRunningServices(conn *zk.Conn, running *[]*dao.RunningService, serviceIds ...string) error {
	for _, serviceId := range serviceIds {
		var s dao.Service
		_, err := LoadService(conn, serviceId, &s)
		if err != nil {
			return err
		}

		servicePath := ServicePath(serviceId)
		childNodes, _, err := conn.Children(servicePath)
		if err != nil {
			return err
		}

		_ss := make([]*dao.RunningService, len(childNodes))
		for i, childId := range childNodes {
			var ss dao.ServiceState
			_, err = LoadServiceState(conn, serviceId, childId, &ss)
			if err != nil {
				return err
			}
			_ss[i] = sssToRs(&s, &ss)

		}
		*running = append(*running, _ss...)
	}
	return nil
}

func LoadHostServiceState(conn *zk.Conn, hostId string, hssId string, hss *HostServiceState) (*zk.Stat, error) {
	hssPath := HostServiceStatePath(hostId, hssId)
	hssBytes, hssStat, err := conn.Get(hssPath)
	if err != nil {
		glog.Errorf("Unable to retrieve host service state %s: %v", hssPath, err)
		return nil, err
	}

	err = json.Unmarshal(hssBytes, &hss)
	if err != nil {
		glog.Errorf("Unable to unmarshal %s", hssPath)
		return nil, err
	}
	return hssStat, nil
}

func LoadHostServiceStateW(conn *zk.Conn, hostId string, hssId string, hss *HostServiceState) (*zk.Stat, <-chan zk.Event, error) {
	hssPath := HostServiceStatePath(hostId, hssId)
	hssBytes, hssStat, event, err := conn.GetW(hssPath)
	if err != nil {
		glog.Errorf("Unable to retrieve host service state %s: %v", hssPath, err)
		return nil, nil, err
	}

	err = json.Unmarshal(hssBytes, &hss)
	if err != nil {
		glog.Errorf("Unable to unmarshal %s", hssPath)
		return nil, nil, err
	}
	return hssStat, event, nil
}

func LoadService(conn *zk.Conn, serviceId string, s *dao.Service) (*zk.Stat, error) {
	sBytes, ssStat, err := conn.Get(ServicePath(serviceId))
	if err != nil {
		glog.Errorf("Unable to retrieve service %s: %v", serviceId, err)
		return nil, err
	}
	err = json.Unmarshal(sBytes, &s)
	if err != nil {
		glog.Errorf("Unable to unmarshal service %s: %v", serviceId, err)
		return nil, err
	}
	return ssStat, nil
}

func LoadServiceW(conn *zk.Conn, serviceId string, s *dao.Service) (*zk.Stat, <-chan zk.Event, error) {
	sBytes, ssStat, event, err := conn.GetW(ServicePath(serviceId))
	if err != nil {
		glog.Errorf("Unable to retrieve service %s: %v", serviceId, err)
		return nil, nil, err
	}
	err = json.Unmarshal(sBytes, &s)
	if err != nil {
		glog.Errorf("Unable to unmarshal service %s: %v", serviceId, err)
		return nil, nil, err
	}
	return ssStat, event, nil
}

func LoadServiceState(conn *zk.Conn, serviceId string, serviceStateId string, ss *dao.ServiceState) (*zk.Stat, error) {
	ssPath := ServiceStatePath(serviceId, serviceStateId)
	ssBytes, ssStat, err := conn.Get(ssPath)
	if err != nil {
		glog.Errorf("Got error for %s: %v", ssPath, err)
		return nil, err
	}
	err = json.Unmarshal(ssBytes, &ss)
	if err != nil {
		glog.Errorf("Unable to unmarshal %s", ssPath)
		return nil, err
	}
	return ssStat, nil
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
	return nil
}

type hssMutator func(*HostServiceState)
type ssMutator func(*dao.ServiceState)

func LoadAndUpdateServiceState(conn *zk.Conn, serviceId string, ssId string, mutator ssMutator) error {
	ssPath := ServiceStatePath(serviceId, ssId)
	var ss dao.ServiceState

	serviceStateNode, stats, err := conn.Get(ssPath)
	if err != nil {
		// Should it really be an error if we can't find anything?
		glog.Errorf("Unable to find data %s: %v", ssPath, err)
		return err
	}
	err = json.Unmarshal(serviceStateNode, &ss)
	if err != nil {
		glog.Errorf("Unable to unmarshal %s: %v", ssPath, err)
		return err
	}

	mutator(&ss)
	ssBytes, err := json.Marshal(ss)
	if err != nil {
		glog.Errorf("Unable to marshal %s: %v", ssPath, err)
		return err
	}
	_, err = conn.Set(ssPath, ssBytes, stats.Version)
	if err != nil {
		glog.Errorf("Unable to update service state %s: %v", ssPath, err)
		return err
	}
	return nil
}


func loadAndUpdateHss(conn *zk.Conn, hostId string, hssId string, mutator hssMutator) error {
	hssPath := HostServiceStatePath(hostId, hssId)
	var hss HostServiceState

	hostStateNode, stats, err := conn.Get(hssPath)
	if err != nil {
		// Should it really be an error if we can't find anything?
		glog.Errorf("Unable to find data %s: %v", hssPath, err)
		return err
	}
	err = json.Unmarshal(hostStateNode, &hss)
	if err != nil {
		glog.Errorf("Unable to unmarshal %s: %v", hssPath, err)
		return err
	}

	mutator(&hss)
	hssBytes, err := json.Marshal(hss)
	if err != nil {
		glog.Errorf("Unable to marshal %s: %v", hssPath, err)
		return err
	}
	_, err = conn.Set(hssPath, hssBytes, stats.Version)
	if err != nil {
		glog.Errorf("Unable to update host service state %s: %v", hssPath, err)
		return err
	}
	return nil
}

// ServiceState to HostServiceState
func SsToHss(ss *dao.ServiceState) *HostServiceState {
	return &HostServiceState{ss.HostId, ss.ServiceId, ss.Id, dao.SVC_RUN}
}

// Service & ServiceState to RunningService
func sssToRs(s *dao.Service, ss *dao.ServiceState) *dao.RunningService {
	rs := &dao.RunningService{}
	rs.Id = ss.Id
	rs.ServiceId = ss.ServiceId
	rs.StartedAt = ss.Started
	rs.HostId = ss.HostId
	rs.DockerId = ss.DockerId
	rs.Startup = s.Startup
	rs.Name = s.Name
	rs.Description = s.Description
	rs.Instances = s.Instances
	rs.PoolId = s.PoolId
	rs.ImageId = s.ImageId
	rs.DesiredState = s.DesiredState
	rs.ParentServiceId = s.ParentServiceId
	return rs
}
