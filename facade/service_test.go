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

// +build integration

package facade

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	zzkmocks "github.com/control-center/serviced/facade/mocks"
	"github.com/control-center/serviced/health"
	ssmmocks "github.com/control-center/serviced/scheduler/servicestatemanager/mocks"
	zks "github.com/control-center/serviced/zzk/service"

	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
	"github.com/control-center/serviced/domain/logfilter"
)

var (
	ErrTestEPValidationFail = errors.New("Endpoint failed validation")
)

func (ft *FacadeIntegrationTest) TestFacade_validateServiceName(c *C) {
	svcA := service.Service{
		ID:           "validate-service-name-A",
		Name:         "TestFacade_validateServiceNameA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)
	svcB := service.Service{
		ID:              "validate-service-name-B",
		ParentServiceID: "validate-service-name-A",
		Name:            "TestFacade_validateServiceNameB",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)
	// parent not exist
	err := ft.Facade.validateServiceName(ft.CTX, &service.Service{
		ID:              "validate-service-name-C",
		ParentServiceID: "bogus-parent",
		Name:            "TestFacade_validateServiceNameB",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	})
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	// collision
	err = ft.Facade.validateServiceName(ft.CTX, &service.Service{
		ID:              "validate-service-name-C",
		ParentServiceID: "validate-service-name-A",
		Name:            "TestFacade_validateServiceNameB",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	})
	c.Assert(err, Equals, ErrServiceCollision)
	// success
	err = ft.Facade.validateServiceName(ft.CTX, &service.Service{
		ID:              "validate-service-name-C",
		ParentServiceID: "validate-service-name-A",
		Name:            "TestFacade_validateServiceNameC",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	})
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceTenant(c *C) {
	svcA := service.Service{
		ID:           "validate-service-tenant-A",
		Name:         "TestFacade_validateServiceTenantA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)
	svcB := service.Service{
		ID:              "validate-service-tenant-B",
		ParentServiceID: "validate-service-tenant-A",
		Name:            "TestFacade_validateServiceTenantA",
		DeploymentID:    "deployment-id",
		PoolID:          "pool-id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)
	svcC := service.Service{
		ID:           "validate-service-tenant-C",
		Name:         "TestFacade_validateServiceTenantC",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcC), IsNil)
	// empty tenant field
	err := ft.Facade.validateServiceTenant(ft.CTX, "", "")
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	err = ft.Facade.validateServiceTenant(ft.CTX, svcA.ID, "")
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	err = ft.Facade.validateServiceTenant(ft.CTX, "", svcB.ID)
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	// service not found
	err = ft.Facade.validateServiceTenant(ft.CTX, "bogus-service", svcC.ID)
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	err = ft.Facade.validateServiceTenant(ft.CTX, svcA.ID, "bogus-service")
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	// not matching tenant
	err = ft.Facade.validateServiceTenant(ft.CTX, svcB.ID, svcC.ID)
	c.Assert(err, Equals, ErrTenantDoesNotMatch)
	err = ft.Facade.validateServiceTenant(ft.CTX, svcA.ID, svcB.ID)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) setup_validateServiceStart(c *C, endpoints ...service.ServiceEndpoint) *service.Service {
	err := ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: "test-pool"})
	c.Assert(err, IsNil)
	svc := service.Service{
		ID:           "validate-service-start",
		Name:         "TestFacade_validateServiceStart",
		DeploymentID: "deployment-id",
		PoolID:       "test-pool",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	svc.Endpoints = endpoints
	for _, ep := range endpoints {
		for _, vhost := range ep.VHostList {
			ft.zzk.On("GetVHost", vhost.Name).Return(svc.ID, ep.Application, nil).Once()
		}
		for _, port := range ep.PortList {
			ft.zzk.On("GetPublicPort", port.PortAddr).Return(svc.ID, ep.Application, nil).Once()
		}
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svc), IsNil)
	return &svc
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStart_emergencyShutdownFlagged(c *C) {
	// successfully add address assignment, vhost, and port
	ep1 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep1",
		Application: "ep1",
		Purpose:     "export",
		AddressConfig: servicedefinition.AddressResourceConfig{
			Port:     1234,
			Protocol: "tcp",
		},
	})
	svc := ft.setup_validateServiceStart(c, ep1)
	// set up an address assignment for ep1
	err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{
		PoolID:        svc.PoolID,
		IP:            "192.168.22.12",
		Netmask:       "255.255.255.0",
		BindInterface: "eth0",
	})
	c.Assert(err, IsNil)
	err = ft.Facade.AssignIPs(ft.CTX, addressassignment.AssignmentRequest{
		ServiceID:      svc.ID,
		AutoAssignment: false,
		IPAddress:      "192.168.22.12",
	})
	c.Assert(err, IsNil)
	ft.zzk.On("GetVHost", "vh1").Return("", "", nil)
	ft.zzk.On("GetPublicPort", ":1234").Return("", "", nil)
	// Make service have EmergencyShutdown flagged
	svc.EmergencyShutdown = true
	err = ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrEmergencyShutdownNoOp)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStart_missingAddressAssignment(c *C) {
	// set up the endpoint with a missing address assignment
	endpoint := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep1",
		Application: "ep1",
		Purpose:     "export",
		AddressConfig: servicedefinition.AddressResourceConfig{
			Port:     1234,
			Protocol: "tcp",
		},
	})
	svc := ft.setup_validateServiceStart(c, endpoint)
	err := ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrServiceMissingAssignment)

}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStart(c *C) {
	// successfully add address assignment, vhost, and port
	ep1 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep1",
		Application: "ep1",
		Purpose:     "export",
		AddressConfig: servicedefinition.AddressResourceConfig{
			Port:     1234,
			Protocol: "tcp",
		},
	})
	ep2 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep2",
		Application: "ep2",
		Purpose:     "export",
		VHostList: []servicedefinition.VHost{
			{
				Name:    "vh1",
				Enabled: true,
			},
		},
	})
	ep3 := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
		Name:        "ep3",
		Application: "ep3",
		Purpose:     "export",
		PortList: []servicedefinition.Port{
			{
				PortAddr: ":1234",
				Enabled:  true,
			},
		},
	})
	svc := ft.setup_validateServiceStart(c, ep1, ep2, ep3)
	// set up an address assignment for ep1
	err := ft.Facade.AddVirtualIP(ft.CTX, pool.VirtualIP{
		PoolID:        svc.PoolID,
		IP:            "192.168.22.12",
		Netmask:       "255.255.255.0",
		BindInterface: "eth0",
	})
	c.Assert(err, IsNil)
	err = ft.Facade.AssignIPs(ft.CTX, addressassignment.AssignmentRequest{
		ServiceID:      svc.ID,
		AutoAssignment: false,
		IPAddress:      "192.168.22.12",
	})
	c.Assert(err, IsNil)
	ft.zzk.On("GetVHost", "vh1").Return("", "", nil)
	ft.zzk.On("GetPublicPort", ":1234").Return("", "", nil)
	err = ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) setup_validateServiceStop(c *C) *service.Service {
	err := ft.Facade.AddResourcePool(ft.CTX, &pool.ResourcePool{ID: "test-pool"})
	c.Assert(err, IsNil)
	svc := service.Service{
		ID:                "validate-service-stop",
		Name:              "TestFacade_validateServiceStop",
		DeploymentID:      "deployment-id",
		PoolID:            "test-pool",
		Launch:            "auto",
		DesiredState:      int(service.SVCStop),
		EmergencyShutdown: false,
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svc), IsNil)
	return &svc
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStop_emergencyShutdownFlagged(c *C) {
	svc := ft.setup_validateServiceStop(c)
	// Test stopping a service that has been emergency stopped
	svc.EmergencyShutdown = true
	err := ft.Facade.validateServiceStop(ft.CTX, svc, false)
	c.Assert(err, Equals, ErrEmergencyShutdownNoOp)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStop(c *C) {
	// Test stopping a service that has not been emergency stopped
	svc := ft.setup_validateServiceStop(c)
	err := ft.Facade.validateServiceStop(ft.CTX, svc, false)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStop_emergencyTrue(c *C) {
	// Test emergency stopping a service that has been stopped
	svc := ft.setup_validateServiceStop(c)
	err := ft.Facade.validateServiceStop(ft.CTX, svc, true)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_validateService_badServiceID(t *C) {
	_, err := ft.Facade.validateServiceUpdate(ft.CTX, &service.Service{ID: "badID"})
	t.Assert(err, ErrorMatches, "No such entity {kind:service, id:badID}")
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceAdd_EnableDuplicatePublicEndpoint(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				Application: "zproxy",
				Name:        "zproxy",
				PortNumber:  8080,
				Protocol:    "tcp",
				Purpose:     "export",
				PortList: []servicedefinition.Port{
					servicedefinition.Port{
						PortAddr: ":22222",
						Enabled:  true,
						UseTLS:   true,
						Protocol: "https",
					},
				},
				VHostList: []servicedefinition.VHost{
					servicedefinition.VHost{
						Name:    "zproxy",
						Enabled: true,
					},
				},
			},
		},
	}

	ft.zzk.On("GetVHost", "zproxy").Return("svc2", "zproxy", nil).Twice()
	ft.zzk.On("GetPublicPort", ":22222").Return("svc2", "zproxy", nil).Twice()
	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return
	}

	svc2, err := ft.Facade.GetService(ft.CTX, svc.ID)
	t.Assert(err, IsNil)
	t.Check(svc2.Endpoints[0].PortList[0].Enabled, Equals, false)
	t.Check(svc2.Endpoints[0].VHostList[0].Enabled, Equals, false)
	svc.Endpoints[0].PortList[0].Enabled = true
	svc.Endpoints[0].VHostList[0].Enabled = true

	t.Assert(ft.Facade.UpdateService(ft.CTX, svc), NotNil)
}

// Add using the servicedefition defines.
func (ft *FacadeIntegrationTest) TestFacade_validateServiceAdd_InvalidServiceOptions(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_InvalidServiceOptions",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		HostPolicy: servicedefinition.RequireSeparate,
		ChangeOptions: []servicedefinition.ChangeOption{
			servicedefinition.RestartAllOnInstanceChanged,
		},
	}

	err := ft.Facade.AddService(ft.CTX, svc)
	// We should have gotten an error here that these options are invalid together.
	t.Assert(err, NotNil)
}

// Make these all uppercase; case should not matter for these options in the service def.
func (ft *FacadeIntegrationTest) TestFacade_validateServiceAdd_InvalidServiceOptions2(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_InvalidServiceOptions",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		HostPolicy: "REQUIRE_SEPARATE",
		ChangeOptions: []servicedefinition.ChangeOption{
			"RESTARTALLONINSTANCECHANGED",
		},
	}

	err := ft.Facade.AddService(ft.CTX, svc)
	// We should have gotten an error here that these options are invalid together.
	t.Assert(err, NotNil)
}

// We should get an error if we add a service with an invalid ChangeOption.
func (ft *FacadeIntegrationTest) TestFacade_validateServiceAdd_InvalidServiceOptions3(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_InvalidServiceOptions",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		ChangeOptions: []servicedefinition.ChangeOption{
			"InvalidChangeOption",
		},
	}

	err := ft.Facade.AddService(ft.CTX, svc)
	// We should have gotten an error here the change option is invalid.
	t.Assert(err, NotNil)
}

// We should get an error if we try to update a service with an invalid set of options.
func (ft *FacadeIntegrationTest) TestFacade_validateServiceUpdate_InvalidServiceOptions(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_InvalidServiceOptions",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		ChangeOptions: []servicedefinition.ChangeOption{
			servicedefinition.RestartAllOnInstanceChanged,
		},
	}

	err := ft.Facade.AddService(ft.CTX, svc)
	t.Assert(err, IsNil)

	svc.HostPolicy = servicedefinition.RequireSeparate
	err = ft.Facade.UpdateService(ft.CTX, svc)
	t.Assert(err, NotNil) // This should have returned an ErrInvalidServiceOption error.
}

func (ft *FacadeIntegrationTest) TestFacade_migrateServiceConfigs_noConfigs(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, nil)
	t.Assert(err, IsNil)

	err = ft.Facade.MigrateService(ft.CTX, *newSvc)
	t.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_migrateServiceConfigs_noChanges(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)

	err = ft.Facade.MigrateService(ft.CTX, *newSvc)
	t.Assert(err, IsNil)
}

