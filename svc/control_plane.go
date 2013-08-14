/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package serviced

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"math/rand"
	"time"
)

// A control plane implementation.
type ControlSvc struct {
	connectionString string
	connectionDriver string
}

// Ensure that ControlSvc implements the ControlPlane interface.
var _ ControlPlane = &ControlSvc{}

// Create a new ControlSvc or load an existing one.
func NewControlSvc(connectionUri string) (s *ControlSvc, err error) {
	s = new(ControlSvc)
	connInfo, err := parseDatabaseUri(connectionUri)
	if err != nil {
		return s, err
	}
	s.connectionString = toMymysqlConnectionString(connInfo)
	s.connectionDriver = "mymysql"
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

	host := dbmap.AddTableWithName(Host{}, "host").SetKeys(false, "Id")
	host.ColMap("Id").Rename("id")
	host.ColMap("Name").Rename("name")
	host.ColMap("IpAddr").Rename("ip_addr")
	host.ColMap("Cores").Rename("cores")
	host.ColMap("Memory").Rename("memory")
	host.ColMap("PrivateNetwork").Rename("private_network")

	pool := dbmap.AddTableWithName(ResourcePool{}, "resource_pool").SetKeys(false, "Id")
	pool.ColMap("Id").Rename("id")
	pool.ColMap("Name").Rename("name")
	pool.ColMap("CoreLimit").Rename("cores")
	pool.ColMap("MemoryLimit").Rename("memory")
	pool.ColMap("Priority").Rename("priority")

	poolhosts := dbmap.AddTableWithName(PoolHost{}, "resource_pool_host")
	poolhosts.ColMap("PoolId").Rename("resource_pool_id")
	poolhosts.ColMap("HostId").Rename("host_id")

	service := dbmap.AddTableWithName(Service{}, "service").SetKeys(false, "Id")
	service.ColMap("Id").Rename("id")
	service.ColMap("Name").Rename("name")
	service.ColMap("Startup").Rename("startup")
	service.ColMap("Description").Rename("description")
	service.ColMap("Instances").Rename("instances")
	service.ColMap("ImageId").Rename("image_id")
	service.ColMap("PoolId").Rename("resource_pool_id")
	service.ColMap("DesiredState").Rename("desired_state")

	servicesate := dbmap.AddTableWithName(ServiceState{}, "service_state").SetKeys(false, "Id")
	servicesate.ColMap("Id").Rename("id")
	servicesate.ColMap("ServiceId").Rename("service_id")
	servicesate.ColMap("HostId").Rename("host_id")
	servicesate.ColMap("Scheduled").Rename("scheduled_at")
	servicesate.ColMap("Terminated").Rename("terminated_at")
	servicesate.ColMap("Started").Rename("started_at")
	servicesate.ColMap("DockerId").Rename("docker_id")
	servicesate.ColMap("PrivateIp").Rename("private_ip")

	// dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds))
	return con, dbmap, err
}

// Get a service endpoint.
func (s *ControlSvc) GetServiceEndpoints(serviceEndpointRequest ServiceEndpointRequest, response *[]ApplicationEndpoint) (err error) {
	// TODO: Is this really useful?
	endpoints := make([]ApplicationEndpoint, 2)
	endpoints[0] = ApplicationEndpoint{"serviceFoo", "192.168.1.5", 8080, TCP}
	endpoints[1] = ApplicationEndpoint{"serviceFoo", "192.168.1.7", 8080, TCP}
	*response = endpoints
	return nil
}

// Return the matching hosts.
func (s *ControlSvc) GetHosts(request EntityRequest, hosts *map[string]*Host) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var hostList []*Host
	_, err = dbmap.Select(&hostList, "SELECT * FROM host")
	if err != nil {
		return err
	}
	hostmap := make(map[string]*Host)
	for _, host := range hostList {
		hostmap[host.Id] = host
	}
	*hosts = hostmap
	return err
}

// Add a host
func (s *ControlSvc) AddHost(host Host, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	return dbmap.Insert(&host)
}

// Update a host.
func (s *ControlSvc) UpdateHost(host Host, ununsed *int) (err error) {
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
	_, err = dbmap.Delete(&Host{Id: hostId})
	return err
}

// Get list of services.
func (s *ControlSvc) GetServices(request EntityRequest, replyServices *[]*Service) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var servicesList []*Service
	_, err = dbmap.Select(&servicesList, "SELECT * from service")
	if err != nil {
		return err
	}
	*replyServices = servicesList
	return err
}

