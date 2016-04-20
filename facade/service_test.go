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
	"strings"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/zzk/registry"

	"errors"
	"fmt"

	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
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
	c.Assert(ft.Facade.AddService(ft.CTX, svc), IsNil)
	return &svc
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

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStart_checkVHostFail(c *C) {
	// set up the endpoint with an invalid vhost
	endpoint := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
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
	svc := ft.setup_validateServiceStart(c, endpoint)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("vh1-0"), svc.ID).Return(ErrTestEPValidationFail)
	err := ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrTestEPValidationFail)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceStart_checkPortFail(c *C) {
	// set up the endpoint with in invalid public port
	endpoint := service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
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
	svc := ft.setup_validateServiceStart(c, endpoint)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey(":1234-1"), svc.ID).Return(ErrTestEPValidationFail)
	err := ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, Equals, ErrTestEPValidationFail)
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
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("vh1-0"), svc.ID).Return(nil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey(":1234-1"), svc.ID).Return(nil)
	err = ft.Facade.validateServiceStart(ft.CTX, svc)
	c.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_validateService_badServiceID(t *C) {
	_, err := ft.Facade.validateServiceUpdate(ft.CTX, &service.Service{ID: "badID"})
	t.Assert(err, ErrorMatches, "No such entity {kind:service, id:badID}")
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceEndpoints_noDupsInOneService(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceEndpoints_noDupsInAllServices(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return
	}

	childSvc := service.Service{
		ID:              "svc2",
		ParentServiceID: svc.ID,
		Name:            "TestFacade_validateServiceEndpoints_child",
		DeploymentID:    "deployment_id",
		PoolID:          "pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_3", Application: "test_ep_3", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_4", Application: "test_ep_4", Purpose: "export"}),
		},
	}
	if err := ft.Facade.AddService(ft.CTX, childSvc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", childSvc.ID, err)
		return
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Assert(err, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceEndpoints_dupsInOneService(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		},
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)
}

func (ft *FacadeIntegrationTest) TestFacade_validateServiceEndpoints_dupsBtwnServices(t *C) {
	svc := service.Service{
		ID:           "svc1",
		Name:         "TestFacade_validateServiceEndpoints",
		DeploymentID: "deployment_id",
		PoolID:       "pool_id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}

	if err := ft.Facade.AddService(ft.CTX, svc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", svc.ID, err)
		return
	}

	childSvc := service.Service{
		ID:              "svc2",
		ParentServiceID: svc.ID,
		Name:            "TestFacade_validateServiceEndpoints_child",
		DeploymentID:    "deployment_id",
		PoolID:          "pool_id",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		},
	}
	if err := ft.Facade.AddService(ft.CTX, childSvc); err != nil {
		t.Fatalf("Setup failed; could not add svc %s: %s", childSvc.ID, err)
		return
	}

	err := ft.Facade.validateServiceEndpoints(ft.CTX, &svc)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)
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
	serviceIDs := []string{svc.ID}
	errorStub := fmt.Errorf("Stub for cannot-connect-to-zookeeper")
	ft.zzk.On("GetServiceStates", svc.PoolID, mock.AnythingOfType("*[]servicestate.ServiceState"), serviceIDs).Return(errorStub)

	endpointMap, err := ft.Facade.GetServiceEndpoints(ft.CTX, svc.ID, true, true, true)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "Could not get service states for service .*")
	t.Assert(endpointMap, IsNil)
}

func (ft *FacadeIntegrationTest) TestFacade_GetServiceEndpoints_ServiceNotRunning(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	serviceIDs := []string{svc.ID}
	ft.zzk.On("GetServiceStates", svc.PoolID, mock.AnythingOfType("*[]servicestate.ServiceState"), serviceIDs).Return(nil)

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
	serviceIDs := []string{svc.ID}
	ft.zzk.On("GetServiceStates", svc.PoolID, mock.AnythingOfType("*[]servicestate.ServiceState"), serviceIDs).
		Return(nil).Run(func(args mock.Arguments) {
		// Mock results for 2 running instances
		statesArg := args.Get(1).(*[]servicestate.ServiceState)
		*statesArg = []servicestate.ServiceState{
			{ServiceID: svc.ID, InstanceID: 0, Endpoints: svc.Endpoints},
			{ServiceID: svc.ID, InstanceID: 1, Endpoints: svc.Endpoints},
		}
		t.Assert(true, Equals, true)
	})
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

	newSvc := service.Service{}

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
	err := ft.setupMigrationTestWithoutEndpoints(t)
	t.Assert(err, IsNil)

	oldSvc, err := ft.Facade.GetService(ft.CTX, "original_service_id_child_0")
	t.Assert(err, IsNil)

	newSvc1 := service.Service{}
	newSvc1 = *oldSvc
	newSvc1.Name = "ModifiedName1"
	newSvc1.Description = "migrated_service"

	oldSvc, err = ft.Facade.GetService(ft.CTX, "original_service_id_child_1")
	t.Assert(err, IsNil)

	newSvc2 := service.Service{}
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
		Deploy: []*dao.ServiceDeploymentRequest{deployRequest},
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
	t.Assert(len(svcs), Equals, 4)				// there should be 1 additional service
	foundAddedService := false
	for _, svc := range(svcs) {
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
		Deploy: []*dao.ServiceDeploymentRequest{deployRequest},
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
		Deploy: []*dao.ServiceDeploymentRequest{deployRequest1, deployRequest2},
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
		Deploy: []*dao.ServiceDeploymentRequest{deployRequest},
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
		Deploy: []*dao.ServiceDeploymentRequest{deployRequest},
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

func (ft *FacadeIntegrationTest) setupMigrationTestWithoutEndpoints(t *C) error {
	return ft.setupMigrationTest(t, false)
}

func (ft *FacadeIntegrationTest) setupMigrationTestWithEndpoints(t *C) error{
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
	t.Assert(len(svcs), Equals, 4)				// there should be 1 additional service
	foundAddedService := false
	for _, svc := range(svcs) {
		if svc.Name == newSvc.Name {
			foundAddedService = true
			t.Assert(svc.ID, Not(Equals), "new-clone-id")	// the service ID should be changed
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
			Name: "added-service-name",
			ImageID: "ubuntu:latest",
			Launch: "auto",
		},
	}

	return &deployRequest
}

