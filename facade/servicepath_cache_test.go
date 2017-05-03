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

package facade

import (
	"fmt"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	servicemocks "github.com/control-center/serviced/domain/service/mocks"
	. "gopkg.in/check.v1"
)

var _ = Suite(&ServicePathCacheTest{})

type ServicePathCacheTest struct {
	cache        *serviceCache
	serviceStore *servicemocks.Store
	unusedCTX    datastore.Context
}

func (t *ServicePathCacheTest) SetUpTest(c *C) {
	t.cache = NewServiceCache()
	t.serviceStore = &servicemocks.Store{}
}

func (t *ServicePathCacheTest) TearDownTest(c *C) {
	t.cache = nil
	t.serviceStore = nil
}

func (t *ServicePathCacheTest) Test_GetTenantID_SeedsCacheForTenantService(c *C) {
	tenantID := "mockTenantId"
	expectedService := service.ServiceDetails{ID: tenantID, Name: tenantID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, tenantID).Return(&expectedService, nil)

	result, err := t.cache.GetTenantID(tenantID, t.getService)

	c.Assert(result, Equals, tenantID)
	c.Assert(err, IsNil)

	// Verify the cache contents
	expectedCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
	}

	t.assertExpectedCacheEntries(c, expectedCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetTenantID_SeedsCacheForNestedServices(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID := "mockGrandChildId"

	tenant := service.ServiceDetails{ID: tenantID, Name: tenantID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, tenantID).Return(&tenant, nil)

	child := service.ServiceDetails{ID: childID, Name: childID, ParentServiceID: tenantID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, childID).Return(&child, nil)

	grandchild := service.ServiceDetails{ID: grandchildID, Name: grandchildID, ParentServiceID: childID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, grandchildID).Return(&grandchild, nil)

	result, err := t.cache.GetTenantID(grandchildID, t.getService)

	c.Assert(result, Equals, tenantID)
	c.Assert(err, IsNil)

	// Verify the cache contents
	expectedCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
		servicePath{
			serviceID:       grandchildID,
			parentID:        childID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID + "/" + grandchildID,
			serviceNamePath: "/" + tenantID + "/" + childID + "/" + grandchildID,
		},
	}
	t.assertExpectedCacheEntries(c, expectedCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetTenantID_UsesCachedValues(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID := "mockGrandChildId"

	// Load the cache manually
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
		servicePath{
			serviceID:       grandchildID,
			parentID:        childID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID + "/" + grandchildID,
			serviceNamePath: "/" + tenantID + "/" + childID + "/" + grandchildID,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	// NOTE: Since no mock expectations are set, the test will fail if any calls to t.getService occur
	result, err := t.cache.GetTenantID(grandchildID, t.getService)

	c.Assert(result, Equals, tenantID)
	c.Assert(err, IsNil)
	t.assertExpectedCacheEntries(c, initialCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetTenantID_UsesCachedValuesForParents(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID := "mockGrandChildId"

	// Load the cache manually
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	grandchild := service.ServiceDetails{ID: grandchildID, Name: grandchildID, ParentServiceID: childID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, grandchildID).Return(&grandchild, nil)

	// NOTE: Since no mock expectations are set for parentID or childID, the test will fail if there
	// any calls to t.getService for those entries
	result, err := t.cache.GetTenantID(grandchildID, t.getService)

	c.Assert(result, Equals, tenantID)
	c.Assert(err, IsNil)
	expectedCacheEntries := append(initialCacheEntries, servicePath{
		serviceID:       grandchildID,
		parentID:        childID,
		tenantID:        tenantID,
		servicePath:     "/" + tenantID + "/" + childID + "/" + grandchildID,
		serviceNamePath: "/" + tenantID + "/" + childID + "/" + grandchildID,
	})
	t.assertExpectedCacheEntries(c, expectedCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetTenantID_ReturnsDBError(c *C) {
	tenantID := "mockTenantId"
	expectedError := fmt.Errorf("fake DB error")
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, tenantID).Return(nil, expectedError)

	result, err := t.cache.GetTenantID(tenantID, t.getService)

	c.Assert(result, Equals, "")
	c.Assert(err.Error(), Equals, expectedError.Error())
}

func (t *ServicePathCacheTest) Test_GetServicePath_SeedsCacheForTenantService(c *C) {
	tenantID := "mockTenantId"
	expectedPath := "/" + tenantID

	expectedService := service.ServiceDetails{ID: tenantID, Name: tenantID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, tenantID).Return(&expectedService, nil)

	tenantResult, pathResult, err := t.cache.GetServicePath(tenantID, t.getService)

	c.Assert(tenantResult, Equals, tenantID)
	c.Assert(pathResult, Equals, expectedPath)
	c.Assert(err, IsNil)

	// Verify the cache contents
	expectedCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
	}
	t.assertExpectedCacheEntries(c, expectedCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetServicePath_SeedsCacheForNestedServices(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID := "mockGrandChildId"
	expectedPath := "/" + tenantID + "/" + childID + "/" + grandchildID

	tenant := service.ServiceDetails{ID: tenantID, Name: tenantID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, tenantID).Return(&tenant, nil)

	child := service.ServiceDetails{ID: childID, Name: childID, ParentServiceID: tenantID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, childID).Return(&child, nil)

	grandchild := service.ServiceDetails{ID: grandchildID, Name: grandchildID, ParentServiceID: childID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, grandchildID).Return(&grandchild, nil)

	tenantResult, pathResult, err := t.cache.GetServicePath(grandchildID, t.getService)

	c.Assert(tenantResult, Equals, tenantID)
	c.Assert(pathResult, Equals, expectedPath)
	c.Assert(err, IsNil)

	// Verify the cache contents
	expectedCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
		servicePath{
			serviceID:       grandchildID,
			parentID:        childID,
			tenantID:        tenantID,
			servicePath:     expectedPath,
			serviceNamePath: expectedPath,
		},
	}
	t.assertExpectedCacheEntries(c, expectedCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetServicePath_UsesCachedValues(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID := "mockGrandChildId"
	expectedPath := "/" + tenantID + "/" + childID + "/" + grandchildID

	// Load the cache manually
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
		servicePath{
			serviceID:       grandchildID,
			parentID:        childID,
			tenantID:        tenantID,
			servicePath:     expectedPath,
			serviceNamePath: expectedPath,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	// NOTE: Since no mock expectations are set, the test will fail if any calls to t.getService occur
	tenantResult, pathResult, err := t.cache.GetServicePath(grandchildID, t.getService)

	c.Assert(tenantResult, Equals, tenantID)
	c.Assert(pathResult, Equals, expectedPath)
	c.Assert(err, IsNil)
	t.assertExpectedCacheEntries(c, initialCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetServicePath_UsesCachedValuesForParents(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID := "mockGrandChildId"
	expectedPath := "/" + tenantID + "/" + childID + "/" + grandchildID

	// Load the cache manually
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	grandchild := service.ServiceDetails{ID: grandchildID, Name: grandchildID, ParentServiceID: childID}
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, grandchildID).Return(&grandchild, nil)

	// NOTE: Since no mock expectations are set for parentID or childID, the test will fail if there
	// any calls to t.getService for those entries
	tenantResult, pathResult, err := t.cache.GetServicePath(grandchildID, t.getService)

	c.Assert(tenantResult, Equals, tenantID)
	c.Assert(pathResult, Equals, expectedPath)
	c.Assert(err, IsNil)

	expectedCacheEntries := append(initialCacheEntries, servicePath{
		serviceID:       grandchildID,
		parentID:        childID,
		tenantID:        tenantID,
		servicePath:     expectedPath,
		serviceNamePath: expectedPath,
	})

	t.assertExpectedCacheEntries(c, expectedCacheEntries)
}

func (t *ServicePathCacheTest) Test_GetServicePath_ReturnsDBError(c *C) {
	tenantID := "mockTenantId"
	expectedError := fmt.Errorf("fake DB error")
	t.serviceStore.On("GetServiceDetails", t.unusedCTX, tenantID).Return(nil, expectedError)

	tenantResult, pathResult, err := t.cache.GetServicePath(tenantID, t.getService)

	c.Assert(tenantResult, Equals, "")
	c.Assert(pathResult, Equals, "")
	c.Assert(err.Error(), Equals, expectedError.Error())
}

func (t *ServicePathCacheTest) Test_RemoveIfParentChanged_OnEmptyCache(c *C) {
	tenantID := "mockTenantId"
	removed := t.cache.RemoveIfParentChanged(tenantID, "")

	// Nothing should be removed, so the cache should be unchanged
	c.Assert(removed, Equals, false)
	t.assertExpectedCacheEntries(c, []servicePath{})
}

func (t *ServicePathCacheTest) Test_RemoveIfParentChanged_OnCacheMiss(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	expectedPath := "/" + tenantID + "/" + childID
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     expectedPath,
			serviceNamePath: expectedPath,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	uncachedServiceID := "something compleletly differennt"
	removed := t.cache.RemoveIfParentChanged(uncachedServiceID, tenantID)

	// Nothing should be removed, so the cache should be unchanged
	c.Assert(removed, Equals, false)
	t.assertExpectedCacheEntries(c, initialCacheEntries)
}

func (t *ServicePathCacheTest) Test_RemoveIfParentChanged_CachedParentMatches(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	expectedPath := "/" + tenantID + "/" + childID
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     expectedPath,
			serviceNamePath: expectedPath,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	removed := t.cache.RemoveIfParentChanged(childID, tenantID)

	// Nothing should be removed, so the cache should be unchanged
	c.Assert(removed, Equals, false)
	t.assertExpectedCacheEntries(c, initialCacheEntries)
}

func (t *ServicePathCacheTest) Test_RemoveIfParentChanged_CachedParentDifferent(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	removed := t.cache.RemoveIfParentChanged(childID, "newParentId")

	// The child should be removed
	c.Assert(removed, Equals, true)
	t.assertExpectedCacheEntries(c, []servicePath{initialCacheEntries[0]})
}

func (t *ServicePathCacheTest) Test_RemoveIfParentChanged_RemoveMultipleEntries(c *C) {
	tenantID := "mockTenantId"
	childID := "mockChildId"
	grandchildID1 := "mockGrandChildId1"
	grandchildID2 := "mockGrandChildId2"

	// Load the cache manually
	initialCacheEntries := []servicePath{
		servicePath{
			serviceID:       tenantID,
			parentID:        "",
			tenantID:        tenantID,
			servicePath:     "/" + tenantID,
			serviceNamePath: "/" + tenantID,
		},
		servicePath{
			serviceID:       childID,
			parentID:        tenantID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID,
			serviceNamePath: "/" + tenantID + "/" + childID,
		},
		servicePath{
			serviceID:       grandchildID1,
			parentID:        childID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID + "/" + grandchildID1,
			serviceNamePath: "/" + tenantID + "/" + childID + "/" + grandchildID1,
		},
		servicePath{
			serviceID:       grandchildID2,
			parentID:        childID,
			tenantID:        tenantID,
			servicePath:     "/" + tenantID + "/" + childID + "/" + grandchildID2,
			serviceNamePath: "/" + tenantID + "/" + childID + "/" + grandchildID2,
		},
	}
	for _, expected := range initialCacheEntries {
		t.cache.paths[expected.serviceID] = expected
	}

	removed := t.cache.RemoveIfParentChanged(childID, "newParentId")

	// The item and its descendants should be removed
	c.Assert(removed, Equals, true)
	t.assertExpectedCacheEntries(c, []servicePath{initialCacheEntries[0]})
}

func (t *ServicePathCacheTest) Test_Reset_EmptyCache(c *C) {
	// should start empty
	c.Assert(len(t.cache.paths), Equals, 0)

	t.cache.Reset()

	// should start still be empty
	c.Assert(len(t.cache.paths), Equals, 0)
}

func (t *ServicePathCacheTest) getService(serviceID string) (*service.ServiceDetails, error) {
	svc, err := t.serviceStore.GetServiceDetails(t.unusedCTX, serviceID)
	if err != nil {
		return &service.ServiceDetails{}, err
	}
	return svc, nil
}

func (t *ServicePathCacheTest) assertExpectedCacheEntries(c *C, expectedCacheEntries []servicePath) {
	c.Assert(len(t.cache.paths), Equals, len(expectedCacheEntries))
	for _, expected := range expectedCacheEntries {
		actual, ok := t.cache.paths[expected.serviceID]
		c.Assert(ok, Equals, true)
		c.Assert(actual.tenantID, Equals, expected.tenantID)
		c.Assert(actual.parentID, Equals, expected.parentID)
		c.Assert(actual.servicePath, Equals, expected.servicePath)
		c.Assert(actual.serviceNamePath, Equals, expected.serviceNamePath)
	}
}