// Verify migration of configuration data when the user has not changed any config files
func (ft *FacadeIntegrationTest) TestFacade_migrateService_withoutUserConfigChanges(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)
	newSvc.ConfigFiles = nil

	err = ft.Facade.MigrateService(ft.CTX, *newSvc)
	t.Assert(err, IsNil)

	result, err := ft.Facade.GetService(ft.CTX, newSvc.ID)
	t.Assert(err, IsNil)

	t.Assert(result.Description, Equals, newSvc.Description)
	t.Assert(result.OriginalConfigs, DeepEquals, newSvc.OriginalConfigs)
	t.Assert(result.ConfigFiles, DeepEquals, newSvc.OriginalConfigs)

	confs, err := ft.getConfigFiles(result)
	t.Assert(err, IsNil)
	t.Assert(len(confs), Equals, 0)
}

func (ft *FacadeIntegrationTest) TestFacade_GetServiceEndpoints_UndefinedService(t *C) {
	endpointMap, err := ft.Facade.GetServiceEndpoints(ft.CTX, "undefined", true, true, true)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "Could not find service undefined.*")
	t.Assert(endpointMap, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_GetServiceEndpoints_ZKUnavailable(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	errorStub := fmt.Errorf("Stub for cannot-connect-to-zookeeper")
	ft.zzk.On("GetServiceStates", ft.CTX, svc.PoolID, svc.ID).Return([]zks.State{}, errorStub)

	endpointMap, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "Could not get service states for service .*")
	t.Assert(endpointMap, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_GetServiceEndpoints_ServiceNotRunning(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)

	state := zks.State{
		ServiceID:  svc.ID,
		InstanceID: 0,
	}
	for _, ep := range svc.Endpoints {
		if ep.Purpose == "export" {
			state.Exports = append(state.Exports, zks.ExportBinding{
				Application: ep.Application,
			})
		} else {
			state.Imports = append(state.Imports, zks.ImportBinding{
				Application: ep.Application,
			})
		}
	}
	ft.zzk.On("GetServiceStates", ft.CTX, svc.PoolID, svc.ID).Return([]zks.State{state}, nil)

	endpoints, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, IsNil)
	t.Assert(endpoints, NotNil)
	t.Assert(len(endpoints), Equals, 2)
	t.Assert(endpoints[0].Endpoint.ServiceID, Equals, svc.ID)
	t.Assert(endpoints[0].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[0].Endpoint.Application, Equals, "test_ep_1")
	t.Assert(endpoints[1].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[1].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[1].Endpoint.Application, Equals, "test_ep_2")
}

func (ft *FacadeIntegrationTest) TestFacade_GetServiceEndpoints_ServiceRunning(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)

	state := zks.State{
		ServiceID:  svc.ID,
		InstanceID: 0,
	}
	for _, ep := range svc.Endpoints {
		if ep.Purpose == "export" {
			state.Exports = append(state.Exports, zks.ExportBinding{
				Application: ep.Application,
			})
		} else {
			state.Imports = append(state.Imports, zks.ImportBinding{
				Application: ep.Application,
			})
		}
	}

	states := make([]zks.State, 2)
	for i := range states {
		states[i] = state
		states[i].InstanceID = i
	}

	ft.zzk.On("GetServiceStates", ft.CTX, svc.PoolID, svc.ID).Return(states, nil)
	// don't worry about mocking the ZK validation
	ft.zzk.On("GetServiceEndpoints", svc.ID, svc.ID, mock.AnythingOfType("*[]applicationendpoint.ApplicationEndpoint")).Return(nil)

	endpoints, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, IsNil)
	t.Assert(endpoints, NotNil)
	t.Assert(len(endpoints), Equals, 4)
	t.Assert(endpoints[0].Endpoint.ServiceID, Equals, svc.ID)
	t.Assert(endpoints[0].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[0].Endpoint.Application, Equals, "test_ep_1")
	t.Assert(endpoints[1].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[1].Endpoint.InstanceID, Equals, 0)
	t.Assert(endpoints[1].Endpoint.Application, Equals, "test_ep_2")
	t.Assert(endpoints[2].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[2].Endpoint.InstanceID, Equals, 1)
	t.Assert(endpoints[2].Endpoint.Application, Equals, "test_ep_1")
	t.Assert(endpoints[3].Endpoint.ServiceID, Equals, "svc1")
	t.Assert(endpoints[3].Endpoint.InstanceID, Equals, 1)
	t.Assert(endpoints[3].Endpoint.Application, Equals, "test_ep_2")
}

func (ft *FacadeIntegrationTest) setupServiceWithEndpoints(t *C) (*service.Service, error) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_GetServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name: "test_ep_1", Application: "test_ep_1", Purpose: "export",
					VHostList: []servicedefinition.VHost{servicedefinition.VHost{Name: "test_vhost_1", Enabled: true}},
					PortList:  []servicedefinition.Port{servicedefinition.Port{PortAddr: ":1234", Enabled: true}},
				},
			),
		},
	}

	ft.zzk.On("GetVHost", "test_vhost_1").Return("", "", nil).Once()
	ft.zzk.On("GetPublicPort", ":1234").Return("", "", nil).Once()
	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return nil, err
	}
	return &svc, nil
}

