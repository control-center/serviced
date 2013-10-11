/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package svc

import (
	"github.com/coopernurse/gorp"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/isvcs"
	_ "github.com/ziutek/mymysql/godrv"

	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

/* A control plane implementation.

ControlSvc implements the ControlPlane interface. It currently uses a mysql
database to keep state. It is responsible for tracking hosts, resource pools,
services, & service instances.
*/
type ControlSvc struct {
	connectionString string
	connectionDriver string
	zookeepers       []string
	scheduler        *scheduler
}

// Ensure that ControlSvc implements the ControlPlane interface.
var _ serviced.ControlPlane = &ControlSvc{}

// Create a new ControlSvc or load an existing one.
func NewControlSvc(connectionUri string, zookeepers []string) (s *ControlSvc, err error) {
	glog.Info("calling NewControlSvc()")
	defer glog.Info("leaving NewControlSvc()")
	s = new(ControlSvc)
	connInfo, err := serviced.ParseDatabaseUri(connectionUri)
	if err != nil {
		return s, err
	}
	s.connectionString = serviced.ToMymysqlConnectionString(connInfo)
	s.connectionDriver = "mymysql"

	if len(zookeepers) == 0 {
		isvcs.ZookeeperContainer.Run()
		s.zookeepers = []string{"127.0.0.1:2181"}
	} else {
		s.zookeepers = zookeepers
	}

	// ensure that a default pool exists
	_, err = s.getDefaultResourcePool()
	if err != nil {
		return s, err
	}

	hostId, err := serviced.HostId()
	if err != nil {
		return nil, err
	}

	go s.handleScheduler(hostId)
	return s, err
}

func (s *ControlSvc) handleScheduler(hostId string) {

	for {
		conn, _, err := zk.Connect(s.zookeepers, time.Second*10)
		if err != nil {
			time.Sleep(time.Second * 3)
			continue
		}
		scheduler, shutdown := newScheduler("", conn, hostId, s.lead)
		scheduler.Start()
		select {
		case <-shutdown:
		}
	}
}

type service_endpoint struct {
	ServiceId       string // service id
	Port            uint16 // port number
	ProtocolType    string // tcp or udp
	ApplicationType string
	Purpose         string // remote or local
}

type service_state_endpoint struct {
	ServiceStateId string
	Port           uint16
	ProtocolType   string
	ExternalPort   uint16
	IpAddr         string
}

// Get a database connection and map db entities to structs.
func (s *ControlSvc) getDbConnection() (con *sql.DB, dbmap *gorp.DbMap, err error) {
	con, err = sql.Open(s.connectionDriver, s.connectionString)
	if err != nil {
		return nil, nil, err
	}

	dbmap = &gorp.DbMap{Db: con, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}

	host := dbmap.AddTableWithName(serviced.Host{}, "host").SetKeys(false, "Id")
	host.ColMap("Id").Rename("id")
	host.ColMap("PoolId").Rename("resource_pool_id")
	host.ColMap("Name").Rename("name")
	host.ColMap("IpAddr").Rename("ip_addr")
	host.ColMap("Cores").Rename("cores")
	host.ColMap("Memory").Rename("memory")
	host.ColMap("PrivateNetwork").Rename("private_network")
	host.ColMap("CreatedAt").Rename("created_at").SetTransient(true)
	host.ColMap("UpdatedAt").Rename("updated_at").SetTransient(true)

	pool := dbmap.AddTableWithName(serviced.ResourcePool{}, "resource_pool").SetKeys(false, "Id")
	pool.ColMap("Id").Rename("id")
	pool.ColMap("ParentId").Rename("parent_id")
	pool.ColMap("CoreLimit").Rename("cores")
	pool.ColMap("MemoryLimit").Rename("memory")
	pool.ColMap("Priority").Rename("priority")
	pool.ColMap("CreatedAt").Rename("created_at").SetTransient(true)
	pool.ColMap("UpdatedAt").Rename("updated_at").SetTransient(true)

	service := dbmap.AddTableWithName(serviced.Service{}, "service").SetKeys(false, "Id")
	service.ColMap("Id").Rename("id")
	service.ColMap("Name").Rename("name")
	service.ColMap("Startup").Rename("startup")
	service.ColMap("Description").Rename("description")
	service.ColMap("Instances").Rename("instances")
	service.ColMap("ImageId").Rename("image_id")
	service.ColMap("PoolId").Rename("resource_pool_id")
	service.ColMap("DesiredState").Rename("desired_state")
	service.ColMap("Endpoints").SetTransient(true)
	service.ColMap("ParentServiceId").Rename("parent_service_id")
	service.ColMap("CreatedAt").Rename("created_at").SetTransient(true)
	service.ColMap("UpdatedAt").Rename("updated_at").SetTransient(true)

	servicesate := dbmap.AddTableWithName(serviced.ServiceState{}, "service_state").SetKeys(false, "Id")
	servicesate.ColMap("Id").Rename("id")
	servicesate.ColMap("ServiceId").Rename("service_id")
	servicesate.ColMap("HostId").Rename("host_id")
	servicesate.ColMap("Scheduled").Rename("scheduled_at")
	servicesate.ColMap("Terminated").Rename("terminated_at")
	servicesate.ColMap("Started").Rename("started_at")
	servicesate.ColMap("DockerId").Rename("docker_id")
	servicesate.ColMap("PrivateIp").Rename("private_ip")
	servicesate.ColMap("PortMapping").SetTransient(true)

	svc_endpoint := dbmap.AddTableWithName(service_endpoint{}, "service_endpoint")
	svc_endpoint.ColMap("ServiceId").Rename("service_id")
	svc_endpoint.ColMap("Port").Rename("port")
	svc_endpoint.ColMap("ProtocolType").Rename("protocol")
	svc_endpoint.ColMap("ApplicationType").Rename("application")
	svc_endpoint.ColMap("Purpose").Rename("purpose")

	svc_state_endpoint := dbmap.AddTableWithName(service_state_endpoint{}, "service_state_endpoint")
	svc_state_endpoint.ColMap("ServiceStateId").Rename("service_state_id")
	svc_state_endpoint.ColMap("Port").Rename("port")
	svc_state_endpoint.ColMap("ProtocolType").Rename("protocol")
	svc_state_endpoint.ColMap("ExternalPort").Rename("external_port")
	svc_state_endpoint.ColMap("IpAddr").Rename("ip_addr").SetTransient(true)

	svc_template := dbmap.AddTableWithName(serviced.ServiceTemplateWrapper{}, "service_template")
	svc_template.ColMap("Id").Rename("id")
	svc_template.ColMap("Name").Rename("name")
	svc_template.ColMap("Description").Rename("description")
	svc_template.ColMap("Data").Rename("data")
	svc_template.ColMap("ApiVersion").Rename("api_version")
	svc_template.ColMap("TemplateVersion").Rename("template_version")

	//dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds))
	return con, dbmap, err
}

const SQL_APPLICATION_ENDPOINTS = `
		select 
		    live_services.service_id as ServiceId, 
		    port as ContainerPort, 
		    external_port as HostPort, 
		    h.ip_addr as HostIp, 
		    live_services.private_ip as ContainerIp, 
		    protocol as Protocol
		from service_state_endpoint sse
		inner join 
		(
		    select ss.id, private_ip, host_id , ss.service_id
		    from service_state ss
		    where ss.terminated_at < '2000-01-01 00:00:00' and ss.started_at > '2000-01-01 00:00:00'
		    and ss.service_id in (
		        select service_id 
		        from service_endpoint se
		        where se.protocol = ? and se.application = ? and se.purpose = 'remote'
		    )
		) as live_services on live_services.id = sse.service_state_id
		inner join host h on h.id = live_services.host_id
`

// Get a service endpoint.
func (s *ControlSvc) GetServiceEndpoints(serviceId string, response *map[string][]*serviced.ApplicationEndpoint) (err error) {

	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	// first, get the list of ports that need proxying
	var service_endpoints []*service_endpoint
	_, err = dbmap.Select(&service_endpoints, `
	select *
from service_endpoint where service_id = ?
and purpose = 'local'`, serviceId)
	if err != nil {
		return err
	}

	// no services need to be proxied
	if len(service_endpoints) == 0 {
		glog.Errorf("No service endpoints found for %s", serviceId)
		return nil
	}

	remoteEndpoints := make(map[string][]*serviced.ApplicationEndpoint)

	// for each proxied port, find list of potential remote endpoints
	for _, localport := range service_endpoints {
		var applicationEndpoints []*serviced.ApplicationEndpoint
		_, err := dbmap.Select(&applicationEndpoints, SQL_APPLICATION_ENDPOINTS,
			string(localport.ProtocolType), string(localport.ApplicationType))
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%s:%d", localport.ProtocolType, localport.Port)
		remoteEndpoints[key] = applicationEndpoints
	}

	*response = remoteEndpoints
	return nil
}

// Return the matching hosts.
func (s *ControlSvc) GetHosts(request serviced.EntityRequest, hosts *map[string]*serviced.Host) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var hostList []*serviced.Host
	_, err = dbmap.Select(&hostList, "SELECT * FROM host")
	if err != nil {
		return err
	}
	hostmap := make(map[string]*serviced.Host)
	for _, host := range hostList {
		hostmap[host.Id] = host
	}
	*hosts = hostmap
	return err
}

