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

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_GetTenantIDForRootApp(c *C) {
	serviceID := getRandomServiceID(c)
	expectedService := service.Service{ID: serviceID}
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(&expectedService, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	c.Assert(err, IsNil)
	c.Assert(result, Equals, serviceID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForRootAppFailsForNoSuchEntity(c *C) {
	serviceID := getRandomServiceID(c)
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(nil, datastore.ErrNoSuchEntity{})

	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	c.Assert(len(result), Equals, 0)
	c.Assert(err, Not(IsNil))
	c.Assert(err, Equals, datastore.ErrNoSuchEntity{})
}

func (ft *FacadeUnitTest) Test_GetTenantIDForRootAppFailsForOtherDBError(c *C) {
	serviceID := getRandomServiceID(c)
	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(nil, expectedError)

	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	c.Assert(len(result), Equals, 0)
	c.Assert(err, Equals, expectedError)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForRootAppUsesCache(c *C) {
	serviceID := getRandomServiceID(c)
	expectedService := service.Service{ID: serviceID}
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(&expectedService, nil)

	// Do the first lookup to seed the internal cache
	result, err := ft.Facade.GetTenantID(ft.ctx, serviceID)

	// verify the first lookup worked
	c.Assert(err, IsNil)
	c.Assert(result, Equals, serviceID)

	// Change the mock to force an error if the DB is called.
	// If the cache is working, then this mock should never be invoked
	ft.serviceStore.On("Get", ft.ctx, serviceID).Return(nil, datastore.ErrEmptyKind)

	// Do the second lookup to hit the internal cache, but not call the mock
	result, err = ft.Facade.GetTenantID(ft.ctx, serviceID)

	// verify the second lookup worked just like the first
	c.Assert(err, IsNil)
	c.Assert(result, Equals, serviceID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForChildApp(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	parent := service.Service{ID: parentID}
	child := service.Service{ID: childID, ParentServiceID: parentID}
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("Get", ft.ctx, childID).Return(&child, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, childID)

	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForGrandchildApp(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	grandchildID := getRandomServiceID(c)
	parent := service.Service{ID: parentID}
	child := service.Service{ID: childID, ParentServiceID: parentID}
	grandchild := service.Service{ID: grandchildID, ParentServiceID: childID}
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("Get", ft.ctx, childID).Return(&child, nil)
	ft.serviceStore.On("Get", ft.ctx, grandchildID).Return(&grandchild, nil)

	// Do the first lookup to seed the internal cache
	result, err := ft.Facade.GetTenantID(ft.ctx, grandchildID)

	// verify the first lookup worked
	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)

	// Change the mock to force an error if the DB is called.
	// If the cache is working, then this mock should never be invoked
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(nil, datastore.ErrEmptyKind)
	ft.serviceStore.On("Get", ft.ctx, childID).Return(nil, datastore.ErrEmptyKind)
	ft.serviceStore.On("Get", ft.ctx, grandchildID).Return(nil, datastore.ErrEmptyKind)

	// Add a new grandchild that's not in the cache, but shares a parent that should be in the cache.
	grandchildID2 := getRandomServiceID(c)
	grandchild2 := service.Service{ID: grandchildID2, ParentServiceID: childID}
	ft.serviceStore.On("Get", ft.ctx, grandchildID2).Return(&grandchild2, nil)

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
	parent := service.Service{ID: parentID}
	child := service.Service{ID: childID, ParentServiceID: parentID}
	grandchild := service.Service{ID: grandchildID, ParentServiceID: childID}
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("Get", ft.ctx, childID).Return(&child, nil)
	ft.serviceStore.On("Get", ft.ctx, grandchildID).Return(&grandchild, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, grandchildID)

	c.Assert(err, IsNil)
	c.Assert(result, Equals, parentID)
}

func (ft *FacadeUnitTest) Test_GetTenantIDForIntermediateParentFails(c *C) {
	parentID := getRandomServiceID(c)
	childID := getRandomServiceID(c)
	grandchildID := getRandomServiceID(c)
	parent := service.Service{ID: parentID}
	grandchild := service.Service{ID: grandchildID, ParentServiceID: childID}
	expectedError := fmt.Errorf("expected error: oops")
	ft.serviceStore.On("Get", ft.ctx, parentID).Return(&parent, nil)
	ft.serviceStore.On("Get", ft.ctx, childID).Return(nil, expectedError)
	ft.serviceStore.On("Get", ft.ctx, grandchildID).Return(&grandchild, nil)

	result, err := ft.Facade.GetTenantID(ft.ctx, grandchildID)

	c.Assert(len(result), Equals, 0)
	c.Assert(err, Not(IsNil))
	c.Assert(err, Equals, expectedError)
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
