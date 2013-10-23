package dao

// A generic ControlPlane error
type ControlPlaneError struct {
	Msg string
}

// Implement the Error() interface for ControlPlaneError
func (s ControlPlaneError) Error() string {
	return s.Msg
}

// An request for a control plane object.
type EntityRequest struct {}

// The ControlPlane interface is the API for a serviced master.
type ControlPlane interface {

  //---------------------------------------------------------------------------
  // Host CRUD

	// Register a host with serviced
	AddHost(host Host, unused *int) error

	// Update Host information for a registered host
	UpdateHost(host Host, ununsed *int) error

	// Remove a Host from serviced
	RemoveHost(hostId string, unused *int) error

  //TODO Does this belong here?
	// Get Host for a registered host
	//GetHost(hostId int, host *Host) error

	// Get a list of registered hosts
	GetHosts(request EntityRequest, hosts *map[string]*Host) error

  //---------------------------------------------------------------------------
  // Service CRUD

	// Add a new service
	AddService(service Service, unused *int) error

	// Update an existing service
	UpdateService(service Service, unused *int) error

	// Remove a service definition
	RemoveService(serviceId string, unused *int) error

  //TODO
	// Get a service from serviced
	//GetService(serviceId int, service *Service) error

	// Get a list of services from serviced
	GetServices(request EntityRequest, services *[]*Service) error

  //---------------------------------------------------------------------------
  //ServiceState CRUD

	// Schedule the given service to start
	StartService(serviceId string, unused *string) error

	// Schedule the given service to restart
	RestartService(serviceId string, unused *int) error

	// Schedule the given service to stop
	StopService(serviceId string, unused *int) error

	// Update the service state
	UpdateServiceState(state ServiceState, unused *int) error

  //TODO
	//// Get the services instances for a given service
	//GetServiceStates(serviceId string, states *[]*ServiceState) error

	// Get all the services that need to be running on the given host
	//GetServicesForHost(hostId string, services *[]*Service) error

	// Get logs for the given app
	GetServiceLogs(serviceId string, logs *string) error

	// Get logs for the given app
	GetServiceStateLogs(serviceStateId string, logs *string) error

	// Get the services instances for a given service
	GetRunningServicesForHost(hostId string, runningServices *[]*RunningService) error

  //---------------------------------------------------------------------------
  // ResourcePool CRUD

	// Add a new service pool to serviced
	AddResourcePool(pool ResourcePool, unused *int) error

	// Update a service pool definition
	UpdateResourcePool(pool ResourcePool, unused *int) error

	// Remove a service pool
	RemoveResourcePool(poolId string, unused *int) error

  //TODO does this belong here
	// Get a list of all the resource pools
	//GetResourcePool(poolId string, pool *ResourcePool) error

	// Get a list of all the resource pools
	GetResourcePools(request EntityRequest, pool *map[string]*ResourcePool) error

	// Get of a list of hosts that are in the given resource pool
	GetHostsForResourcePool(poolId string, poolHosts *[]*PoolHost) error

  //---------------------------------------------------------------------------
  // ServiceTemplate CRUD

	// Deploy an application template in to production
	DeployTemplate(request ServiceTemplateDeploymentRequest, unused *int) error

	// Add a new service Template
	AddServiceTemplate(serviceTemplate ServiceTemplate, unused *int) error

	// Update a new service Template
	UpdateServiceTemplate(serviceTemplate ServiceTemplate, unused *int) error

	// Update a new service Template
	RemoveServiceTemplate(serviceTemplateId string, unused *int) error

	// Get a list of ServiceTemplates
	GetServiceTemplates(unused int, serviceTemplates *map[string]*ServiceTemplate) error
}