// Add a host
func (s *ControlSvc) AddHost(host serviced.Host, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	return dbmap.Insert(&host)
}

// Update a host.
func (s *ControlSvc) UpdateHost(host serviced.Host, ununsed *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Update(&host)
	return err
}

// Remove a host.
func (s *ControlSvc) RemoveHost(hostId string, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Delete(&serviced.Host{Id: hostId})
	return err
}

func portToEndpoint(servicePorts []*service_endpoint) *[]serviced.ServiceEndpoint {
	endpoints := make([]serviced.ServiceEndpoint, len(servicePorts))
	for i, servicePort := range servicePorts {
		endpoints[i] = serviced.ServiceEndpoint{
			string(servicePort.ProtocolType),
			uint16(servicePort.Port),
			string(servicePort.ApplicationType),
			servicePort.Purpose,
		}
	}
	return &endpoints
}

func endpointToPort(service serviced.Service) (servicePorts []service_endpoint) {
	if service.Endpoints == nil {
		return make([]service_endpoint, 0)
	}
	service_ports := make([]service_endpoint, len(*service.Endpoints))
	for i, endpoint := range *service.Endpoints {
		service_ports[i] = service_endpoint{
			service.Id,
			endpoint.PortNumber,
			string(endpoint.Protocol),
			string(endpoint.Application),
			endpoint.Purpose}
	}
	return service_ports
}