// Test a trivial migration of a single property
func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Modify_Success(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	oldSvc, err := ft.Facade.GetService(ft.CTX, "original_service_id_tenant")
	t.Assert(err, IsNil)

	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.Description = "migrated_service"

	request := dao.ServiceMigrationRequest{
		ServiceID: newSvc.ID,
		Modified:  []*service.Service{&newSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)

	out, err := ft.Facade.GetService(ft.CTX, newSvc.ID)
	t.Assert(err, IsNil)
	t.Assert(out.Description, Equals, "migrated_service")
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Modify_Fail(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	oldSvc, err := ft.Facade.GetService(ft.CTX, "original_service_id_child_0")
	t.Assert(err, IsNil)

	newSvc := service.Service{PoolID: "original_service_pool_id"}
	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "original_service_pool_id"}
	t.Assert(ft.Facade.AddResourcePool(ft.CTX, &rp), IsNil)

	// Make sure we fail if we give a bad id.
	newSvc = *oldSvc
	newSvc.ID = "some_unknown_id"
	request := dao.ServiceMigrationRequest{
		ServiceID: newSvc.ID,
		Modified:  []*service.Service{&newSvc},
	}
	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, ErrorMatches, "No such entity.*")

	// Make sure we fail if we give a bad parent id.
	newSvc = *oldSvc
	newSvc.ParentServiceID = "some_unknown_id"
	request = dao.ServiceMigrationRequest{
		ServiceID: newSvc.ID,
		Modified:  []*service.Service{&newSvc},
	}
	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, ErrorMatches, "No such entity.*")

	// Make sure we fail if we cause a name collision with an existing service
	newSvc = *oldSvc
	newSvc.Name = "original_service_name_child_1"
	request = dao.ServiceMigrationRequest{
		ServiceID: newSvc.ID,
		Modified:  []*service.Service{&newSvc},
	}
	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceCollision)

	// Make sure we fail if we set an invalid desired state.
	newSvc = *oldSvc
	newSvc.DesiredState = 9001
	request = dao.ServiceMigrationRequest{
		ServiceID: newSvc.ID,
		Modified:  []*service.Service{&newSvc},
	}
	err = ft.Facade.MigrateServices(ft.CTX, request)
	validationFailure := "9001 not in [1 0 2]"
	msg := fmt.Sprintf("error message '%q' contains %q", err.Error(), validationFailure)
	actual := fmt.Sprintf("%s is %v", msg, strings.Contains(err.Error(), validationFailure))
	expected := fmt.Sprintf("%s is true", msg)
	t.Assert(actual, Equals, expected)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Modify_FailDupNew(t *C) {
	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "default"}
	err := ft.Facade.AddResourcePool(ft.CTX, &rp)
	t.Assert(err, IsNil)

	err = ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	oldSvc, err := ft.Facade.GetService(ft.CTX, "original_service_id_child_0")
	t.Assert(err, IsNil)

	newSvc1 := service.Service{PoolID: "default"}
	newSvc1 = *oldSvc
	newSvc1.Name = "ModifiedName1"
	newSvc1.Description = "migrated_service"

	oldSvc, err = ft.Facade.GetService(ft.CTX, "original_service_id_child_1")
	t.Assert(err, IsNil)

	newSvc2 := service.Service{PoolID: "default"}
	newSvc2 = *oldSvc
	newSvc2.Name = newSvc1.Name
	newSvc2.Description = "migrated_service"

	request := dao.ServiceMigrationRequest{
		ServiceID: oldSvc.ParentServiceID,
		Modified:  []*service.Service{&newSvc1, &newSvc2},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceCollision)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Added_Success(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	newSvc := ft.createNewChildService(t)
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Added:     []*service.Service{newSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)

	t.Assert(err, IsNil)
	ft.assertServiceAdded(t, newSvc)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Added_FailDup(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	// Only change the ID of the added service, such that the Name will conflict the existing child service
	newSvc := ft.createNewChildService(t)
	newSvc.Name = "original_service_name_child_0"
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Added:     []*service.Service{newSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)

	t.Assert(err, Equals, ErrServiceCollision)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Added_FailDupNew(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	newSvc1 := ft.createNewChildService(t)
	newSvc2 := ft.createNewChildService(t)
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Added:     []*service.Service{newSvc1, newSvc2},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)

	t.Assert(err, Equals, ErrServiceCollision)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_AddedAndModified(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	newSvc := ft.createNewChildService(t)
	oldSvc, err := ft.Facade.GetService(ft.CTX, "original_service_id_child_0")
	t.Assert(err, IsNil)

	modSvc := service.Service{}
	modSvc = *oldSvc
	modSvc.Description = "migrated_service"

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Added:     []*service.Service{newSvc},
		Modified:  []*service.Service{&modSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)

	t.Assert(err, IsNil)
	ft.assertServiceAdded(t, newSvc)

	out, err := ft.Facade.GetService(ft.CTX, modSvc.ID)
	t.Assert(err, IsNil)
	t.Assert(out.Description, Equals, "migrated_service")
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_AddedAndModified_FailDup(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	newSvc := ft.createNewChildService(t)
	oldSvc, err := ft.Facade.GetService(ft.CTX, "original_service_id_child_0")
	t.Assert(err, IsNil)

	modSvc := service.Service{}
	modSvc = *oldSvc
	modSvc.Name = newSvc.Name
	modSvc.Description = "migrated_service"

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Added:     []*service.Service{newSvc},
		Modified:  []*service.Service{&modSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)

	t.Assert(err, Equals, ErrServiceCollision)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_AddedAndDeployed_FailDup(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	newSvc := ft.createNewChildService(t)
	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = newSvc.ParentServiceID
	deployRequest.Service.Name = newSvc.Name

	request := dao.ServiceMigrationRequest{
		ServiceID: newSvc.ParentServiceID,
		Added:     []*service.Service{newSvc},
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)

	// Conceptually, this is the same condition as ErrServiceCollision, but since it's caught in deployment
	//	the error value is a different string.
	t.Assert(err, ErrorMatches, "service exists")
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_Success(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	deployRequest := ft.createServiceDeploymentRequest(t)
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)

	svcs, err := ft.Facade.GetServices(ft.CTX, dao.ServiceRequest{TenantID: request.ServiceID})
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 4) // there should be 1 additional service
	foundAddedService := false
	for _, svc := range svcs {
		if svc.Name == deployRequest.Service.Name {
			foundAddedService = true
			break
		}
	}
	t.Assert(foundAddedService, Equals, true)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_FailDup(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = "original_service_id_tenant"
	deployRequest.Service.Name = "original_service_name_child_0"
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)

	// Conceptually, this is the same condition as ErrServiceCollision, but since it's caught in deployment
	//	the error value is a different string.
	t.Assert(err, ErrorMatches, "service exists")
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_FailDupNew(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	deployRequest1 := ft.createServiceDeploymentRequest(t)
	deployRequest1.ParentID = "original_service_id_tenant"
	deployRequest1.Service.Name = "deploy_service_name"
	deployRequest2 := ft.createServiceDeploymentRequest(t)
	deployRequest2.ParentID = "original_service_id_tenant"
	deployRequest2.Service.Name = deployRequest1.Service.Name
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest1, deployRequest2},
	}

	ft.dfs.On("Download",
		deployRequest1.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)

	// Conceptually, this is the same condition as ErrServiceCollision, but since it's caught in deployment
	//	the error value is a different string.
	t.Assert(err, ErrorMatches, "service exists")
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_FailInvalidParentID(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = "bogus-parent"
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, ErrorMatches, "No such entity.*")
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_FailInvalidServiceDefinition(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	deployRequest := ft.createServiceDeploymentRequest(t)
	//deployRequest.Service.ImageID = ""
	deployRequest.Service.Launch = "bogus-launch"
	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Check(strings.Contains(err.Error(), "string bogus-launch not in [auto manual]"), Equals, true)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_FailDupeEndpointsWithinANewService(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	originalID := "original_service_id_child_1"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	// Create a new service has two endpoints with the same name
	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.ID = oldSvc.ID + "_CLONE"
	newSvc.Name = oldSvc.Name + "_CLONE"
	newSvc.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(
			servicedefinition.EndpointDefinition{
				Name:        "original_service_endpoint_name_child_1",
				Application: "original_service_endpoint_application_child_1",
				Purpose:     "export",
			},
		),
		service.BuildServiceEndpoint(
			servicedefinition.EndpointDefinition{
				Name:        "original_service_endpoint_name_child_1",
				Application: "original_service_endpoint_application_child_1",
				Purpose:     "export",
			},
		),
	}

	request := dao.ServiceMigrationRequest{
		ServiceID: originalID,
		Added:     []*service.Service{&newSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_FailDupeEndpointsAcrossNewServices(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	originalID := "original_service_id_child_1"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	// Create 2 new services which have the same endpoints
	newSvc1 := service.Service{}
	newSvc1 = *oldSvc
	newSvc1.ID = oldSvc.ID + "_CLONE1"
	newSvc1.Name = oldSvc.Name + "_CLONE1"
	newSvc1.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(
			servicedefinition.EndpointDefinition{
				Name:        "original_service_endpoint_name_child_1",
				Application: "original_service_endpoint_application_child_1",
				Purpose:     "export",
			},
		),
	}
	newSvc2 := service.Service{}
	newSvc2 = *oldSvc
	newSvc2.ID = oldSvc.ID + "_CLONE2"
	newSvc2.Name = oldSvc.Name + "_CLONE2"
	newSvc2.Endpoints = []service.ServiceEndpoint{newSvc1.Endpoints[0]}

	request := dao.ServiceMigrationRequest{
		ServiceID: originalID,
		Added:     []*service.Service{&newSvc1, &newSvc2},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_FailDupeEndpointsAcrossNewAndModifiedServices(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	originalID := "original_service_id_child_1"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	newSvc1 := service.Service{}
	newSvc1 = *oldSvc
	newSvc1.ID = oldSvc.ID + "_CLONE1"
	newSvc1.Name = oldSvc.Name + "_CLONE1"
	newSvc1.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(
			servicedefinition.EndpointDefinition{
				Name:        "original_service_endpoint_name_child_1",
				Application: "original_service_endpoint_application_child_1",
				Purpose:     "export",
			},
		),
	}
	modSvc := service.Service{}
	modSvc = *oldSvc
	modSvc.Endpoints = []service.ServiceEndpoint{newSvc1.Endpoints[0]}

	request := dao.ServiceMigrationRequest{
		ServiceID: originalID,
		Added:     []*service.Service{&newSvc1},
		Modified:  []*service.Service{&modSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_FailDupeEndpointsAcrossNewAndDeployedServices(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	originalID := "original_service_id_child_1"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.ID = oldSvc.ID + "_CLONE1"
	newSvc.Name = oldSvc.Name + "_CLONE1"
	newSvc.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(
			servicedefinition.EndpointDefinition{
				Name:        "original_service_endpoint_name_child_1",
				Application: "original_service_endpoint_application_child_1",
				Purpose:     "export",
			},
		),
	}

	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Added:     []*service.Service{&newSvc},
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_FailDupeEndpointsWithTemplate(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	// Try to deploy 2 services with the same parent and same templated endpoint
	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = "original_service_id_child_1"
	deployRequest.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	deployRequest2 := ft.createServiceDeploymentRequest(t)
	deployRequest2.ParentID = "original_service_id_child_1"
	deployRequest2.Service.Name = "added-service-2"
	deployRequest2.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest, deployRequest2},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_Deploy_EndpointsWithTemplate(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	// Deploy 2 services with the same templated endpoint but different parents
	deployRequest := ft.createServiceDeploymentRequest(t)
	deployRequest.ParentID = "original_service_id_child_1"
	deployRequest.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	deployRequest2 := ft.createServiceDeploymentRequest(t)
	deployRequest2.ParentID = "original_service_id_child_0"
	deployRequest2.Service.Name = "added-service-2"
	deployRequest2.Service.Endpoints = []servicedefinition.EndpointDefinition{
		servicedefinition.EndpointDefinition{
			Name:        "original_service_endpoint_name_child_1",
			Application: "{{(parent .).Name}}_original_service_endpoint_application_child_1",
			Purpose:     "export",
		},
	}

	request := dao.ServiceMigrationRequest{
		ServiceID: "original_service_id_tenant",
		Deploy:    []*dao.ServiceDeploymentRequest{deployRequest, deployRequest2},
	}

	ft.dfs.On("Download",
		deployRequest.Service.ImageID,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return("mockImageId", nil)

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_FailDupeExistingEndpoint(t *C) {
	err := ft.setupMigrationTestWithEndpoints(t)
	t.Assert(err, IsNil)

	originalID := "original_service_id_child_1"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	// Create a service which has an endpoint that matches an existing service
	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.ID = oldSvc.ID + "_CLONE"
	newSvc.Name = oldSvc.Name + "_CLONE"

	request := dao.ServiceMigrationRequest{
		ServiceID: originalID,
		Added:     []*service.Service{&newSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Equals, ErrServiceDuplicateEndpoint)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_ModifiedWithEndpoint(t *C) {
	err := ft.setupMigrationTestWithEndpoints(t)
	t.Assert(err, IsNil)

	originalID := "original_service_id_child_1"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	// Create a service which has an endpoint that matches an existing service
	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.Name = oldSvc.Name + "_CLONE"

	// Modify the service and make sure it succeeds (no failure on dupe endpoint)
	request := dao.ServiceMigrationRequest{
		ServiceID: originalID,
		Modified:  []*service.Service{&newSvc},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_AddsLogFilter(t *C) {
	filter := logfilter.LogFilter{
		Name:	"filter1",
		Filter: "some filter",
		Version: "1.0",
	}
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	svc, err := ft.Facade.GetService(ft.CTX, "original_service_id_tenant")
	t.Assert(err, IsNil)

	request := dao.ServiceMigrationRequest{
		ServiceID: svc.ID,
		LogFilters: map[string]logfilter.LogFilter{
			filter.Name: filter,
		},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)

	var result *logfilter.LogFilter
	result, err = ft.Facade.logFilterStore.Get(ft.CTX, filter.Name, filter.Version)
	t.Assert(err, IsNil)
	t.Assert(result.Filter, Equals, filter.Filter)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_AddsLogFilterVersion(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	filter1 := logfilter.LogFilter{
		Name:	"filter1",
		Filter: "some filter",
		Version: "1.0",
	}
	err = ft.Facade.logFilterStore.Put(ft.CTX, &filter1)

	svc, err := ft.Facade.GetService(ft.CTX, "original_service_id_tenant")
	t.Assert(err, IsNil)

	filter2 := filter1
	filter2.Filter = "some new filter"
	filter2.Version = "2.0"
	request := dao.ServiceMigrationRequest{
		ServiceID: svc.ID,
		LogFilters: map[string]logfilter.LogFilter{
			filter2.Name: filter2,
		},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)

	var result *logfilter.LogFilter
	result, err = ft.Facade.logFilterStore.Get(ft.CTX, filter1.Name, filter1.Version)
	t.Assert(err, IsNil)

	result, err = ft.Facade.logFilterStore.Get(ft.CTX, filter2.Name, filter2.Version)
	t.Assert(err, IsNil)
	t.Assert(result.Filter, Equals, filter2.Filter)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_UpdatesLogFilter(t *C) {
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	filter := logfilter.LogFilter{
		Name:	"filter1",
		Filter: "some filter",
		Version: "1.0",
	}
	err = ft.Facade.logFilterStore.Put(ft.CTX, &filter)

	svc, err := ft.Facade.GetService(ft.CTX, "original_service_id_tenant")
	t.Assert(err, IsNil)

	filter.Filter = "some new filter"
	request := dao.ServiceMigrationRequest{
		ServiceID: svc.ID,
		LogFilters: map[string]logfilter.LogFilter{
			filter.Name: filter,
		},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, IsNil)

	var result *logfilter.LogFilter
	result, err = ft.Facade.logFilterStore.Get(ft.CTX, filter.Name, filter.Version)
	t.Assert(err, IsNil)
	t.Assert(result.Filter, Equals, filter.Filter)
}

func (ft *FacadeIntegrationTest) TestFacade_MigrateServices_FailsLogFilter(t *C) {
	filter := logfilter.LogFilter{
		Name:	"filter1",
		Filter: "some filter",
	}
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	svc, err := ft.Facade.GetService(ft.CTX, "original_service_id_tenant")
	t.Assert(err, IsNil)

	request := dao.ServiceMigrationRequest{
		ServiceID: svc.ID,
		LogFilters: map[string]logfilter.LogFilter{
			filter.Name: filter,
		},
	}

	err = ft.Facade.MigrateServices(ft.CTX, request)
	t.Assert(err, Not(IsNil))
	t.Assert(strings.Contains(err.Error(), "empty string for LogFilter.Version"), Equals, true)

	_, err = ft.Facade.logFilterStore.Get(ft.CTX, filter.Name, filter.Version)
	t.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
}

func (ft *FacadeIntegrationTest) TestFacade_ResolveServicePath(c *C) {
	svca := service.Service{
		ID:              "svcaid",
		PoolID:          "testPool",
		Name:            "svc_a",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id",
	}
	svcb := service.Service{
		ID:              "svcbid",
		PoolID:          "testPool",
		Name:            "svc_b",
		Launch:          "auto",
		ParentServiceID: "svcaid",
		DeploymentID:    "deployment_id",
	}
	svcc := service.Service{
		ID:              "svccid",
		PoolID:          "testPool",
		Name:            "svc_c",
		Launch:          "auto",
		ParentServiceID: "svcbid",
		DeploymentID:    "deployment_id",
	}
	svcd := service.Service{
		ID:              "svcdid",
		PoolID:          "testPool",
		Name:            "svc_d",
		Launch:          "auto",
		ParentServiceID: "svcbid",
		DeploymentID:    "deployment_id",
	}
	svcd2 := service.Service{
		ID:              "svcd2id",
		PoolID:          "testPool",
		Name:            "svc_d_2",
		Launch:          "auto",
		ParentServiceID: "svcbid",
		DeploymentID:    "deployment_id",
	}
	svc2d := service.Service{
		ID:              "svc2did",
		PoolID:          "testPool",
		Name:            "2_svc_d",
		Launch:          "auto",
		ParentServiceID: "svcbid",
		DeploymentID:    "deployment_id",
	}
	svcdother := service.Service{
		ID:              "svcdotherid",
		PoolID:          "testPool",
		Name:            "svc_d",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id_2",
	}
	svcnoprefix1 := service.Service{
                ID:              "svcnoprefix1id",
                PoolID:          "testPool",
                Name:            "svc_noprefix",
                Launch:          "auto",
                ParentServiceID: "svcbid",
                DeploymentID:    "deployment_id",
        }
        svcnoprefix2 := service.Service{
                ID:              "svcnoprefix2id",
                PoolID:          "testPool",
                Name:            "svc_noprefix2",
                Launch:          "auto",
                ParentServiceID: "svcbid",
                DeploymentID:    "deployment_id",
        }


	c.Assert(ft.Facade.AddService(ft.CTX, svca), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcb), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcc), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcd), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcd2), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svc2d), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcdother), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcnoprefix1), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcnoprefix2), IsNil)

	ft.assertPathResolvesToServices(c, "/svc_a/svc_b/svc_c", false, svcc)
	ft.assertPathResolvesToServices(c, "svc_a/svc_b/svc_c", false, svcc)
	ft.assertPathResolvesToServices(c, "svc_b/svc_c", false, svcc)
	ft.assertPathResolvesToServices(c, "/svc_b/svc_c", false, svcc)
	ft.assertPathResolvesToServices(c, "svc_c", false, svcc)
	ft.assertPathResolvesToServices(c, "/svc_c", false, svcc)

	ft.assertPathResolvesToServices(c, "/svc_a", false, svca)
	ft.assertPathResolvesToServices(c, "svc_a", false, svca)
	ft.assertPathResolvesToServices(c, "/svc_b", false, svcb)
	ft.assertPathResolvesToServices(c, "svc_b", false, svcb)

	// Default is substring match
	ft.assertPathResolvesToServices(c, "svc_d", false, svcd, svcd2, svc2d, svcdother)
	ft.assertPathResolvesToServices(c, "2", false, svcd2, svc2d, svcnoprefix2)

	// Leading slash indicates nothing special
	ft.assertPathResolvesToServices(c, "/svc_d", false, svcd, svcd2, svc2d, svcdother)
	ft.assertPathResolvesToServices(c, "/vc_d", false, svcd, svcd2, svc2d, svcdother)

	// Must be able to restrict by deployment ID
	ft.assertPathResolvesToServices(c, "deployment_id/svc_d", false, svcd, svcd2, svc2d)
	ft.assertPathResolvesToServices(c, "deployment_id_2/svc_d", false, svcdother)

	// Path has to exist underneath that deployment to match
	ft.assertPathResolvesToServices(c, "deployment_id/svc_b/svc_d", false, svcd, svcd2, svc2d)
	ft.assertPathResolvesToServices(c, "deployment_id_2/svc_b/svc_d", false)

	// Make sure invalid matches don't match
	ft.assertPathResolvesToServices(c, "notathing", false)
	ft.assertPathResolvesToServices(c, "svc_d/svc_b", false)
	ft.assertPathResolvesToServices(c, "sv_a", false)

	// Empty paths shouldn't match anything
	ft.assertPathResolvesToServices(c, "/", false)
	ft.assertPathResolvesToServices(c, "", false)

	// Test name no prefix matching
	ft.assertPathResolvesToServices(c, "svc_noprefix", false, svcnoprefix1, svcnoprefix2)
	ft.assertPathResolvesToServices(c, "svc_noprefix2", false, svcnoprefix2)
	ft.assertPathResolvesToServices(c, "svc_noprefix", true, svcnoprefix1)

}

func (ft *FacadeIntegrationTest) TestFacade_StoppingParentStopsChildren(c *C) {
	svc := service.Service{
		ID:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCRun),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	childService1 := service.Service{
		ID:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
	}
	childService2 := service.Service{
		ID:              "childService2",
		Name:            "childservice2",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
	}
	// add a service with a subservice
	var err error

	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "default"}
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService2); err != nil {
		c.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}

	// start the service
	if _, err = ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, true}); err != nil {
		c.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}
	// stop the parent
	if _, err = ft.Facade.StopService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, true}); err != nil {
		c.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}
	// verify the children have all stopped
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	for _, subService := range services {
		if subService.DesiredState == int(service.SVCRun) && subService.ParentServiceID == "ParentServiceID" {
			c.Errorf("Was expecting child services to be stopped %v", subService)
		}
	}
}

func (ft *FacadeIntegrationTest) TestFacade_EmergencyStopService_Synchronous(c *C) {
	// We have to reset the zzk mocks to replace what is in SetUpTest
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Once()
	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), service.SVCRun,
		mock.AnythingOfType("<-chan interface {}")).Return(nil)

	// Set up mocks to handle stopping services and waiting
	stoppedChannels := make(map[string]chan interface{})
	stoppedChannels["ParentServiceID"] = make(chan interface{})
	stoppedChannels["childService1"] = make(chan interface{})
	stoppedChannels["childService2"] = make(chan interface{})
	stoppedChannels["childService3"] = make(chan interface{})
	stoppedChannels["childService4"] = make(chan interface{})
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Run(func(args mock.Arguments) {
		svcs := args.Get(2).([]*service.Service)
		for _, s := range svcs {
			if s.DesiredState == int(service.SVCStop) {
				if ch, ok := stoppedChannels[s.ID]; ok {
					// Spawn a thread that will sleep 1 second and then close the channel
					go func() {
						time.Sleep(time.Second)
						close(ch)
					}()
				}
			}
		}
	})

	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), service.SVCStop,
		mock.AnythingOfType("<-chan interface {}")).Return(nil).Run(func(args mock.Arguments) {
		s := args.Get(0).(*service.Service)
		cancel := args.Get(2).(<-chan interface{})
		if ch, ok := stoppedChannels[s.ID]; ok {
			// Wait for the channel or cancel before returning
			select {
			case <-ch:
			case <-cancel:
			}
		}
	})

	// add a service with 4 subservices
	svc := service.Service{
		ID:                     "ParentServiceID",
		Name:                   "ParentService",
		Startup:                "/usr/bin/ping -c localhost",
		Description:            "Ping a remote host a fixed number of times",
		Instances:              1,
		InstanceLimits:         domain.MinMax{1, 1, 1},
		ImageID:                "test/pinger",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		DesiredState:           int(service.SVCRun),
		Launch:                 "auto",
		Endpoints:              []service.ServiceEndpoint{},
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		EmergencyShutdownLevel: 0,
		StartLevel:             1,
	}
	childService1 := service.Service{
		ID:                     "childService1",
		Name:                   "childservice1",
		Launch:                 "auto",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		Startup:                "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:        "ParentServiceID",
		EmergencyShutdownLevel: 1,
	}
	childService2 := service.Service{
		ID:                     "childService2",
		Name:                   "childservice2",
		Launch:                 "auto",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		Startup:                "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID:        "ParentServiceID",
		EmergencyShutdownLevel: 2,
	}
	childService3 := service.Service{
		ID:                     "childService3",
		Name:                   "childservice3",
		Launch:                 "auto",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		Startup:                "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID:        "ParentServiceID",
		EmergencyShutdownLevel: 0,
		StartLevel:             0,
	}
	childService4 := service.Service{
		ID:                     "childService4",
		Name:                   "childservice4",
		Launch:                 "auto",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		Startup:                "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID:        "ParentServiceID",
		EmergencyShutdownLevel: 0,
		StartLevel:             0,
	}
	var err error

	// add the resource pool (DFS access required for testing emergency shutdown)
	rp := pool.ResourcePool{ID: "default", Permissions: pool.DFSAccess}
	ft.zzk.On("UpdateResourcePool", &rp).Return(nil)
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	// Mock that the mounts for this test are valid.
	ft.dfs.On("VerifyTenantMounts", "ParentServiceID").Return(nil)

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService2); err != nil {
		c.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService3); err != nil {
		c.Fatalf("Failed Loading Child Service 3: %+v, %s", childService3, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService4); err != nil {
		c.Fatalf("Failed Loading Child Service 4: %+v, %s", childService4, err)
	}

	// start the service
	if _, err = ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, true}); err != nil {
		c.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}

	// watch the services to make sure they shutdown in the correct order
	// Emergency shutdown order should be:
	//  (EL 1) childService1
	//  (EL 2) childService2
	//  (EL 0, SL 0) childService3, childService4
	//  (EL 0, SL 1) svc
	go func() {
		// childService2 should be the second service stopped
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-stoppedChannels["childService2"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService2 to stop")
		}

		// If this is stopped, level 1 must also be stopped
		select {
		case <-stoppedChannels["childService1"]:
		default:
			c.Fatalf("Level 2 stopped before Level 1")
		}
	}()

	go func() {
		// childService3 should stop after childservice2 and childService1
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-stoppedChannels["childService3"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService3 to stop")
		}

		// If this is stopped, levels 1 and 2 must also be stopped
		select {
		case <-stoppedChannels["childService2"]:
		default:
			c.Fatalf("Level 0 stopped before Level 2")
		}

		select {
		case <-stoppedChannels["childService1"]:
		default:
			c.Fatalf("Level 0 stopped before Level 1")
		}
	}()

	go func() {
		// childService4 should stop after childservice2 and childService1
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-stoppedChannels["childService4"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService3 to stop")
		}

		// If this is stopped, levels 1 and 2 must also be stopped
		select {
		case <-stoppedChannels["childService2"]:
		default:
			c.Fatalf("Level 0 stopped before Level 2")
		}

		select {
		case <-stoppedChannels["childService1"]:
		default:
			c.Fatalf("Level 0 stopped before Level 1")
		}
	}()

	// This channel is closed when the last service is stopped
	go func() {
		// svc should be the last service stopped
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-stoppedChannels["ParentServiceID"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for parent service to stop")
		}

		// If this is stopped, levels 1 and 2 must also be stopped
		select {
		case <-stoppedChannels["childService1"]:
		default:
			c.Fatalf("Level 0 stopped before Level 1")
		}

		select {
		case <-stoppedChannels["childService2"]:
		default:
			c.Fatalf("Level 0 stopped before Level 2")
		}

		// Level 0 with StartLevel 0 must also be stopped
		select {
		case <-stoppedChannels["childService3"]:
		default:
			c.Fatalf("Level 0, StartLevel 1 stopped before Level 0, StartLevel 0")
		}

		select {
		case <-stoppedChannels["childService4"]:
		default:
			c.Fatalf("Level 0, StartLevel 1 stopped before Level 0, StartLevel 0")
		}
	}()

	// emergency stop the parent synchronously
	if _, err = ft.Facade.EmergencyStopService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"ParentServiceID"}, AutoLaunch: true, Synchronous: true}); err != nil {
		c.Fatalf("Unable to emergency stop parent service: %+v, %s", svc, err)
	}

	// For a synchronous call, make sure the services are stopped before the method returns
	select {
	case <-stoppedChannels["ParentServiceID"]:
	default:
		c.Fatalf("Method returned before services stopped on synchronous call")
	}

	// verify all services have EmergencyShutDown set to true
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	for _, s := range services {
		c.Assert(s.EmergencyShutdown, Equals, true)
	}
}

