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

func (ft *FacadeTest) TestFacade_validateService_badServiceID(t *C) {
	err := ft.Facade.validateService(ft.CTX, "badID", true)
	t.Assert(err, ErrorMatches, "No such entity {kind:service, id:badID}")
}

func (ft *FacadeTest) TestFacade_validateService_validateServiceForStartingFailed(t *C) {
	//Can't test this without mocking store

	// svc, err := ft.setupServiceWithEndpoints(t)
	// t.Assert(err, IsNil)

	// expectedErr := fmt.Sprintf("service %s is missing an address assignment", svc.ID)
	// t.Assert(err, ErrorMatches, expectedErr)
	return
}

func (ft *FacadeTest) TestFacade_validateService_VHostFail(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("test_vhost_1-0"), svc.ID).Return(ErrTestEPValidationFail)

	err = ft.Facade.validateService(ft.CTX, svc.ID, true)
	t.Assert(err, ErrorMatches, ErrTestEPValidationFail.Error())
}

func (ft *FacadeTest) TestFacade_validateService_PortFail(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("test_vhost_1-0"), svc.ID).Return(nil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("1234-1"), svc.ID).Return(ErrTestEPValidationFail)

	err = ft.Facade.validateService(ft.CTX, svc.ID, true)
	t.Assert(err, ErrorMatches, ErrTestEPValidationFail.Error())
}

func (ft *FacadeTest) TestFacade_validateService_Success(t *C) {
	svc, err := ft.setupServiceWithEndpoints(t)
	t.Assert(err, IsNil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("test_vhost_1-0"), svc.ID).Return(nil)
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey("1234-1"), svc.ID).Return(nil)

	err = ft.Facade.validateService(ft.CTX, svc.ID, true)
	t.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_noDupsInOneService(t *C) {
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

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_noDupsInAllServices(t *C) {
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

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_dupsInOneService(t *C) {
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

func (ft *FacadeTest) TestFacade_validateServiceEndpoints_dupsBtwnServices(t *C) {
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

func (ft *FacadeTest) TestFacade_migrateServiceConfigs_noConfigs(t *C) {
	oldSvc, newSvc, err := ft.setupMigrationServices(t, nil)
	t.Assert(err, IsNil)

	err = ft.Facade.migrateServiceConfigs(ft.CTX, oldSvc, newSvc)
	t.Assert(err, IsNil)
}

func (ft *FacadeTest) TestFacade_migrateServiceConfigs_noChanges(t *C) {
	oldSvc, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)

	err = ft.Facade.migrateServiceConfigs(ft.CTX, oldSvc, newSvc)
	t.Assert(err, IsNil)
}

// Verify migration of configuration data when the user has not changed any config files
func (ft *FacadeTest) TestFacade_migrateService_withoutUserConfigChanges(t *C) {
	_, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)

	err = ft.Facade.migrateService(ft.CTX, newSvc)
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

// Verify migration of configuration data when the user has changed the config files
func (ft *FacadeTest) TestFacade_migrateService_withUserConfigChanges(t *C) {
	oldSvc, newSvc, err := ft.setupMigrationServices(t, getOriginalConfigs())
	t.Assert(err, IsNil)

	err = ft.setupConfigCustomizations(oldSvc)
	newSvc.DatabaseVersion = oldSvc.DatabaseVersion
	t.Assert(err, IsNil)

	err = ft.Facade.migrateService(ft.CTX, newSvc)
	t.Assert(err, IsNil)

	result, err := ft.Facade.GetService(ft.CTX, newSvc.ID)
	t.Assert(err, IsNil)

	expectedConfigFiles := make(map[string]servicedefinition.ConfigFile)
	expectedConfigFiles["unchangedConfig"] = oldSvc.ConfigFiles["unchangedConfig"]
	expectedConfigFiles["addedConfig"] = newSvc.OriginalConfigs["addedConfig"]
	t.Assert(result.Description, Equals, newSvc.Description)
	t.Assert(result.OriginalConfigs, DeepEquals, newSvc.OriginalConfigs)
	t.Assert(result.ConfigFiles, DeepEquals, expectedConfigFiles)

	confs, err := ft.getConfigFiles(result)
	t.Assert(err, IsNil)
	t.Assert(len(confs), Equals, 1)
	for _, conf := range confs {
		t.Assert(conf.ConfFile.Filename, Not(Equals), "addedConfig")
		t.Assert(expectedConfigFiles[conf.ConfFile.Filename], Equals, conf.ConfFile)
	}
}

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_UndefinedService(t *C) {
	endpointMap, err := ft.Facade.GetServiceEndpoints(ft.CTX, "undefined", true, true, true)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "Could not find service undefined.*")
	t.Assert(endpointMap, IsNil)
}

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_ZKUnavailable(t *C) {
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

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_ServiceNotRunning(t *C) {
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

func (ft *FacadeTest) TestFacade_GetServiceEndpoints_ServiceRunning(t *C) {
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

func (ft *FacadeTest) setupServiceWithEndpoints(t *C) (*service.Service, error) {
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
					PortList:  []servicedefinition.Port{servicedefinition.Port{PortAddr: 1234, Enabled: true}},
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

func (ft *FacadeTest) setupMigrationServices(t *C, originalConfigs map[string]servicedefinition.ConfigFile) (*service.Service, *service.Service, error) {
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

func (ft *FacadeTest) setupConfigCustomizations(svc *service.Service) error {
	for filename, conf := range svc.OriginalConfigs {
		customizedConf := conf
		customizedConf.Content = "some user customized content"
		svc.ConfigFiles[filename] = customizedConf
	}

	err := ft.Facade.updateService(ft.CTX, svc)
	if err != nil {
		return err
	}

	result, err := ft.Facade.GetService(ft.CTX, svc.ID)
	*svc = *result

	return err
}

func (ft *FacadeTest) getConfigFiles(svc *service.Service) ([]*serviceconfigfile.SvcConfigFile, error) {
	tenantID, servicePath, err := ft.Facade.getTenantIDAndPath(ft.CTX, *svc)
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