func (s *ControlSvc) addEndpointsToServices(servicesList []*serviced.Service) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	// Get the related ports for each service
	for _, service := range servicesList {
		var servicePorts []*service_endpoint
		_, err = dbmap.Select(&servicePorts, "SELECT * FROM service_endpoint WHERE service_id = ?", service.Id)
		if err != nil {
			return err
		}
		service.Endpoints = portToEndpoint(servicePorts)
	}
	return nil
}

// Get list of services.
func (s *ControlSvc) GetServices(request serviced.EntityRequest, replyServices *[]*serviced.Service) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var servicesList []*serviced.Service
	_, err = dbmap.Select(&servicesList, "SELECT * from service")
	if err != nil {
		return err
	}

	// Get the related ports for each service
	err = s.addEndpointsToServices(servicesList)
	if err != nil {
		return err
	}
	*replyServices = servicesList
	return err
}

// Add a service.
func (s *ControlSvc) AddService(service serviced.Service, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := dbmap.Begin()
	if err != nil {
		return err
	}
	err = tx.Insert(&service)
	if err != nil {
		return err
	}
	glog.Infof("Got a service with endpoints: %v", service.Endpoints)
	for _, serviceEndpoint := range endpointToPort(service) {
		err = tx.Insert(&serviceEndpoint)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Update a service.
func (s *ControlSvc) UpdateService(service serviced.Service, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = dbmap.Update(&service)
	return err
}

// Remove a service.
func (s *ControlSvc) RemoveService(serviceId string, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Delete(&serviced.Service{Id: serviceId})
	return err
}

// Get all the services for a host.
func (s *ControlSvc) GetServicesForHost(hostId string, servicesForHost *[]*serviced.Service) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	obj, err := dbmap.Get(&serviced.Host{}, hostId)
	if obj == nil {
		return serviced.ControlPlaneError{"Could not find host"}
	}
	if err != nil {
		return err
	}
	var services []*serviced.Service
	_, err = dbmap.Select(&services, "SELECT service.* "+
		"FROM service_state AS state "+
		"INNER JOIN service ON service.id = state.service_id "+
		"WHERE state.host_id = ?", hostId)
	if err != nil {
		return err
	}
	err = s.addEndpointsToServices(services)
	if err != nil {
		return err
	}
	*servicesForHost = services
	return err
}

// Get the current states of the running service instances.
func (s *ControlSvc) GetServiceStates(serviceId string, states *[]*serviced.ServiceState) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	var serviceStates []*serviced.ServiceState
	_, err = dbmap.Select(&serviceStates, "SELECT * FROM service_state WHERE service_id=? and (terminated_at = '0001-01-01 00:00:00' or terminated_at = '0002-01-01 00:00:00') ", serviceId)
	if err != nil {
		return err
	}
	if len(serviceStates) == 0 {
		return serviced.ControlPlaneError{"Not found"}
	}
	*states = serviceStates
	return nil
}

