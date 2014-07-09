package zzk

import (
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	zkservice "github.com/zenoss/serviced/zzk/service"

	"errors"
	"time"
)

const SERVICE_PATH = "/services"
const HOSTS_PATH = "/hosts"
const SCHEDULER_PATH = "/scheduler"
const SNAPSHOT_PATH = "/snapshots"
const SNAPSHOT_REQUEST_PATH = "/snapshots/requests"

type ZkDao struct {
	client *coordclient.Client
}

func NewZkDao(client *coordclient.Client) *ZkDao {
	return &ZkDao{
		client: client,
	}
}

type ZkConn struct {
	Conn coordclient.Connection
}

// Communicates to the agent that this service instance should stop
func TerminateHostService(conn coordclient.Connection, hostId string, serviceStateId string) error {
	return loadAndUpdateHss(conn, hostId, serviceStateId, func(hss *zkservice.HostState) {
		hss.DesiredState = service.SVCStop
	})
}

func ResetServiceState(conn coordclient.Connection, serviceId string, serviceStateId string) error {
	return LoadAndUpdateServiceState(conn, serviceId, serviceStateId, func(ss *servicestate.ServiceState) {
		ss.Terminated = time.Now()
	})
}

// Communicates to the agent that this service instance should stop
func (zkdao *ZkDao) TerminateHostService(hostId string, serviceStateId string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		glog.Errorf("Unable to connect to zookeeper: %v", err)
		return err
	}
	defer conn.Close()

	return TerminateHostService(conn, hostId, serviceStateId)
}

func (zkdao *ZkDao) AddService(service *service.Service) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return AddService(conn, service)
}

func AddService(conn coordclient.Connection, service *service.Service) error {
	glog.V(2).Infof("Creating new service %s", service.ID)

	svcNode := &zkservice.ServiceNode{
		Service: service,
	}
	servicePath := ServicePath(service.ID)
	if err := conn.Create(servicePath, svcNode); err != nil {
		glog.Errorf("Unable to create service for %s: %v", servicePath, err)
	}

	glog.V(2).Infof("Successfully created %s", servicePath)
	return nil
}

func (zkdao *ZkDao) AddServiceState(state *servicestate.ServiceState) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return AddServiceState(conn, state)
}

func AddServiceState(conn coordclient.Connection, state *servicestate.ServiceState) error {
	serviceStatePath := ServiceStatePath(state.ServiceID, state.ID)

	serviceStateNode := &zkservice.ServiceStateNode{
		ServiceState: state,
	}

	if err := conn.Create(serviceStatePath, serviceStateNode); err != nil {
		glog.Errorf("Unable to create path %s because %v", serviceStatePath, err)
		return err
	}
	hostServicePath := HostServiceStatePath(state.HostID, state.ID)
	hss := SsToHss(state)
	if err := conn.Create(hostServicePath, hss); err != nil {
		glog.Errorf("Unable to create path %s because %v", hostServicePath, err)
		return err
	}
	return nil
}

func (zkdao *ZkDao) UpdateServiceState(state *servicestate.ServiceState) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	serviceStatePath := ServiceStatePath(state.ServiceID, state.ID)
	ssn := zkservice.ServiceStateNode{}
	if err := conn.Get(serviceStatePath, &ssn); err != nil {
		return err
	}
	ssn.ServiceState = state
	return conn.Set(serviceStatePath, &ssn)
}

func (zkdao *ZkDao) UpdateService(service *service.Service) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	servicePath := ServicePath(service.ID)

	sn := zkservice.ServiceNode{}
	if err := conn.Get(servicePath, &sn); err != nil {
		glog.V(3).Infof("ZkDao.UpdateService unexpectedly could not retrieve %s error: %v", servicePath, err)
		err = AddService(conn, service)
		return err
	}

	sn.Service = service
	glog.V(4).Infof("ZkDao.UpdateService %v, %v", servicePath, service)

	return conn.Set(servicePath, &sn)
}

func (zkdao *ZkDao) GetServiceState(serviceState *servicestate.ServiceState, serviceId string, serviceStateId string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()
	return GetServiceState(conn, serviceState, serviceId, serviceStateId)
}

func GetServiceState(conn coordclient.Connection, serviceState *servicestate.ServiceState, serviceId string, serviceStateId string) error {
	serviceStateNode := zkservice.ServiceStateNode{}
	err := conn.Get(ServiceStatePath(serviceId, serviceStateId), &serviceStateNode)
	if err != nil {
		return err
	}
	*serviceState = *serviceStateNode.ServiceState
	return nil
}

func (zkdao *ZkDao) GetServiceStates(serviceStates *[]*servicestate.ServiceState, serviceIds ...string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return GetServiceStates(conn, serviceStates, serviceIds...)
}