func (ft *FacadeIntegrationTest) TestFacade_EmergencyStopService_Asynchronous(c *C) {
	// We have to reset the zzk mocks to replace what is in SetUpTest
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Once()
	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), service.SVCRun,
		mock.AnythingOfType("<-chan interface {}")).Return(nil)

	stoppedChannels := make(map[string]chan interface{})
	stoppedChannels["ParentServiceID"] = make(chan interface{})
	stoppedChannels["childService1"] = make(chan interface{})
	stoppedChannels["childService2"] = make(chan interface{})
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Run(func(args mock.Arguments) {
		svcs := args.Get(2).([]*service.Service)
		for _, s := range svcs {
			if s.DesiredState == int(service.SVCStop) {
				if ch, ok := stoppedChannels[s.ID]; ok {
					// Spawn a thread that will sleep 1 second and then close the channel
					go func() {
						time.Sleep(time.Second)
						close(ch)
					}()
				}
			}
		}
	})

	var waitServiceWG sync.WaitGroup
	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), service.SVCStop,
		mock.AnythingOfType("<-chan interface {}")).Return(nil).Run(func(args mock.Arguments) {
		waitServiceWG.Add(1)
		defer waitServiceWG.Done()
		s := args.Get(0).(*service.Service)
		cancel := args.Get(2).(<-chan interface{})
		if ch, ok := stoppedChannels[s.ID]; ok {
			// Wait for the channel or cancel before returning
			select {
			case <-ch:
			case <-cancel:
			}
		}
	})

	svc := service.Service{
		ID:                     "ParentServiceID",
		Name:                   "ParentService",
		Startup:                "/usr/bin/ping -c localhost",
		Description:            "Ping a remote host a fixed number of times",
		Instances:              1,
		InstanceLimits:         domain.MinMax{1, 1, 1},
		ImageID:                "test/pinger",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		DesiredState:           int(service.SVCRun),
		Launch:                 "auto",
		Endpoints:              []service.ServiceEndpoint{},
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		EmergencyShutdownLevel: 0,
	}
	childService1 := service.Service{
		ID:                     "childService1",
		Name:                   "childservice1",
		Launch:                 "auto",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		Startup:                "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:        "ParentServiceID",
		EmergencyShutdownLevel: 1,
	}
	childService2 := service.Service{
		ID:                     "childService2",
		Name:                   "childservice2",
		Launch:                 "auto",
		PoolID:                 "default",
		DeploymentID:           "deployment_id",
		Startup:                "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID:        "ParentServiceID",
		EmergencyShutdownLevel: 2,
	}

	var err error

	// add the resource pool (DFS Access required to test emergency shutdown)
	rp := pool.ResourcePool{ID: "default", Permissions: pool.DFSAccess}
	ft.zzk.On("UpdateResourcePool", &rp).Return(nil)
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	// Mock that the mounts for this test are valid.
	ft.dfs.On("VerifyTenantMounts", "ParentServiceID").Return(nil)

	// add a service with 2 subservices
	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService2); err != nil {
		c.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}

	// start the service
	if _, err = ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, true}); err != nil {
		c.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}

	// watch the services to make sure they shutdown in the correct order
	go func() {
		// childService2 should be the second service stopped
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-stoppedChannels["childService2"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService2 to stop")
		}

		// If this is stopped, level 1 must also be stopped
		select {
		case <-stoppedChannels["childService1"]:
		default:
			c.Fatalf("Level 2 stopped before Level 1")
		}
	}()

	// This channel is closed when the last service is stopped
	go func() {
		// svc should be the last service stopped
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-stoppedChannels["ParentServiceID"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for parent service to stop")
		}

		// If this is stopped, levels 1 and 2 must also be stopped
		select {
		case <-stoppedChannels["childService1"]:
		default:
			c.Fatalf("Level 0 stopped before Level 1")
		}

		select {
		case <-stoppedChannels["childService2"]:
		default:
			c.Fatalf("Level 0 stopped before Level 2")
		}
	}()

	// emergency stop the parent asynchronously
	methodReturned := make(chan interface{})
	go func() {
		defer close(methodReturned)
		if _, err = ft.Facade.EmergencyStopService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"ParentServiceID"}, AutoLaunch: true, Synchronous: false}); err != nil {
			c.Fatalf("Unable to emergency stop parent service: %+v, %s", svc, err)
		}
	}()

	// Make sure the call was asynchronous
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-methodReturned:
	case <-stoppedChannels["ParentServiceID"]:
		c.Fatalf("Services stopped before method returned on asynchronous call")
	case <-timer.C:
		c.Fatalf("Timeout waiting for method to return")
	}

	// Wait for services to stop
	timer.Reset(10 * time.Second)
	select {
	case <-stoppedChannels["ParentServiceID"]:
	case <-timer.C:
		c.Fatalf("Timeout waiting for all services to stop")
	}

	// verify all services have EmergencyShutDown set to true
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	for _, s := range services {
		c.Assert(s.EmergencyShutdown, Equals, true)
	}

	// Wait for our mocked goroutines to return
	waitServiceWG.Wait()
}

