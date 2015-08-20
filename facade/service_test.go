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

	. "gopkg.in/check.v1"
)

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