// Add a service.
func (s *ControlSvc) AddService(service Service, unused *int) (err error) {
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
func (s *ControlSvc) UpdateService(service Service, unused *int) (err error) {
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
	_, err = dbmap.Delete(Service{Id: serviceId})
	return err
}

// Get all the services for a host.
func (s *ControlSvc) GetServicesForHost(hostId string, servicesForHost *[]*Service) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	obj, err := dbmap.Get(&Host{}, hostId)
	if obj == nil {
		return ControlPlaneError{"Could not find host"}
	}
	if err != nil {
		return err
	}
	var services []*Service
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
func (s *ControlSvc) GetServiceStates(serviceId string, states *[]*ServiceState) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	var serviceStates []*ServiceState
	_, err = dbmap.Select(&serviceStates, "SELECT * FROM service_state WHERE service_id=? and (terminated_at = '0001-01-01 00:00:00' or terminated_at = '0002-01-01 00:00:00') ", serviceId)
	if err != nil {
		return err
	}
	if len(serviceStates) == 0 {
		return ControlPlaneError{"Not found"}
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
			var services []*Service
			// get all service that are supposed to be running
			_, err = dbmap.Select(&services, "SELECT * FROM service WHERE desired_state = 1")
			if err != nil {
				log.Printf("trying to get services: %s", err)
				return err
			}
			for _, service := range services {
				// check current state
				var serviceStates []*ServiceState
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
						var pool_hosts []*PoolHost
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

						serviceState, err := NewServiceState(service.Id, service_host.HostId)
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
	var serviceStates []*ServiceState
	_, err = dbmap.Select(&serviceStates, "SELECT * FROM service_state WHERE service_id=? and terminated_at = '0001-01-01 00:00:00'", serviceId)
	if err != nil {
		return err
	}

	// get service
	obj, err := dbmap.Get(&Service{}, serviceId)
	if err != nil {
		return err
	}
	if obj == nil {
		return ControlPlaneError{"Service does not exist"}
	}
	service := obj.(*Service)

	service.DesiredState = 1
	_, err = dbmap.Update(service)
	if err != nil {
		return ControlPlaneError{"Could not set desired state to start"}
	}
	return err
}

//Schedule a service to restart.
func (s *ControlSvc) RestartService(serviceId string, unused *int) (err error) {
	return ControlPlaneError{"Unimplemented"}

}

//Schedule a service to stop.
func (s *ControlSvc) StopService(serviceId string, unused *int) (err error) {
	return ControlPlaneError{"Unimplemented"}
}

// Update the current state of a service instance.
func (s *ControlSvc) UpdateServiceState(state ServiceState, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Update(&state)
	return err
}

// Get all the hosts assigned to the given pool.
func (s *ControlSvc) GetHostsForResourcePool(poolId string, response *[]*PoolHost) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var poolHosts []*PoolHost
	_, err = dbmap.Select(&poolHosts, "SELECT * FROM resource_pool_host")
	if err == nil {
		*response = poolHosts
	}
	return err
}

// Get the resource pools.
func (s *ControlSvc) GetResourcePools(request EntityRequest, pools *map[string]*ResourcePool) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	var poolList []*ResourcePool
	_, err = dbmap.Select(&poolList, "SELECT * FROM resource_pool")
	if err != nil {
		return err
	}
	tpools := make(map[string]*ResourcePool)
	for _, pool := range poolList {
		tpools[pool.Id] = pool
	}
	*pools = tpools
	return err
}

// Add a resource pool.
func (s *ControlSvc) AddResourcePool(pool ResourcePool, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	err = dbmap.Insert(&pool)
	return err
}

// Update a resource pool.
func (s *ControlSvc) UpdateResourcePool(pool ResourcePool, unused *int) (err error) {
	return ControlPlaneError{"Unimplemented"}
}

// Remove a resource pool.
func (s *ControlSvc) RemoveResourcePool(poolId string, unused *int) (err error) {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Delete(&ResourcePool{Id: poolId})
	return err
}

// Add a host to a resource pool.
func (s *ControlSvc) AddHostToResourcePool(poolHost PoolHost, unused *int) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	obj, err := dbmap.Get(&Host{}, poolHost.HostId)
	if err != nil {
		return err
	}
	if obj == nil {
		return ControlPlaneError{"Host does not exist"}
	}
	obj, err = dbmap.Get(&ResourcePool{}, poolHost.PoolId)
	if err != nil {
		return err
	}
	if obj == nil {
		return ControlPlaneError{"Pool does not exist"}
	}
	return dbmap.Insert(&poolHost)
}

// Remove a  host from a resource pool.
func (s *ControlSvc) RemoveHostFromResourcePool(poolHost PoolHost, unused *int) error {
	db, dbmap, err := s.getDbConnection()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = dbmap.Exec("DELETE FROM resource_pool_host WHERE resource_pool_id = ? and host_id = ?",
		poolHost.PoolId, poolHost.HostId)
	return err
}