func (ft *FacadeIntegrationTest) TestFacade_StartAndStopService_Synchronous(c *C) {
	// Set up mocks to handle starting services and waiting
	// We have to reset the zzk mocks to replace what is in SetUpTest
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	var mutex sync.RWMutex
	rp := pool.ResourcePool{ID: "default"}
	ft.zzk.On("UpdateResourcePool", &rp).Return(nil)
	scheduledChannels := make(map[string]chan interface{})
	scheduledChannels["ParentServiceID"] = make(chan interface{})
	scheduledChannels["childService1"] = make(chan interface{})
	scheduledChannels["childService2"] = make(chan interface{})
	scheduledChannels["childService3"] = make(chan interface{})
	scheduledChannels["childService4"] = make(chan interface{})
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Run(func(args mock.Arguments) {
		mutex.RLock()
		defer mutex.RUnlock()
		svcs := args.Get(2).([]*service.Service)
		for _, s := range svcs {
			if ch, ok := scheduledChannels[s.ID]; ok {
				close(ch)
			}
		}
	})

	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("service.DesiredState"),
		mock.AnythingOfType("<-chan interface {}")).Return(nil).Run(func(args mock.Arguments) {
		mutex.RLock()
		defer mutex.RUnlock()
		s := args.Get(0).(*service.Service)
		cancel := args.Get(2).(<-chan interface{})
		if ch, ok := scheduledChannels[s.ID]; ok {
			// Wait for the channel or cancel before returning
			select {
			case <-ch:
				// Sleep for 1 second and then return
				time.Sleep(time.Second)
			case <-cancel:
			}
		}
	})

	// add a service with 4 subservices
	svc := service.Service{
		ID:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCRun),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		StartLevel:     0,
	}
	childService1 := service.Service{
		ID:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      1,
	}
	childService2 := service.Service{
		ID:              "childService2",
		Name:            "childservice2",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      2,
	}
	childService3 := service.Service{
		ID:              "childService3",
		Name:            "childservice3",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      0,
	}
	childService4 := service.Service{
		ID:              "childService4",
		Name:            "childservice4",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      0,
	}
	var err error

	// add the resource pool (no permissions required)
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService2); err != nil {
		c.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService3); err != nil {
		c.Fatalf("Failed Loading Child Service 3: %+v, %s", childService3, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService4); err != nil {
		c.Fatalf("Failed Loading Child Service 4: %+v, %s", childService4, err)
	}

	// watch the services to make sure they start in the correct order
	// Start order should be:
	//  (SL 1) childService1
	//  (SL 2) childService2
	//  (SL 0) svc, childService3, childService4
	go func() {
		mutex.RLock()
		defer mutex.RUnlock()

		// childService2 should be the second service started
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-scheduledChannels["childService2"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService2 to stop")
		}

		// If this is started, level 1 must also be started
		select {
		case <-scheduledChannels["childService1"]:
		default:
			c.Fatalf("Level 2 started before Level 1")
		}
	}()

	go func() {
		mutex.RLock()
		defer mutex.RUnlock()

		// childService3 should start after childservice2 and childService1
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-scheduledChannels["childService3"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService3 to stop")
		}

		// If this is started, levels 1 and 2 must also be started
		select {
		case <-scheduledChannels["childService2"]:
		default:
			c.Fatalf("Level 0 started before Level 2")
		}

		select {
		case <-scheduledChannels["childService1"]:
		default:
			c.Fatalf("Level 0 started before Level 1")
		}
	}()

	go func() {
		mutex.RLock()
		defer mutex.RUnlock()

		// childService4 should start after childservice2 and childService1
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-scheduledChannels["childService4"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService4 to stop")
		}

		// If this is started, levels 1 and 2 must also be started
		select {
		case <-scheduledChannels["childService2"]:
		default:
			c.Fatalf("Level 0 started before Level 2")
		}

		select {
		case <-scheduledChannels["childService1"]:
		default:
			c.Fatalf("Level 0 started before Level 1")
		}
	}()

	go func() {
		mutex.RLock()
		defer mutex.RUnlock()

		// childService3 should start after childservice2 and childService1
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-scheduledChannels["ParentServiceID"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for svc to stop")
		}

		// If this is started, levels 1 and 2 must also be started
		select {
		case <-scheduledChannels["childService2"]:
		default:
			c.Fatalf("Level 0 started before Level 2")
		}

		select {
		case <-scheduledChannels["childService1"]:
		default:
			c.Fatalf("Level 0 started before Level 1")
		}
	}()

	// start the parent synchronously
	if _, err = ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, true}); err != nil {
		c.Fatalf("Unable to start parent service: %+v, %s", svc, err)
	}

	// For a synchronous call, make sure the services are started before the method returns
	mutex.RLock()
	select {
	case <-scheduledChannels["ParentServiceID"]:
	default:
		c.Fatalf("Method returned before services started on synchronous call")
	}
	mutex.RUnlock()

	// verify all services have desiredState set to run
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	for _, s := range services {
		c.Assert(int(s.DesiredState), Equals, int(service.SVCRun))
	}

	// Now stop the services and make sure they stop in the correct order
	mutex.Lock()
	// Re-create the channels
	scheduledChannels["ParentServiceID"] = make(chan interface{})
	scheduledChannels["childService1"] = make(chan interface{})
	scheduledChannels["childService2"] = make(chan interface{})
	scheduledChannels["childService3"] = make(chan interface{})
	scheduledChannels["childService4"] = make(chan interface{})
	mutex.Unlock()

	// watch the services to make sure they stop in the correct order
	// Stop order should be:
	//  (SL 0) svc, childService3, childService4
	//  (SL 2) childService2
	//  (SL 1) childService1
	go func() {
		mutex.RLock()
		defer mutex.RUnlock()

		// childService2 should be the second service stopped
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-scheduledChannels["childService2"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService2 to stop")
		}

		// If this is stopped, level 0 must also be stopped
		select {
		case <-scheduledChannels["childService3"]:
		default:
			c.Fatalf("Level 2 stopped before Level 0")
		}

		select {
		case <-scheduledChannels["childService4"]:
		default:
			c.Fatalf("Level 2 stopped before Level 0")
		}

		select {
		case <-scheduledChannels["ParentServiceID"]:
		default:
			c.Fatalf("Level 2 stopped before Level 0")
		}
	}()

	go func() {
		mutex.RLock()
		defer mutex.RUnlock()

		// childService1 should start last
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-scheduledChannels["childService1"]:
		case <-timer.C:
			c.Fatalf("Timeout waiting for childService1 to stop")
		}

		// If this is stopped, levels 0 and 2 must also be stopped
		select {
		case <-scheduledChannels["childService2"]:
		default:
			c.Fatalf("Level 0 stopped before Level 2")
		}

		select {
		case <-scheduledChannels["childService3"]:
		default:
			c.Fatalf("Level 1 stopped before Level 0")
		}

		select {
		case <-scheduledChannels["childService4"]:
		default:
			c.Fatalf("Level 1 stopped before Level 0")
		}

		select {
		case <-scheduledChannels["ParentServiceID"]:
		default:
			c.Fatalf("Level 1 stopped before Level 0")
		}
	}()

	// stop the parent synchronously
	if _, err = ft.Facade.StopService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, true}); err != nil {
		c.Fatalf("Unable to start parent service: %+v, %s", svc, err)
	}

	// For a synchronous call, make sure the services are stopped before the method returns
	mutex.RLock()
	select {
	case <-scheduledChannels["childService1"]:
	default:
		c.Fatalf("Method returned before services started on synchronous call")
	}
	mutex.RUnlock()

	// verify all services have desiredState set to stop
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	for _, s := range services {
		c.Assert(int(s.DesiredState), Equals, int(service.SVCStop))
	}
}

