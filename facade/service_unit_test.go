// Copyright 2016 The Serviced Authors.
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

// +build unit

package facade_test

import (
	"fmt"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_GetTenantIDForRootApp(c *C) {
	serviceID := getRandomServiceID(c)
	expectedService := service.ServiceDetails{ID: serviceID}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, serviceID).Return(&expectedService, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	c.Assert(err, IsNil)
	c.Assert(result, Equals, serviceID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForRootAppFailsForNoSuchEntity(c *C) {
	serviceID := getRandomServiceID(c)
	expectedError := fmt.Errorf("mock DB error")
	ft.serviceStore.On("GetServiceDetails", ft.ctx, serviceID).Return(nil, expectedError)

	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	c.Assert(len(result), Equals, 0)
	c.Assert(err, Not(IsNil))
	c.Assert(err.Error(), Equals, expectedError.Error())
}

func (ft *FacadeUnitTest) Test_GetTenantIDForRootAppFailsForOtherDBError(c *C) {
	serviceID := getRandomServiceID(c)
	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("GetServiceDetails", ft.ctx, serviceID).Return(nil, expectedError)

	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	c.Assert(len(result), Equals, 0)
	c.Assert(err, Equals, expectedError)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForRootAppUsesCache(c *C) {
	serviceID := getRandomServiceID(c)
	expectedService := service.ServiceDetails{ID: serviceID}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, serviceID).Return(&expectedService, nil)

	// Do the first lookup to seed the internal cache
	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	// verify the first lookup worked
	c.Assert(err, IsNil)
	c.Assert(result, Equals, serviceID)

	// Change the mock to force an error if the DB is called.
	// If the cache is working, then this mock should never be invoked
	ft.serviceStore.On("GetServiceDetails", ft.ctx, serviceID).Return(nil, datastore.ErrEmptyKind)

	// Do the second lookup to hit the internal cache, but not call the mock
	result, err = ft.Facade.GetTenantID(ft.ctx, serviceID)

	// verify the second lookup worked just like the first
	c.Assert(err, IsNil)
	c.Assert(result, Equals, serviceID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForChildApp(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	parent := service.ServiceDetails{ID: parentID}
	child := service.ServiceDetails{ID: childID, ParentServiceID: parentID}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, childID).Return(&child, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, childID)

	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForGrandchildApp(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	grandchildID := getRandomServiceID(c)
	parent := service.ServiceDetails{ID: parentID}
	child := service.ServiceDetails{ID: childID, ParentServiceID: parentID}
	grandchild := service.ServiceDetails{ID: grandchildID, ParentServiceID: childID}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, childID).Return(&child, nil)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, grandchildID).Return(&grandchild, nil)

	// Do the first lookup to seed the internal cache
	result, err := ft.Facade.GetTenantID(ft.ctx, grandchildID)

	// verify the first lookup worked
	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)

	// Change the mock to force an error if the DB is called.
	// If the cache is working, then this mock should never be invoked
	ft.serviceStore.On("GetServiceDetails", ft.ctx, parentID).Return(nil, datastore.ErrEmptyKind)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, childID).Return(nil, datastore.ErrEmptyKind)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, grandchildID).Return(nil, datastore.ErrEmptyKind)

	// Add a new grandchild that's not in the cache, but shares a parent that should be in the cache.
	grandchildID2 := getRandomServiceID(c)
	grandchild2 := service.ServiceDetails{ID: grandchildID2, ParentServiceID: childID}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, grandchildID2).Return(&grandchild2, nil)

	// Do the second lookup to hit the internal cache, but not call the mock
	result, err = ft.Facade.GetTenantID(ft.ctx, grandchildID2)

	// verify the second lookup worked just like the first
	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForGrandchildAppUsesCache(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	grandchildID := getRandomServiceID(c)
	parent := service.ServiceDetails{ID: parentID}
	child := service.ServiceDetails{ID: childID, ParentServiceID: parentID}
	grandchild := service.ServiceDetails{ID: grandchildID, ParentServiceID: childID}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, childID).Return(&child, nil)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, grandchildID).Return(&grandchild, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, grandchildID)

	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForIntermediateParentFails(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	grandchildID := getRandomServiceID(c)
	parent := service.ServiceDetails{ID: parentID}
	grandchild := service.ServiceDetails{ID: grandchildID, ParentServiceID: childID}
	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("GetServiceDetails", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, childID).Return(nil, expectedError)
	ft.serviceStore.On("GetServiceDetails", ft.ctx, grandchildID).Return(&grandchild, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, grandchildID)

	c.Assert(len(result), Equals, 0)
	c.Assert(err, Not(IsNil))
	c.Assert(err, Equals, expectedError)
}

