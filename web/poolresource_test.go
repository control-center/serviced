// Copyright 2014 The Serviced Authors.
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

package web

import (
	"fmt"
	"net/http"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *TestWebSuite) TestRestGetPools(c *C) {
	expectedPools := []pool.ResourcePool{
		{ID: "pool1"},
		{ID: "pool2"},
	}
	expectedResult := map[string]pool.ResourcePool{
		"pool1": expectedPools[0],
		"pool2": expectedPools[1],
	}
	request := s.buildRequest("GET", "/pools", "")
	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return(expectedPools, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPools[0].ID).
		Return([]host.Host{}, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPools[1].ID).
		Return([]host.Host{}, nil)

	restGetPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)

	actualResult := map[string]pool.ResourcePool{}
	s.getResult(c, &actualResult)
	s.assertMapKeys(c, actualResult, expectedResult)
	for poolID, pool := range actualResult {
		c.Assert(pool.ID, Equals, expectedResult[poolID].ID)
	}
}

func (s *TestWebSuite) TestRestGetPoolsReturnsEmptyList(c *C) {
	emptyPools := []pool.ResourcePool{}
	request := s.buildRequest("GET", "/pools", "")
	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return(emptyPools, nil)

	restGetPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := map[string]pool.ResourcePool{}
	s.getResult(c, &actualResult)
	c.Assert(len(actualResult), Equals, 0)
}

