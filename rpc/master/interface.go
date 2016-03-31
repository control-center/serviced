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
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/volume"
	"time"
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
	AddHost(host host.Host) error

	// UpdateHost updates a host
	UpdateHost(host host.Host) error

	// RemoveHost removes a host
	RemoveHost(hostID string) error

	// FindHostsInPool returns all hosts in a pool
	FindHostsInPool(poolID string) ([]host.Host, error)

	//--------------------------------------------------------------------------
	// Pool Management Functions

	// GetResourcePool gets the pool for the given poolID or nil
	GetResourcePool(poolID string) (*pool.ResourcePool, error)

	// GetResourcePools returns all pools or empty array
	GetResourcePools() ([]pool.ResourcePool, error)

	// AddResourcePool adds the ResourcePool
	AddResourcePool(pool pool.ResourcePool) error

	// UpdateResourcePool adds the ResourcePool
	UpdateResourcePool(pool pool.ResourcePool) error

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
}
