/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package svc

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	"github.com/zenoss/serviced"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"math/rand"
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
}

// Ensure that ControlSvc implements the ControlPlane interface.
var _ serviced.ControlPlane = &ControlSvc{}

// Create a new ControlSvc or load an existing one.
func NewControlSvc(connectionUri string) (s *ControlSvc, err error) {
	s = new(ControlSvc)
	connInfo, err := serviced.ParseDatabaseUri(connectionUri)
	if err != nil {
		return s, err
	}
	s.connectionString = serviced.ToMymysqlConnectionString(connInfo)
	s.connectionDriver = "mymysql"

	// ensure that a default pool exists
	_, err = s.getDefaultResourcePool()
	if err != nil {
		return s, err
	}

	go s.scheduler()
	return s, err
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

	pool := dbmap.AddTableWithName(serviced.ResourcePool{}, "resource_pool").SetKeys(false, "Id")
	pool.ColMap("Id").Rename("id")
	pool.ColMap("ParentId").Rename("parent_id")
	pool.ColMap("CoreLimit").Rename("cores")
	pool.ColMap("MemoryLimit").Rename("memory")
	pool.ColMap("Priority").Rename("priority")

	service := dbmap.AddTableWithName(serviced.Service{}, "service").SetKeys(false, "Id")
	service.ColMap("Id").Rename("id")
	service.ColMap("Name").Rename("name")
	service.ColMap("Startup").Rename("startup")
	service.ColMap("Description").Rename("description")
	service.ColMap("Instances").Rename("instances")
	service.ColMap("ImageId").Rename("image_id")
	service.ColMap("PoolId").Rename("resource_pool_id")
	service.ColMap("DesiredState").Rename("desired_state")

	servicesate := dbmap.AddTableWithName(serviced.ServiceState{}, "service_state").SetKeys(false, "Id")
	servicesate.ColMap("Id").Rename("id")
	servicesate.ColMap("ServiceId").Rename("service_id")
	servicesate.ColMap("HostId").Rename("host_id")
	servicesate.ColMap("Scheduled").Rename("scheduled_at")
	servicesate.ColMap("Terminated").Rename("terminated_at")
	servicesate.ColMap("Started").Rename("started_at")
	servicesate.ColMap("DockerId").Rename("docker_id")
	servicesate.ColMap("PrivateIp").Rename("private_ip")

	//dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds))
	return con, dbmap, err
}

// Get a service endpoint.
func (s *ControlSvc) GetServiceEndpoints(service serviced.ServiceEndpointRequest, response *[]serviced.ApplicationEndpoint) (err error) {
	// TODO: Is this really useful?
	endpoints := make([]serviced.ApplicationEndpoint, 2)
	endpoints[0] = serviced.ApplicationEndpoint{"serviceFoo", "192.168.1.5", 8080, serviced.TCP}
	endpoints[1] = serviced.ApplicationEndpoint{"serviceFoo", "192.168.1.7", 8080, serviced.TCP}
	*response = endpoints
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

	if err != nil {
		return err
	}
	return dbmap.Insert(&service)
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

// This is the main loop for the scheduler.
func (s *ControlSvc) scheduler() {

	for {
		time.Sleep(time.Second)
		func() error {
			db, dbmap, err := s.getDbConnection()
			if err != nil {
				log.Printf("trying to connection: %s", err)
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
					log.Printf("Got error checking service state of %s, %s", service.Id, err.Error())
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
						_, err = dbmap.Select(&pool_hosts, "SELECT * FROM resource_pool_host "+
							"WHERE resource_pool_id = ?", service.PoolId)
						if err != nil {
							return err
						}
						if len(pool_hosts) == 0 {
							log.Printf("Pool %s has no hosts", service.PoolId)
							break
						}

						// randomly select host
						service_host := pool_hosts[rand.Intn(len(pool_hosts))]

						serviceState, err := service.NewServiceState(service_host.HostId)
						if err != nil {
							log.Printf("Error creating ServiceState instance: %v", err)
							break
						}
						log.Printf("cp: serviceState %s", serviceState.Started)
						err = dbmap.Insert(serviceState)
					}
				} else {
					// pick service instances to kill!
					instancesToKill := len(serviceStates) - service.Instances
					for i := 0; i < instancesToKill; i++ {
						log.Printf("CP: Choosing to kill %s:%s\n", serviceStates[i].HostId, serviceStates[i].DockerId)
						serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
						_, err = dbmap.Update(serviceStates[i])
					}
				}
			}
			return nil
		}()
	}
}

// Schedule a service to run on a host.
func (s *ControlSvc) StartService(serviceId string, hostId *string) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	// check current state
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
	return err
}

//Schedule a service to restart.
func (s *ControlSvc) RestartService(serviceId string, unused *int) (err error) {
	return serviced.ControlPlaneError{"Unimplemented"}

}

//Schedule a service to stop.
func (s *ControlSvc) StopService(serviceId string, unused *int) (err error) {
	return serviced.ControlPlaneError{"Unimplemented"}
}

// Update the current state of a service instance.
func (s *ControlSvc) UpdateServiceState(state serviced.ServiceState, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Update(&state)
	return err
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
	log.Printf("SQL: %s", stmt)
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
		log.Printf("'default' resource pool not found; creating...")
		default_pool := serviced.ResourcePool{}
		default_pool.Id = "default"
		err = dbmap.Insert(&default_pool)
		return &default_pool, err
	}
	pool, ok := obj.(*serviced.ResourcePool)
	if !ok {
		log.Printf("Could not cast obj.")
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

// Add a host to a resource pool.
func (s *ControlSvc) AddHostToResourcePool(poolHost serviced.PoolHost, unused *int) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	obj, err := dbmap.Get(&serviced.Host{}, poolHost.HostId)
	if err != nil {
		return err
	}
	if obj == nil {
		return serviced.ControlPlaneError{"Host does not exist"}
	}
	obj, err = dbmap.Get(&serviced.ResourcePool{}, poolHost.PoolId)
	if err != nil {
		return err
	}
	if obj == nil {
		return serviced.ControlPlaneError{"Pool does not exist"}
	}
	return dbmap.Insert(&poolHost)
}

// Remove a  host from a resource pool.
func (s *ControlSvc) RemoveHostFromResourcePool(poolHost serviced.PoolHost, unused *int) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Exec("DELETE FROM resource_pool_host WHERE resource_pool_id = ? and host_id = ?",
		poolHost.PoolId, poolHost.HostId)
	return err
}