// Start a service and its subservices
func startService(dbmap *gorp.DbMap, serviceId string) error {
	// check current state
	var err error
	var serviceStates []*serviced.ServiceState
	_, err = dbmap.Select(&serviceStates, "SELECT * FROM service_state WHERE service_id=? and terminated_at = '0001-01-01 00:00:00'", serviceId)
	if err != nil {
		return err
	}

	// get service
	obj, err := dbmap.Get(&serviced.Service{}, serviceId)
	if err != nil {
		return err
	}
	if obj == nil {
		return serviced.ControlPlaneError{"Service does not exist"}
	}
	service := obj.(*serviced.Service)

	service.DesiredState = 1
	_, err = dbmap.Update(service)
	if err != nil {
		return serviced.ControlPlaneError{"Could not set desired state to start"}
	}

	// find sub services and start them
	var subserviceIds []*struct{ Id string }
	_, err = dbmap.Select(&subserviceIds, "SELECT id as Id from service WHERE parent_service_id = ? ", serviceId)
	if err != nil {
		return serviced.ControlPlaneError{
			fmt.Sprintf("Could not get subservices: %s", err.Error()),
		}
	}
	for _, obj := range subserviceIds {
		err = startService(dbmap, obj.Id)
		if err != nil {
			return err
		}
	}

	return err

}

// Schedule a service to run on a host.
func (s *ControlSvc) StartService(serviceId string, unused *string) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	return startService(dbmap, serviceId)
}

//Schedule a service to restart.
func (s *ControlSvc) RestartService(serviceId string, unused *int) (err error) {
	return serviced.ControlPlaneError{"Unimplemented"}

}

//Schedule a service to stop.
func (s *ControlSvc) StopService(serviceId string, unused *int) (err error) {
	return serviced.ControlPlaneError{"Unimplemented"}
}

func (s *ControlSvc) getServiceStateEndpoints(serviceStateId string) (endpoints map[string]service_state_endpoint, err error) {

	endpoints = make(map[string]service_state_endpoint)

	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return endpoints, err
	}
	defer db.Close()
	var endpointList []*service_state_endpoint
	_, err = dbmap.Select(&endpointList, "SELECT * FROM service_state_endpoint WHERE service_state_id = ? ORDER BY port", serviceStateId)

	if err != nil {
		return endpoints, err
	}
	for _, endpoint := range endpointList {
		endpoints[fmt.Sprintf("%d", endpoint.Port)] = *endpoint
	}
	return endpoints, nil
}