// Test a simple case of GetEvaluatedService where there is no parent service (i.e. no recursion).
//
// Note that the actual domain/service/evaluate_test.go has tests to exercise all of the various aspects of
// the templating evaluation, so all we need to verify here is that (A) some evaluation occurs, and
// (B) the instance ID is evaluated as expected.
func (ft *FacadeUnitTest) Test_GetEvaluatedServiceSimple(c *C) {
	serviceID := "0"
	serviceName := "service0"
	svc := service.Service{
		ID:      serviceID,
		Name:    serviceName,
		Actions: map[string]string{"name": "{{.Name}}", "instanceID": "{{.InstanceID}}"},
	}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, serviceID).Return(&service.ServiceDetails{ID: serviceID}, nil)
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(&svc, nil)
	ft.configfileStore.On("GetConfigFiles", ft.ctx, serviceID, "/"+serviceID).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	instanceID := 99
	result, err := ft.Facade.GetEvaluatedService(ft.ctx, serviceID, instanceID)

	c.Assert(result, Not(IsNil))
	c.Assert(err, IsNil)

	c.Assert(result.Actions["name"], Equals, serviceName)
	c.Assert(result.Actions["instanceID"], Equals, fmt.Sprintf("%d", instanceID))
}

// Test that the 'getService' function defined by facade.evaluateService() works properly on success
func (ft *FacadeUnitTest) Test_GetEvaluatedServiceUsesParent(c *C) {
	parentID := "parentServiceID"
	parentName := "parentServiceName"
	parentSvc := service.Service{
		ID:   parentID,
		Name: parentName,
	}
	childID := "childServiceID"
	childName := "childServiceName"
	childSvc := service.Service{
		ID:              childID,
		Name:            childName,
		ParentServiceID: parentID,
		Actions:         map[string]string{"parent": "{{(parent .).ID}}", "instanceID": "{{.InstanceID}}"},
	}
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parentSvc, nil)
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, "/"+parentID).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	ft.serviceStore.On("Get", ft.ctx, childID).Return(&childSvc, nil)
	childServicePath := "/" + parentID + "/" + childID
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, childServicePath).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	instanceID := 99
	result, err := ft.Facade.GetEvaluatedService(ft.ctx, childID, instanceID)

	c.Assert(result, Not(IsNil))
	c.Assert(err, IsNil)
	c.Assert(result.Actions["parent"], Equals, parentID)
	c.Assert(result.Actions["instanceID"], Equals, fmt.Sprintf("%d", instanceID))
}

// Test that the 'getServiceChild' function defined by facade.evaluateService() works properly on success
func (ft *FacadeUnitTest) Test_GetEvaluatedServiceUsesChild(c *C) {
	parentID := "parentServiceID"
	parentName := "parentServiceName"
	deploymentID := "testDeployment"
	parentSvc := service.Service{
		ID:           parentID,
		Name:         parentName,
		DeploymentID: deploymentID,
		Actions:      map[string]string{"child": "{{(child . \"childServiceName\").Title}}", "instanceID": "{{.InstanceID}}"},
	}
	childID := "childServiceID"
	childName := "childServiceName"
	childTitle := "Child Title"
	childSvc := service.Service{
		ID:              childID,
		Name:            childName,
		ParentServiceID: parentID,
		Title:           childTitle,
	}
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parentSvc, nil)
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, "/"+parentID).Return([]*serviceconfigfile.SvcConfigFile{}, nil)
	ft.serviceStore.On("FindChildService", ft.ctx, deploymentID, parentID, childName).Return(&childSvc, nil)

	ft.serviceStore.On("Get", ft.ctx, childID).Return(&childSvc, nil)
	childServicePath := "/" + parentID + "/" + childID
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, childServicePath).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	instanceID := 99
	result, err := ft.Facade.GetEvaluatedService(ft.ctx, parentID, instanceID)

	c.Assert(result, Not(IsNil))
	c.Assert(err, IsNil)
	c.Assert(result.Actions["child"], Equals, childTitle)
	c.Assert(result.Actions["instanceID"], Equals, fmt.Sprintf("%d", instanceID))
}

func (ft *FacadeUnitTest) Test_GetEvaluatedServiceFails(c *C) {
	serviceID := "someService"
	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(nil, expectedError)

	unused := 0
	result, err := ft.Facade.GetEvaluatedService(ft.ctx, serviceID, unused)

	c.Assert(result, IsNil)
	c.Assert(err, Not(IsNil))
	c.Assert(err, Equals, expectedError)
}