func (ft *FacadeIntegrationTest) TestFacade_RebalanceService_Asynchronous(c *C) {
	// Set up mocks to handle starting services and waiting
	// We have to reset the zzk mocks to replace what is in SetUpTest
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	var mutex sync.RWMutex
	rp := pool.ResourcePool{ID: "default"}
	ft.zzk.On("UpdateResourcePool", &rp).Return(nil)
	scheduledChannels := make(map[string]chan int)
	scheduledChannels["ParentServiceID"] = make(chan int, 1)
	scheduledChannels["childService1"] = make(chan int, 1)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Run(func(args mock.Arguments) {
		mutex.RLock()
		defer mutex.RUnlock()
		svcs := args.Get(2).([]*service.Service)
		for _, s := range svcs {
			if ch, ok := scheduledChannels[s.ID]; ok {
				ch <- s.DesiredState
			}
		}
	})

	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("service.DesiredState"),
		mock.AnythingOfType("<-chan interface {}")).Return(nil).Run(func(args mock.Arguments) {
		// Sleep for 1 second and then return
		time.Sleep(time.Second)
		return
	})

	// add a service with 1 subservice
	svc := service.Service{
		ID:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCRun),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		StartLevel:     0,
	}
	childService1 := service.Service{
		ID:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		DesiredState:    int(service.SVCRun),
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      1,
	}
	var err error

	// add the resource pool (no permissions required)
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}

	// start the services and consume the value off the channels
	count, err := ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, false})
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 2)
	<-scheduledChannels["ParentServiceID"]
	<-scheduledChannels["childService1"]

	// rebalance the parent asynchronously
	count, err = ft.Facade.RebalanceService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, false})
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 2)

	// Make sure both services stop first, in the correct order
	timer := time.NewTimer(2 * time.Second)
	var state int
	select {
	case state = <-scheduledChannels["ParentServiceID"]:
		c.Assert(state, Equals, int(service.SVCStop))
	case state = <-scheduledChannels["childService1"]:
		c.Fatalf("childService1 stopped before ParentService")
	case <-timer.C:
		c.Fatalf("Timeout waiting for ParentService to stop")
	}

	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(2 * time.Second)
	select {
	case state = <-scheduledChannels["childService1"]:
		c.Assert(state, Equals, int(service.SVCStop))
	case <-timer.C:
		c.Fatalf("Timeout waiting for childService1 to stop")
	}

	// Now both services should start, in the correct order
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(2 * time.Second)
	select {
	case state = <-scheduledChannels["childService1"]:
		c.Assert(state, Equals, int(service.SVCRun))
	case state = <-scheduledChannels["ParentServiceID"]:
		c.Fatalf("ParentService started before childService1")
	case <-timer.C:
		c.Fatalf("Timeout waiting for childService1 to Start")
	}

	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(2 * time.Second)
	select {
	case state = <-scheduledChannels["ParentServiceID"]:
		c.Assert(state, Equals, int(service.SVCRun))
	case <-timer.C:
		c.Fatalf("Timeout waiting for ParentService to start")
	}
}

func (ft *FacadeIntegrationTest) TestFacade_ModifyServiceWhilePending(c *C) {
	// If a service changes while pending, the change should not be reverted when it starts

	// Set up mocks to handle starting services and waiting
	// We have to reset the zzk mocks to replace what is in SetUpTest
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	rp := pool.ResourcePool{ID: "default"}
	ft.zzk.On("UpdateResourcePool", &rp).Return(nil)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil)

	releaseWait := make(chan time.Time)
	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("service.DesiredState"),
		mock.AnythingOfType("<-chan interface {}")).WaitUntil(releaseWait).Return(nil)

	// Create two services with different start levels
	svc := service.Service{
		ID:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCRun),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		StartLevel:     0,
	}
	childService1 := service.Service{
		ID:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      1,
	}

	var err error

	// add the resource pool (no permissions required)
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}

	// Start the services asynchronously.  After starting level 1, it will block until we close releaseWait
	if _, err = ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{[]string{"ParentServiceID"}, true, false}); err != nil {
		c.Fatalf("Unable to start parent service: %+v, %s", svc, err)
	}

	// Wait for level 1 to get scheduled
	ft.ssm.WaitScheduled("childService1")

	// Change parent service description
	svc.Description = "I changed this"
	err = ft.Facade.UpdateService(ft.CTX, svc)
	c.Assert(err, IsNil)

	// Let the service get scheduled
	close(releaseWait)
	ft.ssm.WaitScheduled("ParentServiceID")

	// Make sure the description change didn't get overwritten
	updatedSvc, err := ft.Facade.GetService(ft.CTX, "ParentServiceID")
	c.Assert(err, IsNil)
	c.Assert(updatedSvc.Description, Equals, svc.Description)
	c.Assert(int(updatedSvc.DesiredState), Equals, int(service.SVCRun))
}

func (ft *FacadeIntegrationTest) TestFacade_SnapshotAlwaysPauses(c *C) {
	// Test to make sure services are always paused during a snapshot

	// Set up mocks to handle starting services and waiting
	// We have to reset the zzk mocks to replace what is in SetUpTest
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	var mutex sync.RWMutex
	desiredStates := make(map[string]service.DesiredState)
	desiredStates["ParentServiceID"] = service.SVCStop
	desiredStates["childService1"] = service.SVCStop

	ft.dfs.On("Timeout").Return(10 * time.Second)
	ft.zzk.On("LockServices", ft.CTX, mock.AnythingOfType("[]service.ServiceDetails")).Return(nil)
	ft.zzk.On("UnlockServices", ft.CTX, mock.AnythingOfType("[]service.ServiceDetails")).Return(nil)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"),
		mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"),
		mock.AnythingOfType("bool")).Return(nil).Run(func(args mock.Arguments) {
		svcs := args.Get(2).([]*service.Service)
		mutex.Lock()
		defer mutex.Unlock()
		for _, s := range svcs {
			desiredStates[s.ID] = service.DesiredState(s.DesiredState)
		}
	})

	p1 := &pool.ResourcePool{
		ID:          "default",
		Permissions: pool.DFSAccess | pool.AdminAccess,
	}
	if err := ft.Facade.AddResourcePool(ft.CTX, p1); err != nil {
		c.Fatalf("Failed add pool %+v: %s", p1, err)
	}

	// add a service with 1 subservice
	svc := service.Service{
		ID:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCStop),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		StartLevel:     0,
	}
	childService1 := service.Service{
		ID:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
		StartLevel:      1,
	}

	var err error

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}

	waitBlocker := make(chan interface{})
	close(waitBlocker)
	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("service.DesiredState"),
		mock.AnythingOfType("<-chan interface {}")).Return(nil).Run(func(args mock.Arguments) {
		s := args.Get(0).(*service.Service)
		desiredState := args.Get(1).(service.DesiredState)
		cancel := args.Get(2).(<-chan interface{})
		for {
			select {
			case <-cancel:
				return
			default:
			}
			mutex.RLock()
			state, _ := desiredStates[s.ID]
			mutex.RUnlock()
			if int(state) == int(desiredState) || (int(desiredState) == int(service.SVCPause) && int(state) == int(service.SVCStop)) {
				// Sleep for 1 second and then return
				time.Sleep(time.Second)
				// Wait for the waitBlocker
				mutex.RLock()
				defer mutex.RUnlock()
				<-waitBlocker
				return
			} else {
				// wait 100 ms and try again
				time.Sleep(100 * time.Millisecond)
			}
		}
	})

	// On Snapshot, fail if all services aren't either paused or stopped
	ft.dfs.On("Snapshot", mock.AnythingOfType("dfs.SnapshotInfo")).Return("snapshotID", nil).Run(func(args mock.Arguments) {
		mutex.RLock()
		defer mutex.RUnlock()

		for _, state := range desiredStates {
			if state != service.SVCPause && state != service.SVCStop {
				c.Fatalf("Attempted to snapshot with running services")
			}
		}
	})

	// For this test, tenant mounts are valid.
	ft.dfs.On("VerifyTenantMounts", "ParentServiceID").Return(nil)

	// Start the parent service synchronously with AutoLaunch set to false, so that the child service stays stopped
	_, err = ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"ParentServiceID"}, AutoLaunch: false, Synchronous: true})
	c.Assert(err, IsNil)

	// Snapshot the service.
	_, err = ft.Facade.Snapshot(ft.CTX, "ParentServiceID", "", []string{}, 0)
	c.Assert(err, IsNil)

	// Make sure services returned to their original state
	err = ft.Facade.WaitService(ft.CTX, service.SVCRun, 5*time.Second, false, "ParentServiceID")
	c.Assert(err, IsNil)
	services, err := ft.Facade.GetServices(ft.CTX, dao.ServiceRequest{})
	c.Assert(err, IsNil)
	for _, s := range services {
		switch s.ID {
		case "ParentServiceID":
			c.Assert(int(s.DesiredState), Equals, int(service.SVCRun))
		default:
			c.Assert(int(s.DesiredState), Equals, int(service.SVCStop))
		}
	}

	// Stop the parent service
	_, err = ft.Facade.StopService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"ParentServiceID"}, AutoLaunch: false, Synchronous: true})
	c.Assert(err, IsNil)

	// TEST: Snapshot during service start
	// Block Wait
	mutex.Lock()
	waitBlocker = make(chan interface{})
	mutex.Unlock()
	// Start services in a goroutine
	startDone := make(chan interface{})
	go func() {
		defer close(startDone)
		_, err := ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"ParentServiceID"}, AutoLaunch: true, Synchronous: true})
		c.Assert(err, IsNil)
	}()

	time.Sleep(10 * time.Millisecond)

	// Call snapshot in a goroutine
	snapshotDone := make(chan interface{})
	go func() {
		defer close(snapshotDone)
		_, err := ft.Facade.Snapshot(ft.CTX, "ParentServiceID", "", []string{}, 0)
		c.Assert(err, IsNil)

		// Wait for services to return to running
		err = ft.Facade.WaitService(ft.CTX, service.SVCRun, 5*time.Second, false, "ParentServiceID", "childService1")
		c.Assert(err, IsNil)
	}()

	// Unblock wait
	close(waitBlocker)

	// Wait for the snapshot and start services to complete
	<-startDone
	<-snapshotDone

	// All services should be running
	services, err = ft.Facade.GetServices(ft.CTX, dao.ServiceRequest{})
	c.Assert(err, IsNil)
	for _, s := range services {
		c.Assert(int(s.DesiredState), Equals, int(service.SVCRun))
	}

	// TEST: Start services during snapshot

	//block wait and call snapshot in a goroutine
	mutex.Lock()
	waitBlocker = make(chan interface{})
	mutex.Unlock()
	snapshotDone = make(chan interface{})
	go func() {
		defer close(snapshotDone)
		_, err := ft.Facade.Snapshot(ft.CTX, "ParentServiceID", "", []string{}, 0)
		c.Assert(err, IsNil)

		// Wait for services to return to running
		err = ft.Facade.WaitService(ft.CTX, service.SVCRun, 5*time.Second, false, "ParentServiceID", "childService1")
		c.Assert(err, IsNil)
	}()

	time.Sleep(10 * time.Millisecond)
	// Start services in a goroutine
	startDone = make(chan interface{})
	go func() {
		defer close(startDone)
		_, err := ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"ParentServiceID"}, AutoLaunch: true, Synchronous: true})
		c.Assert(err, IsNil)
	}()

	//unblock wait
	close(waitBlocker)

	<-snapshotDone
	<-startDone

	// All services should still be running
	services, err = ft.Facade.GetServices(ft.CTX, dao.ServiceRequest{})
	c.Assert(err, IsNil)
	for _, s := range services {
		c.Assert(int(s.DesiredState), Equals, int(service.SVCRun))
	}

}

// If the dfs mounts are invalid. Do not start services.
func (ft *FacadeIntegrationTest) TestFacade_StartService_InvalidMounts(c *C) {
	svc := service.Service{
		ID: "TestFacade_StartService_PoolWithDFS",
		Name: "TestFacade_StartService_PoolWithDFS",
		PoolID: "default",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCStop),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		StartLevel:     0,
	}

	// add the resource pool (with DFS access so it checks mounts.)
	rp := pool.ResourcePool{ID: "default", Permissions: pool.DFSAccess}
	err := ft.Facade.AddResourcePool(ft.CTX, &rp)
	if err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed adding service testService: %+v, %s", svc, err)
	}

	ft.dfs.On("VerifyTenantMounts", "TestFacade_StartService_PoolWithDFS").Return(fmt.Errorf("some error"))

	err = ft.Facade.validateServiceStart(datastore.Get(), &svc)
	if err == nil {
		c.Error("Service should have failed tenant mount validation for starting...")
	}
}

func (ft *FacadeIntegrationTest) TestFacade_ClearEmergencyStopFlag(c *C) {
	// add a service with 2 subservices and set EmergencyShutdown to true for all 3
	svc := service.Service{
		ID:                "ParentServiceID",
		Name:              "ParentService",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         1,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: true,
	}
	childService1 := service.Service{
		ID:                "childService1",
		Name:              "childservice1",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:   "ParentServiceID",
		EmergencyShutdown: true,
	}
	childService2 := service.Service{
		ID:                "childService2",
		Name:              "childservice2",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID:   "ParentServiceID",
		EmergencyShutdown: true,
	}
	var err error
	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService2); err != nil {
		c.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}

	// Clear emergency stop on one child
	count, err := ft.Facade.ClearEmergencyStopFlag(ft.CTX, "childService1")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 1)

	// Make sure emergency stop is cleared for only that one service
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	c.Assert(err, IsNil)
	for _, s := range services {
		if s.ID == "childService1" {
			c.Assert(s.EmergencyShutdown, Equals, false)
		} else {
			c.Assert(s.EmergencyShutdown, Equals, true)
		}
	}

	// clear emergency stop on the parent service
	count, err = ft.Facade.ClearEmergencyStopFlag(ft.CTX, "ParentServiceID")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 2)

	// Make sure emergency stop is cleared on all services
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	c.Assert(err, IsNil)
	for _, s := range services {
		c.Assert(s.EmergencyShutdown, Equals, false)
	}

	// Clear emergency stop on a service that is already cleared
	count, err = ft.Facade.ClearEmergencyStopFlag(ft.CTX, "ParentServiceID")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 0)

	// Make sure emergency stop is still cleared on all services
	services, err = ft.Facade.GetServices(ft.CTX, serviceRequest)
	c.Assert(err, IsNil)
	for _, s := range services {
		c.Assert(s.EmergencyShutdown, Equals, false)
	}
}

