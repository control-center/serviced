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

package web

import (
	"fmt"
	"net/http"

	"github.com/control-center/serviced/domain/host"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *TestWebSuite) TestRestGetHosts(c *C) {
	expectedHosts := []host.Host{
		{ID: "host1"},
		{ID: "host2"},
	}
	expectedResult := map[string]host.Host{
		"host1": expectedHosts[0],
		"host2": expectedHosts[1],
	}
	request := s.buildRequest("GET", "/hosts", "")
	s.mockFacade.
		On("GetHosts", s.ctx.getDatastoreContext()).
		Return(expectedHosts, nil)
	
	restGetHosts(&(s.writer), &request, s.ctx)
	
	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := map[string]host.Host{}
	s.getResult(c, &actualResult)
	s.assertMapKeys(c, actualResult, expectedResult)
	for hostID, host := range actualResult {
		c.Assert(host.ID, Equals, expectedResult[hostID].ID)
	}
}

func (s *TestWebSuite) TestRestGetHostsReturnsEmptyList(c *C) {
	emptyHosts := []host.Host{}
	request := s.buildRequest("GET", "/hosts", "")
	s.mockFacade.
		On("GetHosts", s.ctx.getDatastoreContext()).
		Return(emptyHosts, nil)

	restGetHosts(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := map[string]host.Host{}
	s.getResult(c, &actualResult)
	c.Assert(len(actualResult), Equals, 0)
}

func (s *TestWebSuite) TestRestGetHostsFails(c *C) {
	expectedError := fmt.Errorf("mock GetHosts failed")
	request := s.buildRequest("GET", "/hosts", "")
	s.mockFacade.
		On("GetHosts", s.ctx.getDatastoreContext()).
		Return(nil, expectedError)

	restGetHosts(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetActiveHostIDs(c *C) {
	expectedHostIDs := []string{
		"host1",
		"host2",
	}
	request := s.buildRequest("GET", "/hosts/running", "")
	s.mockFacade.
		On("GetActiveHostIDs", s.ctx.getDatastoreContext()).
		Return(expectedHostIDs, nil)

	restGetActiveHostIDs(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := []string{}
	s.getResult(c, &actualResult)
	c.Assert(len(actualResult), Equals, len(expectedHostIDs))
	c.Assert(actualResult, DeepEquals, expectedHostIDs)
}

func (s *TestWebSuite) TestRestGetActiveHostIDFails(c *C) {
	expectedError := fmt.Errorf("mock GetActiveHostIDs failed")
	request := s.buildRequest("GET", "/hosts/running", "")
	s.mockFacade.
		On("GetActiveHostIDs", s.ctx.getDatastoreContext()).
		Return(nil, expectedError)

	restGetActiveHostIDs(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetHost(c *C) {
	hostID := "someHostID"
	expectedHost := host.Host{ID: hostID}
	request := s.buildRequest("GET", "/hosts/someHostID", "")
	request.PathParams["hostId"] = hostID
	s.mockFacade.
		On("GetHost", s.ctx.getDatastoreContext(), hostID).
		Return(&expectedHost, nil)

	restGetHost(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := host.Host{}
	s.getResult(c, &actualResult)
	c.Assert(actualResult.ID, Equals, expectedHost.ID)
}

func (s *TestWebSuite) TestRestGetHostFails(c *C) {
	expectedError := fmt.Errorf("mock GetHost failed")
	hostID := "someHostID"
	request := s.buildRequest("GET", "/hosts/someHostID", "")
	request.PathParams["hostId"] = hostID
	s.mockFacade.
		On("GetHost", s.ctx.getDatastoreContext(), hostID).
		Return(nil, expectedError)

	restGetHost(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestGetHostFailsForInvalidURL(c *C) {
	request := s.buildRequest("GET", "/hosts", "")
	request.PathParams["hostId"] = "%zzz"

	restGetHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestGetHostFailsForMissingHostID(c *C) {
	request := s.buildRequest("GET", "/hosts", "")

	restGetHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddHostFailsForBadJSON(c *C) {
	request := s.buildRequest("POST", "/hosts/add", "{this is not valid json}")

	restAddHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddHostFailsForBadIP(c *C) {
	hostID := "testHost"
	hostJSON := `{"ID": "` + hostID + `", "Description": "test host", "IPAddr": "badip:4979"}`
	request := s.buildRequest("POST", "/hosts/add", hostJSON)

	restAddHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddHostFailsForMissingPort(c *C) {
	hostID := "testHost"
	hostJSON := `{"ID": "` + hostID + `", "Description": "test host", "IPAddr": "127.0.0.1"}`
	request := s.buildRequest("POST", "/hosts/add", hostJSON)

	restAddHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddHostFailsForInvalidPort(c *C) {
	hostID := "testHost"
	hostJSON := `{"ID": "` + hostID + `", "Description": "test host", "IPAddr": "127.0.0.1:badPort"}`
	request := s.buildRequest("POST", "/hosts/add", hostJSON)

	restAddHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestUpdateHost(c *C) {
	hostID := "testHost"
	hostJSON := `{"ID": "` + hostID + `", "Description": "test host"}`
	request := s.buildRequest("PUT", "/hosts/testHost", hostJSON)
	request.PathParams["hostId"] = hostID
	s.mockFacade.
		On("UpdateHost", s.ctx.getDatastoreContext(), mock.AnythingOfType("*host.Host")).
		Return(nil)

	restUpdateHost(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, "Updated host", hostLinks(hostID))
}

func (s *TestWebSuite) TestRestUpdateHostFails(c *C) {
	expectedError := fmt.Errorf("mock UpdateHost failed")
	hostID := "testHost"
	hostJSON := `{"ID": "` + hostID + `", "Description": "test host"}`
	request := s.buildRequest("PUT", "/hosts/testHost", hostJSON)
	request.PathParams["hostId"] = hostID
	s.mockFacade.
		On("UpdateHost", s.ctx.getDatastoreContext(), mock.AnythingOfType("*host.Host")).
		Return(expectedError)

	restUpdateHost(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestUpdateHostFailsForInvalidURL(c *C) {
	request := s.buildRequest("PUT", "/hosts/%zzz", "")
	request.PathParams["hostId"] = "%zzz"

	restUpdateHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestUpdateHostFailsForBadJSON(c *C) {
	request := s.buildRequest("PUT", "/hosts/someHostID", "{this is not valid json}")
	request.PathParams["hostId"] = "someHostID"

	restUpdateHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestUpdateHostFailsForMissingHostID(c *C) {
	request := s.buildRequest("PUT", "/hosts", `{"ID": "someHostID"}`)

	restUpdateHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestRemoveHost(c *C) {
	hostID := "testHost"
	request := s.buildRequest("DELETE", "/hosts/testHost", "")
	request.PathParams["hostId"] = hostID
	s.mockFacade.
		On("RemoveHost", s.ctx.getDatastoreContext(), hostID).
		Return(nil)

	restRemoveHost(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, "Removed host", hostsLinks())
}

func (s *TestWebSuite) TestRestRemoveHostFails(c *C) {
	expectedError := fmt.Errorf("mock RemoveHost failed")
	hostID := "testHost"
	request := s.buildRequest("DELETE", "/hosts/testHost", "")
	request.PathParams["hostId"] = hostID
	s.mockFacade.
		On("RemoveHost", s.ctx.getDatastoreContext(), hostID).
		Return(expectedError)

	restRemoveHost(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestRemoveHostFailsForInvalidURL(c *C) {
	request := s.buildRequest("DELETE", "/hosts/%zzz", "")
	request.PathParams["hostId"] = "%zzz"

	restRemoveHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestRemoveHostFailsForMissingHostID(c *C) {
	request := s.buildRequest("DELETE", "/hosts/testHost", "")

	restRemoveHost(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestBuildHostMonitoringProfile(c *C) {
	host := host.Host{}
	err := buildHostMonitoringProfile(&host)

	c.Assert(err, IsNil)
	c.Assert(len(host.MonitoringProfile.MetricConfigs), Not(Equals), 0)
	c.Assert(len(host.MonitoringProfile.GraphConfigs), Equals, 6)
	// FIXME: validate the expected content of the metric and graph configs
}