func (ft *FacadeUnitTest) Test_CheckRemoveRegistryImage_RemovesImage(c *C) {
	imageID := "imageID"

	// Expectations: if we get 0 from GetServiceCountByImage..
	ft.serviceStore.On("GetServiceCountByImage", ft.ctx, imageID).Return(0, nil)
	// .. then we should call Delete on the registryStore.
	ft.registryStore.On("Delete", ft.ctx, imageID).Return(nil)

	err := ft.Facade.CheckRemoveRegistryImage(ft.ctx, imageID)

	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_CheckRemoveRegistryImage_KeepsImage(c *C) {
	imageID := "imageID"

	// Expectations: if we get 1 from GetServiceCountByImage..
	// .. then it should exit without calling Delete on the registry store.
	ft.serviceStore.On("GetServiceCountByImage", ft.ctx, imageID).Return(1, nil)

	err := ft.Facade.CheckRemoveRegistryImage(ft.ctx, imageID)

	c.Assert(err, IsNil)
}

// Test that the 'getService' function defined by facade.evaluateService() works properly on failure
func (ft *FacadeUnitTest) Test_GetEvaluatedServiceGetParentFails(c *C) {
	parentID := "parentServiceID"
	childID := "childServiceID"
	childName := "childServiceName"
	childSvc := service.Service{
		ID:              childID,
		Name:            childName,
		ParentServiceID: parentID,
		Actions:         map[string]string{"parent": "{{(parent .).ID}}", "instanceID": "{{.InstanceID}}"},
	}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, childID).Return(&service.ServiceDetails{ID: childID, ParentServiceID: parentID}, nil)
	ft.serviceStore.On("Get", ft.ctx, childID).Return(&childSvc, nil)
	childServicePath := "/" + parentID + "/" + childID
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, childServicePath).Return([]*serviceconfigfile.SvcConfigFile{}, nil).Twice()

	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(nil, expectedError)
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, "/"+parentID).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	unused := 0
	result, err := ft.Facade.GetEvaluatedService(ft.ctx, childID, unused)

	c.Assert(result, IsNil)
	c.Assert(err, Not(IsNil))
	c.Assert(strings.Contains(err.Error(), expectedError.Error()), Equals, true)
}

// Test that the 'getServiceChild' function defined by facade.evaluateService() works properly on failure
func (ft *FacadeUnitTest) Test_GetEvaluatedServiceGetChildFails(c *C) {
	parentID := "parentServiceID"
	parentName := "parentServiceName"
	deploymentID := "testDeployment"
	parentSvc := service.Service{
		ID:           parentID,
		Name:         parentName,
		DeploymentID: deploymentID,
		Actions:      map[string]string{"child": "{{(child . \"childServiceName\").Title}}", "instanceID": "{{.InstanceID}}"},
	}
	childID := "childServiceID"
	childName := "childServiceName"
	childTitle := "Child Title"
	childSvc := service.Service{
		ID:              childID,
		Name:            childName,
		ParentServiceID: parentID,
		Title:           childTitle,
	}
	ft.serviceStore.On("GetServiceDetails", ft.ctx, parentID).Return(&service.ServiceDetails{ID: parentID}, nil)
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parentSvc, nil)
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, "/"+parentID).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("FindChildService", ft.ctx, deploymentID, parentID, childName).Return(nil, expectedError)

	ft.serviceStore.On("Get", ft.ctx, childID).Return(&childSvc, nil)
	childServicePath := "/" + parentID + "/" + childID
	ft.configfileStore.On("GetConfigFiles", ft.ctx, parentID, childServicePath).Return([]*serviceconfigfile.SvcConfigFile{}, nil)

	unused := 0
	result, err := ft.Facade.GetEvaluatedService(ft.ctx, parentID, unused)

	c.Assert(result, IsNil)
	c.Assert(err, Not(IsNil))
	c.Assert(strings.Contains(err.Error(), expectedError.Error()), Equals, true)
}

// Since the facade is optimized to cache serviceIDs, we need to generate a unique serviceID for each test
func getRandomServiceID(c *C) string {
	serviceID, err := utils.NewUUID()
	if err != nil {
		c.Fatalf("Failed to generate random service ID: %s", err)
	}
	return serviceID
}

//service store returned not-found
//service store returned other error
//service store return err=nil and svc=nil