func (ft *FacadeIntegrationTest) TestFacade_rollingRestart_Pass(c *C) {
	svc := service.Service{
		ID:                "serviceID",
		Name:              "Service",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         3,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
		HealthChecks:      map[string]health.HealthCheck{"healthcheck": health.HealthCheck{}},
	}

	hcache := health.New()
	ft.Facade.SetHealthCache(hcache)
	key0 := health.HealthStatusKey{
		ServiceID:       svc.ID,
		InstanceID:      0,
		HealthCheckName: "healthcheck",
	}
	key1 := health.HealthStatusKey{
		ServiceID:       svc.ID,
		InstanceID:      1,
		HealthCheckName: "healthcheck",
	}
	key2 := health.HealthStatusKey{
		ServiceID:       svc.ID,
		InstanceID:      2,
		HealthCheckName: "healthcheck",
	}

	statusOK := health.HealthStatus{
		Status:    health.OK,
		StartedAt: time.Now(),
		Duration:  time.Minute,
	}
	statusFailed := health.HealthStatus{
		Status:    health.Failed,
		StartedAt: time.Now(),
		Duration:  time.Minute,
	}

	hcache.Set(key0, statusFailed, time.Hour)
	hcache.Set(key1, statusFailed, time.Hour)
	hcache.Set(key2, statusFailed, time.Hour)

	restarted0 := make(chan struct{})
	restarted1 := make(chan struct{})
	restarted2 := make(chan struct{})

	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 0).Return(&zks.State{}, nil).Once()
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 1).Return(&zks.State{}, nil).Once()
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 2).Return(&zks.State{}, nil).Once()

	// Make sure we call RestartInstance once for each instance of svc
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 0).Return(nil).Once()
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 1).Return(nil).Once()
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 2).Return(nil).Once()

	// Make sure we call WaitInstance once for each instance of svc
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 0, mock.AnythingOfType("func(*service.State, bool) bool"),
		mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		close(restarted0)
	}).Return(nil).Once()
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 1, mock.AnythingOfType("func(*service.State, bool) bool"),
		mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		close(restarted1)
	}).Return(nil).Once()
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 2, mock.AnythingOfType("func(*service.State, bool) bool"),
		mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		close(restarted2)
	}).Return(nil).Once()

	done := make(chan struct{})
	go func() {
		err := ft.Facade.rollingRestart(ft.CTX, &svc, 30*time.Second, make(chan interface{}))
		c.Assert(err, IsNil)
		close(done)
	}()
	timer := time.NewTimer(5 * time.Second)
	// Should see instance 0 restarted first
	select {
	case <-restarted1:
		c.Fatalf("Instance 1 restarted before 0")
	case <-restarted2:
		c.Fatalf("Instance 2 restarted before 0")
	case <-timer.C:
		c.Fatalf("Timeout waiting for instance 0 to restart")
	case <-restarted0:
	}

	// Instance 1 won't restart until healthchecks pass for instance 0
	timer.Reset(2 * time.Second)
	select {
	case <-restarted1:
		c.Fatalf("Instance 1 restarted before 0 passed healthcheck")
	case <-restarted2:
		c.Fatalf("Instance 2 restarted before 0 passed healthcheck")
	case <-timer.C:
	}

	// Pass the healthchecks for instance 0
	hcache.Set(key0, statusOK, time.Hour)

	// Now we should see instance 1 restart
	timer.Reset(5 * time.Second)
	select {
	case <-restarted2:
		c.Fatalf("Instance 2 restarted before 1")
	case <-timer.C:
		c.Fatalf("Timeout waiting for instance 1 to restart")
	case <-restarted1:
	}

	// Instance 2 won't restart until healthchecks pass for instance 1
	timer.Reset(2 * time.Second)
	select {
	case <-restarted2:
		c.Fatalf("Instance 2 restarted before 1 passed healthcheck")
	case <-timer.C:
	}

	// Pass the healthchecks for instance 1
	hcache.Set(key1, statusOK, time.Hour)

	// Now we should see instance 2 restart
	timer.Reset(5 * time.Second)
	select {
	case <-timer.C:
		c.Fatalf("Timeout waiting for instance 1 to restart")
	case <-restarted2:
	}

	// Pass the healthchecks for instance 2
	hcache.Set(key2, statusOK, time.Hour)

	select {
	case <-timer.C:
		c.Fatalf("Timeout waiting for rolling restart")
	case <-done:
	}
}

func (ft *FacadeIntegrationTest) TestFacade_rollingRestart_TimeoutWait(c *C) {
	svc := service.Service{
		ID:                "serviceID",
		Name:              "Service",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         2,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
	}

	// Use a timeout of 1 second
	timeout := 1 * time.Second

	restarted1 := make(chan struct{})

	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 0).Return(&zks.State{}, nil).Once()
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 1).Return(&zks.State{}, nil).Once()

	// Make sure we call RestartInstance once for each instance of svc
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 0).Return(nil).Once()
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 1).Return(nil).Once()

	// Make sure we call WaitInstance once for each instance of svc
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 0, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		cancel := args[4].(<-chan struct{})
		// Wait twice the timeout or until cancelled to force a timeout
		select {
		case <-cancel:
		case <-time.After(2 * timeout):
			c.Fatalf("Wait not cancelled after timeout")
		}
	}).Return(nil).Once()

	ft.zzk.On("WaitInstance", ft.CTX, &svc, 1, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		close(restarted1)
	}).Return(nil).Once()

	done := make(chan struct{})
	go func() {
		err := ft.Facade.rollingRestart(ft.CTX, &svc, timeout, make(chan interface{}))
		c.Assert(err, IsNil)
		close(done)
	}()
	timer := time.NewTimer(3 * timeout)
	// instance 0 will timeout, but we should see instance 1 restart
	select {
	case <-restarted1:
	case <-timer.C:
		c.Fatalf("Timeout waiting for instance 1 to restart")
	}

	timer.Reset(3 * timeout)
	select {
	case <-timer.C:
		c.Fatalf("Timeout waiting for rolling restart")
	case <-done:
	}
}

func (ft *FacadeIntegrationTest) TestFacade_rollingRestart_TimeoutHealthcheck(c *C) {
	svc := service.Service{
		ID:                "serviceID",
		Name:              "Service",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         2,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
		HealthChecks:      map[string]health.HealthCheck{"healthcheck": health.HealthCheck{}},
	}

	hcache := health.New()
	ft.Facade.SetHealthCache(hcache)
	key0 := health.HealthStatusKey{
		ServiceID:       svc.ID,
		InstanceID:      0,
		HealthCheckName: "healthcheck",
	}
	key1 := health.HealthStatusKey{
		ServiceID:       svc.ID,
		InstanceID:      1,
		HealthCheckName: "healthcheck",
	}

	statusOK := health.HealthStatus{
		Status:    health.OK,
		StartedAt: time.Now(),
		Duration:  time.Minute,
	}
	statusFailed := health.HealthStatus{
		Status:    health.Failed,
		StartedAt: time.Now(),
		Duration:  time.Minute,
	}

	// set a 1 second timeout
	timeout := 1 * time.Second

	hcache.Set(key0, statusFailed, time.Hour)
	hcache.Set(key1, statusFailed, time.Hour)

	restarted0 := make(chan struct{})
	restarted1 := make(chan struct{})

	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 0).Return(&zks.State{}, nil).Once()
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 1).Return(&zks.State{}, nil).Once()

	// Make sure we call RestartInstance once for each instance of svc
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 0).Return(nil).Once()
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 1).Return(nil).Once()

	// Make sure we call WaitInstance once for each instance of svc
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 0, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		close(restarted0)
	}).Return(nil).Once()
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 1, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Run(func(args mock.Arguments) {
		close(restarted1)
	}).Return(nil).Once()

	done := make(chan struct{})
	go func() {
		err := ft.Facade.rollingRestart(ft.CTX, &svc, timeout, make(chan interface{}))
		c.Assert(err, IsNil)
		close(done)
	}()
	timer := time.NewTimer(5 * time.Second)
	// Should see instance 0 restarted first
	select {
	case <-restarted1:
		c.Fatalf("Instance 1 restarted before 0")
	case <-timer.C:
		c.Fatalf("Timeout waiting for instance 0 to restart")
	case <-restarted0:
	}

	// Leave Instance 0's healthchecks failing and make sure we move on
	// Now we should see instance 1 restart
	timer.Reset(3 * timeout)
	select {
	case <-timer.C:
		c.Fatalf("Timeout waiting for instance 1 to restart")
	case <-restarted1:
	}

	// Pass the healthchecks for instance 1
	hcache.Set(key1, statusOK, time.Hour)

	timer.Reset(3 * timeout)
	select {
	case <-timer.C:
		c.Fatalf("Timeout waiting for rolling restart")
	case <-done:
	}
}

func (ft *FacadeIntegrationTest) TestFacade_rollingRestart_FailWait(c *C) {
	svc := service.Service{
		ID:                "serviceID",
		Name:              "Service",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         3,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
	}

	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 0).Return(&zks.State{}, nil).Once()
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 1).Return(&zks.State{}, nil).Once()

	// Make sure we call RestartInstance once for each insance of svc that gets called
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 0).Return(nil).Once()
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 1).Return(nil).Once()

	// Make sure we call WaitInstance once for each instance of svc that gets called
	testerr := errors.New("test error")
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 0, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Return(nil).Once()
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 1, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Return(testerr).Once()

	err := ft.Facade.rollingRestart(ft.CTX, &svc, 30*time.Second, make(chan interface{}))
	// Make sure our rollingRestart bailed after it failed for one instance
	c.Assert(err, Equals, testerr)
}

func (ft *FacadeIntegrationTest) TestFacade_rollingRestart_FailRestartInstance(c *C) {
	svc := service.Service{
		ID:                "serviceID",
		Name:              "Service",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         3,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
	}

	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 0).Return(&zks.State{}, nil).Once()
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 1).Return(&zks.State{}, nil).Once()

	// Make sure we call RestartInstance once for each insance of svc that gets called
	testerr := errors.New("test error")
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 0).Return(nil).Once()
	ft.zzk.On("RestartInstance", ft.CTX, svc.PoolID, svc.ID, 1).Return(testerr).Once()

	// Make sure we call WaitInstance once for each insance of svc that gets called
	ft.zzk.On("WaitInstance", ft.CTX, &svc, 0, mock.AnythingOfType("func(*service.State, bool) bool"), mock.AnythingOfType("<-chan struct {}")).Return(nil).Once()
	err := ft.Facade.rollingRestart(ft.CTX, &svc, 30*time.Second, make(chan interface{}))
	// Make sure our rollingRestart bailed after it failed for one instance
	c.Assert(err, Equals, testerr)
}

func (ft *FacadeIntegrationTest) TestFacade_rollingRestart_FailGetState(c *C) {
	svc := service.Service{
		ID:                "serviceID",
		Name:              "Service",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         3,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
	}

	testerr := errors.New("test error")
	ft.zzk.On("GetServiceState", ft.CTX, svc.PoolID, svc.ID, 0).Return(nil, testerr).Once()

	err := ft.Facade.rollingRestart(ft.CTX, &svc, 30*time.Second, make(chan interface{}))
	// Make sure our rollingRestart bailed after it failed for one instance
	c.Assert(err, Equals, testerr)
}