func (s *TestWebSuite) TestRestGetPoolsFails(c *C) {
	expectedError := fmt.Errorf("mock GetResourcePools failed")
	request := s.buildRequest("GET", "/pools", "")
	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return(nil, expectedError)

	restGetPools(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetPoolsWhenFindHostsInPoolFails(c *C) {
	expectedError := fmt.Errorf("mock FindHostsInPool failed")
	expectedPools := []pool.ResourcePool{
		{ID: "pool1"},
		{ID: "pool2"},
	}
	request := s.buildRequest("GET", "/pools", "")
	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return(expectedPools, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPools[0].ID).
		Return(nil, expectedError)

	restGetPools(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetPoolsWhenGetHostFails(c *C) {
	expectedError := fmt.Errorf("mock GetHost failed")
	expectedPools := []pool.ResourcePool{
		{ID: "pool1"},
		{ID: "pool2"},
	}
	expectedHosts := []host.Host{
		{ID: "host1"},
	}
	request := s.buildRequest("GET", "/pools", "")
	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return(expectedPools, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPools[0].ID).
		Return(expectedHosts, nil)
	s.mockFacade.
		On("GetHost", s.ctx.getDatastoreContext(), expectedHosts[0].ID).
		Return(nil, expectedError)

	restGetPools(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetPool(c *C) {
	poolID := "somePool"
	expectedPool := pool.ResourcePool{
		ID: poolID,
	}
	request := s.buildRequest("GET", "/pools/somePool", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("GetResourcePool", s.ctx.getDatastoreContext(), poolID).
		Return(&expectedPool, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPool.ID).
		Return([]host.Host{}, nil)

	restGetPool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := pool.ResourcePool{}
	s.getResult(c, &actualResult)
	c.Assert(actualResult.ID, Equals, expectedPool.ID)
}

func (s *TestWebSuite) TestRestGetPoolFails(c *C) {
	expectedError := fmt.Errorf("mock GetResourcePool failed")
	poolID := "somePool"
	request := s.buildRequest("GET", "/pools/somePool", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("GetResourcePool", s.ctx.getDatastoreContext(), poolID).
		Return(nil, expectedError)

	restGetPool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetPoolFailsForInvalidURL(c *C) {
	request := s.buildRequest("GET", "/pools/%zzz", "")
	request.PathParams["poolId"] = "%zzz"

	restGetPool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetPoolFailsForMissingPoolID(c *C) {
	request := s.buildRequest("GET","/pools", "")

	restGetPool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetPoolWhenFindHostsInPoolFails(c *C) {
	expectedError := fmt.Errorf("mock FindHostsInPool failed")
	poolID := "somePool"
	expectedPool := pool.ResourcePool{
		ID: poolID,
	}
	request := s.buildRequest("GET", "/pools/somePool", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("GetResourcePool", s.ctx.getDatastoreContext(), poolID).
		Return(&expectedPool, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPool.ID).
		Return(nil, expectedError)

	restGetPool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetPoolWhenGetHostFails(c *C) {
	expectedError := fmt.Errorf("mock GetHost failed")
	poolID := "somePool"
	expectedPool := pool.ResourcePool{
		ID: poolID,
	}
	expectedHosts := []host.Host{
		{ID: "host1"},
	}
	request := s.buildRequest("GET", "/pools/somePool", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("GetResourcePool", s.ctx.getDatastoreContext(), poolID).
		Return(&expectedPool, nil)
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), expectedPool.ID).
		Return(expectedHosts, nil)
	s.mockFacade.
		On("GetHost", s.ctx.getDatastoreContext(), expectedHosts[0].ID).
		Return(nil, expectedError)

	restGetPool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestAddPool(c *C) {
	poolID := "testPool"
	poolJSON := `{"ID": "` + poolID + `", "Description": "test pool"}`
	request := s.buildRequest("POST", "/pools/add", poolJSON)
	s.mockFacade.
		On("AddResourcePool", s.ctx.getDatastoreContext(), mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil)

	restAddPool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, "Added resource pool", poolLinks(poolID))
}

func (s *TestWebSuite) TestRestAddPoolFailsForBadJSON(c *C) {
	request := s.buildRequest("POST", "/pools/add", "{this is not valid json}")

	restAddPool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddPoolFails(c *C) {
	request := s.buildRequest("POST", "/pools/add", `{"ID": "somePool"}`)
	expectedError := fmt.Errorf("mock AddResourcePool failed")
	s.mockFacade.
		On("AddResourcePool", s.ctx.getDatastoreContext(), mock.AnythingOfType("*pool.ResourcePool")).
		Return(expectedError)

	restAddPool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestUpdatePool(c *C) {
	poolID := "testPool"
	poolJSON := `{"ID": "` + poolID + `", "Description": "test pool"}`
	request := s.buildRequest("PUT", "/pools/testPool/", poolJSON)
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("UpdateResourcePool", s.ctx.getDatastoreContext(), mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil)

	restUpdatePool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, "Updated resource pool", poolLinks(poolID))
}

func (s *TestWebSuite) TestRestUpdatePoolFails(c *C) {
	poolID := "testPool"
	poolJSON := `{"ID": "` + poolID + `", "Description": "test pool"}`
	request := s.buildRequest("PUT", "/pools/testPool", poolJSON)
	request.PathParams["poolId"] = poolID
	expectedError := fmt.Errorf("mock UpdateResourcePool failed")
	s.mockFacade.
		On("UpdateResourcePool", s.ctx.getDatastoreContext(), mock.AnythingOfType("*pool.ResourcePool")).
		Return(expectedError)

	restUpdatePool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestUpdatePoolFailsForInvalidURL(c *C) {
	request := s.buildRequest("PUT", "/pools/%zzz", `{"ID": "somePool"}`)
	request.PathParams["poolId"] = "%zzz"

	restUpdatePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestUpdatePoolFailsForBadJSON(c *C) {
	request := s.buildRequest("PUT", "/pools/someID", "{this is not valid json}")
	request.PathParams["poolId"] = "someID"

	restUpdatePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestUpdatePoolFailsForMissingPoolID(c *C) {
	request := s.buildRequest("PUT", "/pools", `{"ID": "somePool"}`)

	restUpdatePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestRemovePool(c *C) {
	poolID := "testPool"
	request := s.buildRequest("DELETE", "/pools/testPool", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("RemoveResourcePool", s.ctx.getDatastoreContext(), poolID).
		Return(nil)

	restRemovePool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, "Removed resource pool", poolsLinks())
}

func (s *TestWebSuite) TestRestRemovePoolFails(c *C) {
	poolID := "testPool"
	request := s.buildRequest("DELETE", "/pools/testPool", "")
	request.PathParams["poolId"] = poolID
	expectedError := fmt.Errorf("mock RemoveResourcePool failed")
	s.mockFacade.
		On("RemoveResourcePool", s.ctx.getDatastoreContext(), poolID).
		Return(expectedError)

	restRemovePool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestRemovePoolFailsForInvalidURL(c *C) {
	request := s.buildRequest("DELETE", "/pools/%zzz", "")
	request.PathParams["poolId"] = "%zzz"

	restRemovePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestRemovePoolFailsForMissingPoolID(c *C) {
	request := s.buildRequest("DELETE", "/pools", `{"ID": "somePool"}`)

	restRemovePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetHostsForResourcePool(c *C) {
	poolID := "testPool"
	request := s.buildRequest("GET", "/pools/testPool/hosts", "")
	request.PathParams["poolId"] = poolID
	expectedHosts := []host.Host{
		{ID: "host1", IPAddr: "192.10.168.1"},
		{ID: "host2", IPAddr: "192.10.168.2"},
	}
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), poolID).
		Return(expectedHosts, nil)

	restGetHostsForResourcePool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := []pool.PoolHost{}
	s.getResult(c, &actualResult)
	c.Assert(len(actualResult), Equals, len(expectedHosts))
	c.Assert(actualResult[0].HostID, Equals, expectedHosts[0].ID)
	c.Assert(actualResult[0].PoolID, Equals, poolID)
	c.Assert(actualResult[0].HostIP, Equals, expectedHosts[0].IPAddr)
	c.Assert(actualResult[1].HostID, Equals, expectedHosts[1].ID)
	c.Assert(actualResult[1].PoolID, Equals, poolID)
	c.Assert(actualResult[1].HostIP, Equals, expectedHosts[1].IPAddr)
}

func (s *TestWebSuite) TestRestGetHostsForResourcePoolReturnsEmptyList(c *C) {
	poolID := "testPool"
	request := s.buildRequest("GET", "/pools/testPool/hosts", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), poolID).
		Return([]host.Host{}, nil)

	restGetHostsForResourcePool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := []pool.PoolHost{}
	s.getResult(c, &actualResult)
	c.Assert(len(actualResult), Equals, 0)
}

func (s *TestWebSuite) TestRestGetHostsForResourcePoolFails(c *C) {
	expectedError := fmt.Errorf("mock FindHostsInPool failed")
	poolID := "testPool"
	request := s.buildRequest("GET", "/pools/testPool/hosts", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("FindHostsInPool", s.ctx.getDatastoreContext(), poolID).
		Return(nil, expectedError)

	restGetHostsForResourcePool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetHostsForResourcePoolFailsForInvalidURL(c *C) {
	request := s.buildRequest("GET", "/pools/%zzz/hosts", "")
	request.PathParams["poolId"] = "%zzz"

	restGetHostsForResourcePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetHostsForResourcePoolFailsForMissingPoolID(c *C) {
	request := s.buildRequest("GET", "/pools/somePoolID/hosts", "")

	restGetHostsForResourcePool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetPoolIps(c *C) {
	poolID := "testPool"
	request := s.buildRequest("GET", "/pools/testPool/ips", "")
	request.PathParams["poolId"] = poolID
	expectedIps := pool.PoolIPs{
		PoolID:  poolID,
		HostIPs: []host.HostIPResource{{HostID: "host1"}},
	}
	s.mockFacade.
		On("GetPoolIPs", s.ctx.getDatastoreContext(), poolID).
		Return(&expectedIps, nil)

	restGetPoolIps(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := pool.PoolIPs{}
	s.getResult(c, &actualResult)
	c.Assert(actualResult.PoolID, Equals, expectedIps.PoolID)
	c.Assert(len(actualResult.HostIPs), Equals, len(expectedIps.HostIPs))
	c.Assert(actualResult.HostIPs[0].HostID, Equals, expectedIps.HostIPs[0].HostID)
}

func (s *TestWebSuite) TestRestGetPoolIpsFails(c *C) {
	expectedError := fmt.Errorf("mock GetPoolIPs failed")
	poolID := "testPool"
	request := s.buildRequest("GET", "/pools/testPool/ips", "")
	request.PathParams["poolId"] = poolID
	s.mockFacade.
		On("GetPoolIPs", s.ctx.getDatastoreContext(), poolID).
		Return(nil, expectedError)

	restGetPoolIps(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetPoolIpsForInvalidURL(c *C) {
	request := s.buildRequest("GET", "/pools/%zzz/ips", "")
	request.PathParams["poolId"] = "%zzz"

	restGetPoolIps(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetPoolIpsFailsForMissingPoolID(c *C) {
	request := s.buildRequest("GET", "/pools/testPool/ips", "")

	restGetPoolIps(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestBuildPoolMonitoringProfile(c *C) {
	pool := pool.ResourcePool{}
	err := buildPoolMonitoringProfile(&pool, []string{}, s.mockFacade, s.ctx.getDatastoreContext())

	c.Assert(err, IsNil)
	c.Assert(len(pool.MonitoringProfile.MetricConfigs), Not(Equals), 0)
	c.Assert(len(pool.MonitoringProfile.GraphConfigs), Equals, 3)
	// FIXME: validate the expected content of the metric and graph configs
}