func GetServiceStates(conn coordclient.Connection, serviceStates *[]*servicestate.ServiceState, serviceIds ...string) error {
	for _, serviceId := range serviceIds {
		err := appendServiceStates(conn, serviceId, serviceStates)
		if err != nil {
			return err
		}
	}
	return nil
}

func (zkdao *ZkDao) GetRunningService(serviceId string, serviceStateId string, running *dao.RunningService) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	rs, err := zkservice.LoadRunningService(conn, serviceId, serviceStateId)
	if err != nil {
		return err
	}

	*running = *rs
	return nil
}

func (zkdao *ZkDao) RegisterHost(hostID string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()
	return zkservice.RegisterHost(conn, hostID)
}

func (zkdao *ZkDao) UnregisterHost(hostID string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()
	return zkservice.UnregisterHost(conn, hostID)
}

func (zkdao *ZkDao) RemoveHost(hostId string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.Delete(HostPath(hostId))
	if err != nil {
		return err
	}
	return nil
}

func (zkdao *ZkDao) GetRunningServicesForHost(hostId string, running *[]*dao.RunningService) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	*running, err = zkservice.LoadRunningServicesByHost(conn, hostId)
	return err
}

func (zkdao *ZkDao) GetRunningServicesForService(serviceId string, running *[]*dao.RunningService) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	*running, err = zkservice.LoadRunningServicesByService(conn, serviceId)
	return err
}

func (zkdao *ZkDao) GetAllRunningServices(running *[]*dao.RunningService) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	*running, err = zkservice.LoadRunningServices(conn)
	return err
}

func HostPath(hostId string) string {
	return HOSTS_PATH + "/" + hostId
}

func ServicePath(serviceId string) string {
	return SERVICE_PATH + "/" + serviceId
}

func ServiceStatePath(serviceId string, serviceStateId string) string {
	return SERVICE_PATH + "/" + serviceId + "/" + serviceStateId
}

func HostServiceStatePath(hostId string, serviceStateId string) string {
	return HOSTS_PATH + "/" + hostId + "/" + serviceStateId
}

func (zkdao *ZkDao) RemoveService(id string) error {
	conn, err := zkdao.client.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return RemoveService(conn, id)
}

