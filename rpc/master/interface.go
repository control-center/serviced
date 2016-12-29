// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package master

import (
	"time"

	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/volume"
)

// The RPC interface is the API for a serviced master.
type ClientInterface interface {

	//--------------------------------------------------------------------------
	// RPC Management Functions
	Close() (err error)

	//--------------------------------------------------------------------------
	// Host Management Functions

	// GetHost gets the host for the given hostID or nil
	GetHost(hostID string) (*host.Host, error)

	// GetHosts returns all hosts or empty array
	GetHosts() ([]host.Host, error)

	// GetActiveHosts returns all active host ids or empty array
	GetActiveHostIDs() ([]string, error)

	// AddHost adds a Host
	AddHost(h host.Host) ([]byte, error)

	// UpdateHost updates a host
	UpdateHost(h host.Host) error

	// RemoveHost removes a host
	RemoveHost(hostID string) error

	// FindHostsInPool returns all hosts in a pool
	FindHostsInPool(poolID string) ([]host.Host, error)

	// Authenticate a host and receive an identity token and expiration
	AuthenticateHost(hostID string) (string, int64, error)

	// Get hostID's public key
	GetHostPublicKey(hostID string) ([]byte, error)

	// Reset hostID's private key
	ResetHostKey(hostID string) ([]byte, error)

	// HostsAuthenticated returns if the hosts passed are authenticated or not
	HostsAuthenticated(hostIDs []string) (map[string]bool, error)

	//--------------------------------------------------------------------------
	// Pool Management Functions

	// GetResourcePool gets the pool for the given poolID or nil
	GetResourcePool(poolID string) (*pool.ResourcePool, error)

	// GetResourcePools returns all pools or empty array
	GetResourcePools() ([]pool.ResourcePool, error)

	// AddResourcePool adds the ResourcePool
	AddResourcePool(p pool.ResourcePool) error

	// UpdateResourcePool adds the ResourcePool
	UpdateResourcePool(p pool.ResourcePool) error

	// RemoveResourcePool removes a ResourcePool
	RemoveResourcePool(poolID string) error

	// GetPoolIPs returns a all IPs in a ResourcePool.
	GetPoolIPs(poolID string) (*pool.PoolIPs, error)

	// AddVirtualIP adds a VirtualIP to a specific pool
	AddVirtualIP(requestVirtualIP pool.VirtualIP) error

	// RemoveVirtualIP removes a VirtualIP from a specific pool
	RemoveVirtualIP(requestVirtualIP pool.VirtualIP) error

	//--------------------------------------------------------------------------
	// Service Management Functions

	// ServiceUse will use a new image for a given service - this will pull the image and tag it
	ServiceUse(serviceID string, imageID string, registry string, replaceImgs []string, noOp bool) (string, error)

	// WaitService will wait for the specified services to reach the specified state, within the given timeout
	WaitService(serviceIDs []string, state service.DesiredState, timeout time.Duration, recursive bool) error

	// GetAllServiceDetails will return a list of all ServiceDetails
	GetAllServiceDetails(since time.Duration) ([]service.ServiceDetails, error)

	// GetServiceDetailsByTenantID will return a list of ServiceDetails for the specified tenant ID
	GetServiceDetailsByTenantID(tenantID string) ([]service.ServiceDetails, error)

	// GetServiceDetails will return a ServiceDetails for the specified service
	GetServiceDetails(serviceID string) (*service.ServiceDetails, error)

	// ResolveServicePath will return ServiceDetails that match the given path
	ResolveServicePath(path string) ([]service.ServiceDetails, error)

	// ClearEmergency will set EmergencyShutdown to false on the service and all child services
	ClearEmergency(serviceID string) (int, error)

	//--------------------------------------------------------------------------
	// Service Instance Management Functions

	// GetServiceInstances returns all running instances of a service
	GetServiceInstances(serviceID string) ([]service.Instance, error)

	// Get a service from serviced where all templated properties have been evaluated
	GetEvaluatedService(serviceID string, instanceID int) (*service.Service, string, error)

	// Get the tenant ID for a service
	GetTenantID(serviceID string) (string, error)

	// StopServiceInstance stops a single service instance
	StopServiceInstance(serviceID string, instanceID int) error

	// LocateServiceInstance returns location information about a service
	// instance
	LocateServiceInstance(serviceID string, instanceID int) (*service.LocationInstance, error)

	// SendDockerAction submits a docker action to a running container
	SendDockerAction(serviceID string, instanceID int, action string, args []string) error

	//--------------------------------------------------------------------------
	// Service Tempatate Management Functions

	// Add a new service template
	AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate) (templateID string, err error)

	// Get a list of ServiceTemplates
	GetServiceTemplates() (serviceTemplates map[string]servicetemplate.ServiceTemplate, err error)

	// Remove a service Template
	RemoveServiceTemplate(serviceTemplateID string) error

	// Deploy an application template
	DeployTemplate(request servicetemplate.ServiceTemplateDeploymentRequest) (tenantIDs []string, err error)

	//--------------------------------------------------------------------------
	// Volume Management Functions

	// GetVolumeStatus gets status information for the given volume or nil
	GetVolumeStatus() (*volume.Statuses, error)

	//--------------------------------------------------------------------------
	// Endpoint Management Functions

	// GetServiceEndpoints gets the endpoints for one or more services
	GetServiceEndpoints(serviceIDs []string, reportImports, reportExports bool, validate bool) ([]applicationendpoint.EndpointReport, error)

	//--------------------------------------------------------------------------
	// Docker Registry Management Functions

	// ResetRegistry pulls images from the docker registry and updates the
	// index.
	ResetRegistry() error

	// SyncRegistry prompts the master to push its images into the docker
	// registry.
	SyncRegistry() error

	// UpgradeRegistry migrates images from an older or remote docker registry.
	UpgradeRegistry(endpoint string, override bool) error

	// DockerOverride replaces an image in the docker registry with a new image
	DockerOverride(newImage, oldImage string) error

	//--------------------------------------------------------------------------
	// Public Endpoint Management Functions
	AddPublicEndpointPort(serviceid, endpointName, portAddr string, usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error)

	RemovePublicEndpointPort(serviceid, endpointName, portAddr string) error

	EnablePublicEndpointPort(serviceid, endpointName, portAddr string, isEnabled bool) error

	AddPublicEndpointVHost(serviceid, endpointName, vhost string, isEnabled, restart bool) (*servicedefinition.VHost, error)

	RemovePublicEndpointVHost(serviceid, endpointName, vhost string) error

	EnablePublicEndpointVHost(serviceid, endpointName, vhost string, isEnabled bool) error

	GetAllPublicEndpoints() ([]service.PublicEndpoint, error)

	//--------------------------------------------------------------------------
	// User Management Functions

	// Get the system user record
	GetSystemUser() (user.User, error)

	// Validate the credentials of the specified user
	ValidateCredentials(user user.User) (bool, error)

	//--------------------------------------------------------------------------
	// Healthcheck Management Functions

	// GetISvcsHealth returns health status for a list of isvcs
	GetISvcsHealth(IServiceNames []string) ([]isvcs.IServiceHealthResult, error)

	// GetServicesHealth returns health checks for all services.
	GetServicesHealth() (map[string]map[int]map[string]health.HealthStatus, error)

	// ReportHealthStatus sends an update to the health check status cache.
	ReportHealthStatus(key health.HealthStatusKey, value health.HealthStatus, expires time.Duration) error

	// ReportInstanceDead removes stopped instances from the health check status cache.
	ReportInstanceDead(serviceID string, instanceID int) error

	//--------------------------------------------------------------------------
	// Debug Management Functions

	// Enable internal metrics collection
	DebugEnableMetrics() (string, error)

	// Disable internal metrics collection
	DebugDisableMetrics() (string, error)
}