// Update the current state of a service instance.
func (s *ControlSvc) UpdateServiceState(state serviced.ServiceState, unused *int) (err error) {
	glog.Infoln("Entering UpdateServiceState()")
	defer glog.Infoln("Leaving UpdateServiceState()")
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	glog.Infof("Got back a service state with portmappings: %v", state.PortMapping)
	tx, err := dbmap.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Update(&state)
	if err != nil {
		return err
	}
	// Get all the existing service_state_endpoints
	endpoints, err := s.getServiceStateEndpoints(state.Id)
	if err != nil {
		glog.Fatalf("Problem getting service state endpoints: %v", err)
		return err
	}
	glog.Infof("About to iterate state port mappings: %v", state)
	if tcpMap, ok := state.PortMapping["Tcp"]; ok {
		for internalStr, externalStr := range tcpMap {
			if _, ok := endpoints[internalStr]; !ok {
				external, err := strconv.Atoi(externalStr)
				if err != nil {
					return err
				}
				internal, err := strconv.Atoi(internalStr)
				if err != nil {
					return err
				}
				tx.Insert(&service_state_endpoint{state.Id, uint16(internal), "tcp", uint16(external), ""})
			}
		}
	}

	return tx.Commit()
}

func (s *ControlSvc) getSubResourcePools(poolId string) (poolIds []string, err error) {

	poolIds = make([]string, 0)
	if len(poolId) == 0 {
		return poolIds, nil
	}
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return poolIds, err
	}
	defer db.Close()

	var pools []*serviced.ResourcePool
	_, err = dbmap.Select(&pools, "SELECT * FROM resource_pool WHERE parent_id = ?", poolId)
	if err != nil {
		return poolIds, err
	}
	for _, pool := range pools {
		poolIds = append(poolIds, pool.Id)
		subPoolIds, err := s.getSubResourcePools(pool.Id)
		if err != nil {
			return poolIds, err
		}
		for _, subPoolId := range subPoolIds {
			poolIds = append(poolIds, subPoolId)
		}
	}
	return poolIds, nil
}

// Get all the hosts assigned to the given pool.
func (s *ControlSvc) GetHostsForResourcePool(poolId string, response *[]*serviced.PoolHost) (err error) {

	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	poolIds, err := s.getSubResourcePools(poolId)
	if err != nil {
		return err
	}
	poolIds = append(poolIds, poolId)

	var hosts []*serviced.Host
	stmt := "SELECT * FROM host WHERE resource_pool_id in ('" + strings.Join(poolIds, "','") + "')"
	glog.Infof("SQL: %s", stmt)
	_, err = dbmap.Select(&hosts, stmt)
	if err != nil {
		return err
	}
	responseList := make([]*serviced.PoolHost, len(hosts))
	for i, host := range hosts {
		responseList[i] = &serviced.PoolHost{host.Id, host.PoolId}
	}
	*response = responseList
	return err
}

// Get the default resource pool. If it doesn't exist, create it. Return any
// errors in retrieving or creating the default resource pool.
func (s *ControlSvc) getDefaultResourcePool() (pool *serviced.ResourcePool, err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return pool, err
	}
	defer db.Close()

	obj, err := dbmap.Get(serviced.ResourcePool{}, "default")
	if obj == nil {
		glog.Infof("'default' resource pool not found; creating...")
		default_pool := serviced.ResourcePool{}
		default_pool.Id = "default"
		err = dbmap.Insert(&default_pool)
		return &default_pool, err
	}
	pool, ok := obj.(*serviced.ResourcePool)
	if !ok {
		glog.Errorln("Could not cast obj.")
	}
	return pool, err
}

// Get the resource pools.
func (s *ControlSvc) GetResourcePools(request serviced.EntityRequest, pools *map[string]*serviced.ResourcePool) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var poolList []*serviced.ResourcePool
	_, err = dbmap.Select(&poolList, "SELECT * FROM resource_pool")
	if err != nil {
		return err
	}
	tpools := make(map[string]*serviced.ResourcePool)
	for _, pool := range poolList {
		tpools[pool.Id] = pool
	}
	*pools = tpools
	return err
}

// Add a resource pool.
func (s *ControlSvc) AddResourcePool(pool serviced.ResourcePool, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	err = dbmap.Insert(&pool)
	return err
}

// Update a resource pool.
func (s *ControlSvc) UpdateResourcePool(pool serviced.ResourcePool, unused *int) (err error) {
	return serviced.ControlPlaneError{"Unimplemented"}
}

