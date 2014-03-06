package zzk

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"encoding/json"
	"errors"
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

	return AddService(conn, service)
}

func AddService(conn *zk.Conn, service *dao.Service) error {
	glog.V(2).Infof("Creating new service %s", service.Id)

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

	glog.V(2).Infof("Successfully created %s", servicePath)
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
		glog.V(0).Infof("Unexpectedly could not retrieve %s", servicePath)
		err = AddService(conn, service)
		return err
	}
	glog.V(4).Infof("ZkDao.UpdateService %v, %v", servicePath, service)

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

func (z *ZkDao) RemoveService(id string) error {
	conn, _, err := zk.Connect(z.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	return RemoveService(conn, id)
}

func RemoveService(conn *zk.Conn, id string) error {
	glog.V(2).Infof("zkdao.RemoveService: %s - begin", id)
	defer glog.V(2).Infof("zkdao.RemoveService: %s - complete", id)

	servicePath := ServicePath(id)

	// First mark the service as needing to shutdown so the scheduler
	// doesn't keep trying to schedule new instances
	err := loadAndUpdateService(conn, id, func(s *dao.Service) {
		s.DesiredState = dao.SVC_STOP
	})
	if err != nil {
		return err
	} // Error already logged

	children, _, zke, err := conn.ChildrenW(servicePath)
	for ; err == nil && len(children) > 0; children, _, zke, err = conn.ChildrenW(servicePath) {

		select {

		case evt := <-zke:
			glog.V(1).Infof("RemoveService saw ZK event: %v", evt)
			continue

		case <-time.After(30 * time.Second):
			glog.V(0).Infof("Gave up deleting %s with %d children", servicePath, len(children))
			return errors.New("Timed out waiting for children to die for " + servicePath)
		}
	}
	if err != nil {
		glog.Errorf("Unable to get children for %s: %v", id, err)
		return err
	}

	var service dao.Service
	stats, err := LoadService(conn, id, &service)
	if err != nil {
		// Error already logged
		return err
	}
	err = conn.Delete(servicePath, stats.Version)
	if err != nil {
		glog.Errorf("Unable to delete service %s because: %v", servicePath, err)
		return err
	}
	glog.V(1).Infof("Service %s removed", servicePath)

	return nil
}

func RemoveServiceState(conn *zk.Conn, serviceId string, serviceStateId string) error {
	ssPath := ServiceStatePath(serviceId, serviceStateId)

	var ss dao.ServiceState
	stats, err := LoadServiceState(conn, serviceId, serviceStateId, &ss)
	if err != nil {
		return err
	} // Error already logged

	err = conn.Delete(ssPath, stats.Version)
	if err != nil {
		glog.Errorf("Unable to delete service state %s because: %v", ssPath, err)
		return err
	}

	hssPath := HostServiceStatePath(ss.HostId, serviceStateId)
	_, stats, err = conn.Get(hssPath)
	if err != nil {
		glog.Errorf("Unable to get host service state %s for delete because: %v", hssPath, err)
		return err
	}

	err = conn.Delete(hssPath, stats.Version)
	if err != nil {
		glog.Errorf("Unable to delete host service state %s", hssPath)
		return err
	}
	return nil
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

type serviceMutator func(*dao.Service)
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
	ssBytes, err := json.Marshal(&ss)
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

func loadAndUpdateService(conn *zk.Conn, serviceId string, mutator serviceMutator) error {
	servicePath := ServicePath(serviceId)
	var service dao.Service

	serviceNode, stats, err := conn.Get(servicePath)
	if err != nil {
		glog.Errorf("Unable to find data %s: %v", servicePath, err)
		return err
	}
	err = json.Unmarshal(serviceNode, &service)
	if err != nil {
		glog.Errorf("Unable to unmarshal %s: %v", servicePath, err)
		return err
	}

	mutator(&service)
	serviceBytes, err := json.Marshal(service)
	if err != nil {
		glog.Errorf("Unable to marshal %s: %v", servicePath, err)
		return err
	}
	_, err = conn.Set(servicePath, serviceBytes, stats.Version)
	if err != nil {
		glog.Errorf("Unable to update service %s: %v", servicePath, err)
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

// Snapshot section start
func SnapshotRequestsPath(requestId string) string {
	return SNAPSHOT_REQUEST_PATH + "/" + requestId
}

func (this *ZkDao) AddSnapshotRequest(snapshotRequest *dao.SnapshotRequest) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	return AddSnapshotRequest(conn, snapshotRequest)
}

func AddSnapshotRequest(conn *zk.Conn, snapshotRequest *dao.SnapshotRequest) error {
	glog.V(3).Infof("Creating new snapshot request %s", snapshotRequest.Id)

	// make sure toplevel paths exist
	paths := []string{SNAPSHOT_PATH, SNAPSHOT_REQUEST_PATH}
	for _, path := range paths {
		exists, _, _, err := conn.ExistsW(path)
		if err != nil {
			if err == zk.ErrNoNode {
				CreateNode(path, conn)
			}
		}
		if !exists {
			CreateNode(path, conn)
		}
	}

	// add the request to the snapshot request path
	snapshotRequestsPath := SnapshotRequestsPath(snapshotRequest.Id)
	sBytes, err := json.Marshal(snapshotRequest)
	if err != nil {
		glog.Errorf("Unable to marshal data for snapshot request %s", snapshotRequestsPath)
		return err
	}

	_, err = conn.Create(snapshotRequestsPath, sBytes, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		glog.Errorf("Unable to create snapshot request %s: %v", snapshotRequestsPath, err)
	}

	glog.V(3).Infof("Successfully created snapshot request %s", snapshotRequestsPath)
	return err
}

func (z *ZkDao) LoadSnapshotRequest(requestId string, sr *dao.SnapshotRequest) (*zk.Stat, error) {
	conn, _, err := zk.Connect(z.Zookeepers, time.Second*10)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return LoadSnapshotRequest(conn, requestId, sr)
}

func LoadSnapshotRequest(conn *zk.Conn, requestId string, sr *dao.SnapshotRequest) (*zk.Stat, error) {
	sBytes, ssStat, err := conn.Get(SnapshotRequestsPath(requestId))
	if err != nil {
		glog.Errorf("Unable to retrieve snapshot request %s: %v", requestId, err)
		return nil, err
	}
	err = json.Unmarshal(sBytes, &sr)
	if err != nil {
		glog.Errorf("Unable to unmarshal snapshot request %s: %v", requestId, err)
		return nil, err
	}
	return ssStat, nil
}

func (z *ZkDao) LoadSnapshotRequestW(requestId string, sr *dao.SnapshotRequest) (*zk.Stat, <-chan zk.Event, error) {
	conn, _, err := zk.Connect(z.Zookeepers, time.Second*10)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	return LoadSnapshotRequestW(conn, requestId, sr)
}

func LoadSnapshotRequestW(conn *zk.Conn, requestId string, sr *dao.SnapshotRequest) (*zk.Stat, <-chan zk.Event, error) {
	sBytes, ssStat, event, err := conn.GetW(SnapshotRequestsPath(requestId))
	if err != nil {
		glog.Errorf("Unable to retrieve snapshot request %s: %v", requestId, err)
		return nil, nil, err
	}
	err = json.Unmarshal(sBytes, &sr)
	if err != nil {
		glog.Errorf("Unable to unmarshal snapshot request %s: %v", requestId, err)
		return nil, nil, err
	}
	return ssStat, event, nil
}

func (this *ZkDao) UpdateSnapshotRequest(snapshotRequest *dao.SnapshotRequest) error {
	conn, _, err := zk.Connect(this.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	return UpdateSnapshotRequest(conn, snapshotRequest)
}

func UpdateSnapshotRequest(conn *zk.Conn, snapshotRequest *dao.SnapshotRequest) error {
	ssBytes, err := json.Marshal(snapshotRequest)
	if err != nil {
		return err
	}

	snapshotRequestsPath := SnapshotRequestsPath(snapshotRequest.Id)
	exists, _, _, err := conn.ExistsW(snapshotRequestsPath)
	if err != nil {
		if err == zk.ErrNoNode {
			return AddSnapshotRequest(conn, snapshotRequest)
		}
	}
	if !exists {
		return AddSnapshotRequest(conn, snapshotRequest)
	}

	_, stats, err := conn.Get(snapshotRequestsPath)
	if err != nil {
		return err
	}
	_, err = conn.Set(snapshotRequestsPath, ssBytes, stats.Version)
	return err
}

func (z *ZkDao) RemoveSnapshotRequest(requestId string) error {
	conn, _, err := zk.Connect(z.Zookeepers, time.Second*10)
	if err != nil {
		return err
	}
	defer conn.Close()

	return RemoveSnapshotRequest(conn, requestId)
}

func RemoveSnapshotRequest(conn *zk.Conn, requestId string) error {
	snapshotRequestsPath := SnapshotRequestsPath(requestId)
	_, stats, err := conn.Get(snapshotRequestsPath)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil
		}
		glog.Errorf("Unable to retrieve SnapshotRequest znode:%s error:%v", snapshotRequestsPath, err)
		return err
	}

	err = conn.Delete(snapshotRequestsPath, stats.Version)
	if err != nil {
		glog.Errorf("Unable to delete SnapshotRequest znode:%s error:%v", snapshotRequestsPath, err)
		return err
	}

	return nil
}

// Snapshot section end