func (ft *FacadeIntegrationTest) TestFacade_StartMultipleServices(c *C) {
	// create a service tree that looks like this:
	// ParentServiceID
	// ->childService1
	// ->childService2
	//   ->childService3
	//   ->childService4
	//   ->childService5 (MANUAL)
	//   ->childService6 (MANUAL)
	// ParentServiceID2

	svc := service.Service{
		ID:                "ParentServiceID",
		Name:              "ParentService",
		Startup:           "/usr/bin/ping -c localhost",
		Description:       "Ping a remote host a fixed number of times",
		Instances:         1,
		InstanceLimits:    domain.MinMax{1, 1, 1},
		ImageID:           "test/pinger",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		DesiredState:      int(service.SVCRun),
		Launch:            "auto",
		Endpoints:         []service.ServiceEndpoint{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmergencyShutdown: false,
	}
	childService1 := service.Service{
		ID:                "childService1",
		Name:              "childservice1",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:   "ParentServiceID",
		EmergencyShutdown: false,
	}
	childService2 := service.Service{
		ID:                "childService2",
		Name:              "childservice2",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID:   "ParentServiceID",
		EmergencyShutdown: false,
	}
	childService3 := service.Service{
		ID:                "childService3",
		Name:              "childservice3",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:   "childService2",
		EmergencyShutdown: false,
	}
	childService4 := service.Service{
		ID:                "childService4",
		Name:              "childservice4",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:   "childService2",
		EmergencyShutdown: false,
	}
	childService5 := service.Service{
		ID:                "childService5",
		Name:              "childservice5",
		Launch:            "manual",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:   "childService2",
		EmergencyShutdown: false,
	}
	childService6 := service.Service{
		ID:                "childService6",
		Name:              "childservice6",
		Launch:            "manual",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID:   "childService2",
		EmergencyShutdown: false,
	}
	svc2 := service.Service{
		ID:                "ParentServiceID2",
		Name:              "ParentService2",
		Launch:            "auto",
		PoolID:            "default",
		DeploymentID:      "deployment_id",
		Startup:           "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		EmergencyShutdown: false,
	}

	var err error

	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "default"}
	if err = ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}

	if err = ft.Facade.AddService(ft.CTX, svc); err != nil {
		c.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	if err = ft.Facade.AddService(ft.CTX, childService1); err != nil {
		c.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService2); err != nil {
		c.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService3); err != nil {
		c.Fatalf("Failed Loading Child Service 3: %+v, %s", childService3, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService4); err != nil {
		c.Fatalf("Failed Loading Child Service 4: %+v, %s", childService4, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService5); err != nil {
		c.Fatalf("Failed Loading Child Service 5: %+v, %s", childService5, err)
	}
	if err = ft.Facade.AddService(ft.CTX, childService6); err != nil {
		c.Fatalf("Failed Loading Child Service 6: %+v, %s", childService6, err)
	}
	if err = ft.Facade.AddService(ft.CTX, svc2); err != nil {
		c.Fatalf("Failed Loading Parent Service 2 Service: %+v, %s", svc2, err)
	}

	// Start services childService1, childService2, childService3, childService5, and ParentServiceID2
	//  Mock the service state manager and make sure the right services get passed with no duplicates
	//  Should be childService1, childService2, childService3, childService4, childService5 in one call and
	//  ParentServiceID2 in a second call with different tenants
	mockedSSM := &ssmmocks.ServiceStateManager{}
	ft.Facade.SetServiceStateManager(mockedSSM)

	mockedSSM.On("ScheduleServices", mock.AnythingOfType("[]*service.Service"),
		"ParentServiceID", service.SVCRun, false).Return(nil).Run(func(args mock.Arguments) {
		services := args.Get(0).([]*service.Service)
		c.Assert(len(services), Equals, 5)
		found := make(map[string]bool)
		for _, s := range services {
			found[s.ID] = true
		}

		c.Assert(found["childService1"], Equals, true)
		c.Assert(found["childService2"], Equals, true)
		c.Assert(found["childService3"], Equals, true)
		c.Assert(found["childService4"], Equals, true)
		c.Assert(found["childService5"], Equals, true)
	}).Once()

	mockedSSM.On("ScheduleServices", mock.AnythingOfType("[]*service.Service"),
		"ParentServiceID2", service.SVCRun, false).Return(nil).Run(func(args mock.Arguments) {
		services := args.Get(0).([]*service.Service)
		c.Assert(len(services), Equals, 1)
		c.Assert(services[0].ID, Equals, "ParentServiceID2")
	}).Once()

	mockedSSM.On("WaitScheduled", "ParentServiceID", mock.AnythingOfType("[]string")).Run(func(args mock.Arguments) {
		sIDs := args.Get(1).([]string)
		c.Assert(len(sIDs), Equals, 5)
		found := make(map[string]bool)
		for _, s := range sIDs {
			found[s] = true
		}
		c.Assert(found["childService1"], Equals, true)
		c.Assert(found["childService2"], Equals, true)
		c.Assert(found["childService3"], Equals, true)
		c.Assert(found["childService4"], Equals, true)
		c.Assert(found["childService5"], Equals, true)
	}).Once()

	mockedSSM.On("WaitScheduled", "ParentServiceID2", mock.AnythingOfType("[]string")).Run(func(args mock.Arguments) {
		sIDs := args.Get(1).([]string)
		c.Assert(len(sIDs), Equals, 1)
		c.Assert(sIDs[0], Equals, "ParentServiceID2")
	}).Once()

	count, err := ft.Facade.StartService(ft.CTX, dao.ScheduleServiceRequest{ServiceIDs: []string{"childService1", "childService2", "childService3", "childService5", "ParentServiceID2"}, AutoLaunch: true, Synchronous: true})
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 6)

	mockedSSM.AssertExpectations(c)
}

func (ft *FacadeIntegrationTest) setupMigrationTestWithoutEndpoints(t *C) error {
	return ft.setupMigrationTest(t, false)
}

func (ft *FacadeIntegrationTest) setupMigrationTestWithEndpoints(t *C) error {
	return ft.setupMigrationTest(t, true)
}

func (ft *FacadeIntegrationTest) setupMigrationTest(t *C, addEndpoint bool) error {
	tenant := service.Service{
		ID:           "original_service_id_tenant",
		Name:         "original_service_name_tenant",
		DeploymentID: "original_service_deployment_id",
		PoolID:       "original_service_pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}
	c0 := service.Service{
		ID:              "original_service_id_child_0",
		ParentServiceID: "original_service_id_tenant",
		Name:            "original_service_name_child_0",
		DeploymentID:    "original_service_deployment_id",
		PoolID:          "original_service_pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	}
	c1 := service.Service{
		ID:              "original_service_id_child_1",
		ParentServiceID: "original_service_id_tenant",
		Name:            "original_service_name_child_1",
		DeploymentID:    "original_service_deployment_id",
		PoolID:          "original_service_pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
	}

	if addEndpoint {
		c1.Endpoints = []service.ServiceEndpoint{
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name:        "original_service_endpoint_name_child_1",
					Application: "original_service_endpoint_application_child_1",
					Purpose:     "export",
				},
			),
		}
	}

	if err := ft.Facade.AddService(ft.CTX, tenant); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", tenant.ID, err)
		return err
	}
	if err := ft.Facade.AddService(ft.CTX, c0); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", c0.ID, err)
		return err
	}
	if err := ft.Facade.AddService(ft.CTX, c1); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", c1.ID, err)
		return err
	}

	return nil
}

func (ft *FacadeIntegrationTest) setupMigrationServices(t *C, originalConfigs map[string]servicedefinition.ConfigFile) (*service.Service, *service.Service, error) {
	svc := service.Service{
		ID:              "svc1",
		Name:            "TestFacade_migrateServiceConfigs_oldSvc",
		DeploymentID:    "deployment_id",
		PoolID:          "pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		OriginalConfigs: originalConfigs,
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Errorf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return nil, nil, err
	}

	oldSvc, err := ft.Facade.GetService(ft.CTX, svc.ID)
	if err != nil {
		t.Errorf("Setup failed; could not get svc %s: %s", oldSvc.ID, err)
		return nil, nil, err
	}

	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.Description = "migrated service"

	if originalConfigs != nil {
		newSvc.OriginalConfigs = make(map[string]servicedefinition.ConfigFile)
		newSvc.OriginalConfigs["unchangedConfig"] = oldSvc.OriginalConfigs["unchangedConfig"]
		newSvc.OriginalConfigs["addedConfig"] = servicedefinition.ConfigFile{Filename: "addedConfig", Content: "original version"}

		newSvc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
		for filename, conf := range newSvc.OriginalConfigs {
			newSvc.ConfigFiles[filename] = conf
		}
	}

	return oldSvc, &newSvc, nil
}

func (ft *FacadeIntegrationTest) getConfigFiles(svc *service.Service) ([]*serviceconfigfile.SvcConfigFile, error) {
	tenantID, servicePath, err := ft.Facade.getServicePath(ft.CTX, svc.ID)
	if err != nil {
		return nil, err
	}
	configStore := serviceconfigfile.NewStore()
	return configStore.GetConfigFiles(ft.CTX, tenantID, servicePath)
}

func getOriginalConfigs() map[string]servicedefinition.ConfigFile {
	originalConfigs := make(map[string]servicedefinition.ConfigFile)
	originalConfigs["unchangedConfig"] = servicedefinition.ConfigFile{Filename: "unchangedConfig", Content: "original version"}
	originalConfigs["deletedConfig"] = servicedefinition.ConfigFile{Filename: "deletedConfig", Content: "original version"}
	return originalConfigs
}

func (ft *FacadeIntegrationTest) createNewChildService(t *C) *service.Service {
	originalID := "original_service_id_child_0"
	oldSvc, err := ft.Facade.GetService(ft.CTX, originalID)
	t.Assert(err, IsNil)

	newSvc := service.Service{}
	newSvc = *oldSvc
	newSvc.ID = "new-clone-id"
	newSvc.Name = oldSvc.Name + "_CLONE"
	return &newSvc
}

func (ft *FacadeIntegrationTest) assertServiceAdded(t *C, newSvc *service.Service) {
	svcs, err := ft.Facade.GetServices(ft.CTX, dao.ServiceRequest{TenantID: newSvc.ParentServiceID})
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 4) // there should be 1 additional service
	foundAddedService := false
	for _, svc := range svcs {
		if svc.Name == newSvc.Name {
			foundAddedService = true
			t.Assert(svc.ID, Not(Equals), "new-clone-id") // the service ID should be changed
			break
		}
	}
	t.Assert(foundAddedService, Equals, true)
}

func (ft *FacadeIntegrationTest) createServiceDeploymentRequest(t *C) *dao.ServiceDeploymentRequest {
	deployRequest := dao.ServiceDeploymentRequest{
		ParentID: "original_service_id_child_0",

		// A minimally valid ServiceDefinition
		Service: servicedefinition.ServiceDefinition{
			Name:    "added-service-name",
			ImageID: "ubuntu:latest",
			Launch:  "auto",
		},
	}

	return &deployRequest
}

func (ft *FacadeIntegrationTest) assertPathResolvesToServices(c *C, path string, noprefix bool, services ...service.Service) {
	details, err := ft.Facade.ResolveServicePath(ft.CTX, path, noprefix)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, len(services))
	foundids := make([]string, len(details))
	for i, d := range details {
		foundids[i] = d.ID
	}
	sort.Strings(foundids)
	ids := make([]string, len(services))
	for i, d := range services {
		ids[i] = d.ID
	}
	sort.Strings(ids)
	c.Assert(foundids, DeepEquals, ids)
}

func (ft *FacadeIntegrationTest) TestFacade_getChanges(c *C) {
	cursvc := service.Service{
		ID:			"get-changes-service",
		Name:			"TestFacade_getChanges_Current",
		DeploymentID:		"deployment-id",
		PoolID:			"pool-id",
		Launch:			"auto",
	}
	c.Assert(ft.Facade.AddService(ft.CTX, cursvc), IsNil)
	svc , _ := ft.Facade.getService(ft.CTX, cursvc.ID)
	svc.Name = "TestFacade_getChanges_Updated"
	updates := ft.Facade.getChanges(ft.CTX, svc)
	expected := "Name:TestFacade_getChanges_Updated;"
	c.Assert(expected, Equals, updates)
}