func RemoveService(conn coordclient.Connection, id string) error {
	glog.V(2).Infof("zkdao.RemoveService: %s - begin", id)
	defer glog.V(2).Infof("zkdao.RemoveService: %s - complete", id)

	servicePath := ServicePath(id)

	// First mark the service as needing to shutdown so the scheduler
	// doesn't keep trying to schedule new instances
	err := loadAndUpdateService(conn, id, func(s *service.Service) {
		s.DesiredState = service.SVCStop
	})
	if err != nil {
		return err
	} // Error already logged

	children, zke, err := conn.ChildrenW(servicePath)
	for ; err == nil && len(children) > 0; children, zke, err = conn.ChildrenW(servicePath) {

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

	var service service.Service
	if err := LoadService(conn, id, &service); err != nil {
		// Error already logged
		return err
	}
	if err := conn.Delete(servicePath); err != nil {
		glog.Errorf("Unable to delete service %s because: %v", servicePath, err)
		return err
	}
	glog.V(1).Infof("Service %s removed", servicePath)

	return nil
}

func RemoveServiceState(conn coordclient.Connection, serviceId string, serviceStateId string) error {
	ssPath := ServiceStatePath(serviceId, serviceStateId)

	var ss servicestate.ServiceState
	if err := LoadServiceState(conn, serviceId, serviceStateId, &ss); err != nil {
		return err
	} // Error already logged

	if err := conn.Delete(ssPath); err != nil {
		glog.Errorf("Unable to delete service state %s because: %v", ssPath, err)
		return err
	}

	hssPath := HostServiceStatePath(ss.HostID, serviceStateId)
	hss := zkservice.HostState{}
	if err := conn.Get(hssPath, &hss); err != nil {
		glog.Errorf("Unable to get host service state %s for delete because: %v", hssPath, err)
		return err
	}

	if err := conn.Delete(hssPath); err != nil {
		glog.Errorf("Unable to delete host service state %s", hssPath)
		return err
	}
	return nil
}

func LoadHostServiceState(conn coordclient.Connection, hostId string, hssId string, hss *zkservice.HostState) error {
	hssPath := HostServiceStatePath(hostId, hssId)
	err := conn.Get(hssPath, hss)
	if err != nil {
		glog.Errorf("Unable to retrieve host service state %s: %v", hssPath, err)
		return err
	}
	return nil
}

func LoadHostServiceStateW(conn coordclient.Connection, hostId string, hssId string, hss *zkservice.HostState) (<-chan coordclient.Event, error) {
	hssPath := HostServiceStatePath(hostId, hssId)
	event, err := conn.GetW(hssPath, hss)
	if err != nil {
		glog.Errorf("Unable to retrieve host service state %s: %v", hssPath, err)
		return nil, err
	}
	return event, nil
}

func LoadService(conn coordclient.Connection, serviceId string, s *service.Service) error {
	sn := zkservice.ServiceNode{}
	err := conn.Get(ServicePath(serviceId), &sn)
	if err != nil {
		glog.Errorf("Unable to retrieve service %s: %v", serviceId, err)
		return err
	}
	*s = *sn.Service
	return nil
}

func LoadServiceW(conn coordclient.Connection, serviceId string, s *service.Service) (<-chan coordclient.Event, error) {
	sn := zkservice.ServiceNode{}
	event, err := conn.GetW(ServicePath(serviceId), &sn)
	if err != nil {
		//glog.Errorf("Unable to retrieve service %s: %v", serviceId, err)
		return nil, err
	}
	*s = *sn.Service
	return event, nil
}

func LoadServiceState(conn coordclient.Connection, serviceId string, serviceStateId string, ss *servicestate.ServiceState) error {
	ssPath := ServiceStatePath(serviceId, serviceStateId)
	ssn := zkservice.ServiceStateNode{}
	err := conn.Get(ssPath, &ssn)
	if err != nil {
		glog.Errorf("Got error for %s: %v", ssPath, err)
		return err
	}
	*ss = *ssn.ServiceState
	return nil
}

func appendServiceStates(conn coordclient.Connection, serviceId string, serviceStates *[]*servicestate.ServiceState) error {
	servicePath := ServicePath(serviceId)
	childNodes, err := conn.Children(servicePath)
	if err != nil {
		return err
	}
	_ss := make([]*servicestate.ServiceState, len(childNodes))
	for i, childId := range childNodes {
		childPath := servicePath + "/" + childId
		ssn := zkservice.ServiceStateNode{}
		err := conn.Get(childPath, &ssn)
		if err != nil {
			glog.Errorf("Got error for %s: %v", childId, err)
			return err
		}
		_ss[i] = ssn.ServiceState
	}
	*serviceStates = append(*serviceStates, _ss...)
	return nil
}

type serviceMutator func(*service.Service)
type hssMutator func(*zkservice.HostState)
type ssMutator func(*servicestate.ServiceState)

func LoadAndUpdateServiceState(conn coordclient.Connection, serviceId string, ssId string, mutator ssMutator) error {
	ssPath := ServiceStatePath(serviceId, ssId)

	ssn := zkservice.ServiceStateNode{}
	err := conn.Get(ssPath, &ssn)
	if err != nil {
		// Should it really be an error if we can't find anything?
		glog.Errorf("Unable to find data %s: %v", ssPath, err)
		return err
	}
	mutator(ssn.ServiceState)
	if err := conn.Set(ssPath, &ssn); err != nil {
		glog.Errorf("Unable to update service state %s: %v", ssPath, err)
		return err
	}
	return nil
}

func loadAndUpdateService(conn coordclient.Connection, serviceId string, mutator serviceMutator) error {
	servicePath := ServicePath(serviceId)

	serviceNode := zkservice.ServiceNode{}
	err := conn.Get(servicePath, &serviceNode)
	if err != nil {
		glog.Errorf("Unable to find data %s: %v", servicePath, err)
		return err
	}

	mutator(serviceNode.Service)
	if err := conn.Set(servicePath, &serviceNode); err != nil {
		glog.Errorf("Unable to update service %s: %v", servicePath, err)
		return err
	}
	return nil
}

func loadAndUpdateHss(conn coordclient.Connection, hostId string, hssId string, mutator hssMutator) error {
	hssPath := HostServiceStatePath(hostId, hssId)
	var hss zkservice.HostState

	err := conn.Get(hssPath, &hss)
	if err != nil {
		// Should it really be an error if we can't find anything?
		glog.Errorf("Unable to find data %s: %v", hssPath, err)
		return err
	}

	mutator(&hss)
	if err := conn.Set(hssPath, &hss); err != nil {
		glog.Errorf("Unable to update host service state %s: %v", hssPath, err)
		return err
	}
	return nil
}

// ServiceState to HostServiceState
func SsToHss(ss *servicestate.ServiceState) *zkservice.HostState {
	return &zkservice.HostState{
		HostID:         ss.HostID,
		ServiceID:      ss.ServiceID,
		ServiceStateID: ss.ID,
		DesiredState:   service.SVCRun,
	}
}