// Remove a resource pool.
func (s *ControlSvc) RemoveResourcePool(poolId string, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Delete(&serviced.ResourcePool{Id: poolId})
	return err
}

func (s *ControlSvc) lead(zkEvent <-chan zk.Event) {
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
				// passthru
			}
			db, dbmap, err := s.getDbConnection()
			if err != nil {
				glog.Infof("trying to connection: %s", err)
				return err
			}
			defer db.Close()
			var services []*serviced.Service
			// get all service that are supposed to be running
			_, err = dbmap.Select(&services, "SELECT * FROM `service` WHERE desired_state = 1")
			if err != nil {
				return err
			}
			for _, service := range services {
				// check current state
				var serviceStates []*serviced.ServiceState
				_, err = dbmap.Select(&serviceStates, "SELECT * FROM service_state WHERE service_id=? and terminated_at = '0001-01-01 00:00:00'", service.Id)
				if err != nil {
					glog.Errorf("Got error checking service state of %s, %s", service.Id, err.Error())
					return err
				}
				if len(serviceStates) == service.Instances {
					continue
				}

				instancesToStart := service.Instances - len(serviceStates)
				if instancesToStart >= 0 {
					for i := 0; i < instancesToStart; i++ {
						// get hosts
						var pool_hosts []*serviced.PoolHost
						err = s.GetHostsForResourcePool(service.PoolId, &pool_hosts)
						if err != nil {
							return err
						}
						if len(pool_hosts) == 0 {
							glog.Infof("Pool %s has no hosts", service.PoolId)
							break
						}

						// randomly select host
						service_host := pool_hosts[rand.Intn(len(pool_hosts))]

						serviceState, err := service.NewServiceState(service_host.HostId)
						if err != nil {
							glog.Errorf("Error creating ServiceState instance: %v", err)
							break
						}
						glog.Infof("cp: serviceState %s", serviceState.Started)
						err = dbmap.Insert(serviceState)
					}
				} else {
					// pick service instances to kill!
					instancesToKill := len(serviceStates) - service.Instances
					for i := 0; i < instancesToKill; i++ {
						glog.Infof("CP: Choosing to kill %s:%s\n", serviceStates[i].HostId, serviceStates[i].DockerId)
						serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
						_, err = dbmap.Update(serviceStates[i])
					}
				}
			}
			return nil
		}()
	}

}

func (s *ControlSvc) DeployTemplate(request serviced.ServiceTemplateDeploymentRequest, unused *int) error {
	return fmt.Errorf("unimplemented DeployTemplate")
}

func (s *ControlSvc) GetServiceTemplates(unused int, serviceTemplates *map[string]*serviced.ServiceTemplate) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var templateWrappers []*serviced.ServiceTemplateWrapper
	_, err = dbmap.Select(&templateWrappers, "SELECT * FROM service_template")
	if err != nil {
		fmt.Errorf("could not get service templates: %s", err)
	}

	templates := make(map[string]*serviced.ServiceTemplate, len(templateWrappers))
	for _, templateWrapper := range templateWrappers {
		var template serviced.ServiceTemplate
		err := json.Unmarshal([]byte(templateWrapper.Data), &template)
		if err != nil {
			return err
		}
		templates[templateWrapper.Id] = &template
	}

	*serviceTemplates = templates

	return nil
}

func (s *ControlSvc) AddServiceTemplate(serviceTemplate serviced.ServiceTemplate, unused *int) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	data, err := json.Marshal(serviceTemplate)
	if err != nil {
		return err
	}

	uuid, err := serviced.NewUuid()
	if err != nil {
		return err
	}

	wrapper := serviced.ServiceTemplateWrapper{uuid, serviceTemplate.Name, serviceTemplate.Description, string(data), 1, 1}

	return dbmap.Insert(&wrapper)
}

func (s *ControlSvc) UpdateServiceTemplate(serviceTemplate serviced.ServiceTemplate, unused *int) error {
	return fmt.Errorf("unimplemented UpdateServiceTemplate")
}

func (s *ControlSvc) RemoveServiceTemplate(serviceTemplateId string, unused *int) error {
	return fmt.Errorf("unimplemented RemoveServiceTemplate")
}
