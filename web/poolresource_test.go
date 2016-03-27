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

	"github.com/control-center/serviced/domain/pool"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *TestWebSuite) TestRestAddPool(c *C) {
	poolID := "testPool"
	poolJson := `{"ID": "` + poolID + `", "Description": "test pool"}`
	request := s.buildRequest("POST", "/pool/add", poolJson)
	s.mockFacade.
		On("AddResourcePool", s.ctx.getDatastoreContext(), mock.AnythingOfType("*pool.ResourcePool")).
		Return(nil)

	restAddPool(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, "Added resource pool", poolLinks(poolID))
}

func (s *TestWebSuite) TestRestAddPoolFailsForBadJSON(c *C) {
	request := s.buildRequest("POST", "/pool/add", "{this is not valid json}")

	restAddPool(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddPoolFails(c *C) {
	request := s.buildRequest("POST", "/pool/add", `{"ID": "somePool"}`)
	expectedError := fmt.Errorf("mock add failed")
	s.mockFacade.
		On("AddResourcePool", s.ctx.getDatastoreContext(), mock.AnythingOfType("*pool.ResourcePool")).
		Return(expectedError)

	restAddPool(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestBuildPoolMonitoringProfile(c *C) {
	pool := pool.ResourcePool{}
	err := buildPoolMonitoringProfile(&pool, []string{}, s.mockFacade, s.ctx.getDatastoreContext())

	c.Assert(err, IsNil)
	c.Assert(len(pool.MonitoringProfile.MetricConfigs), Not(Equals), 0)
	c.Assert(len(pool.MonitoringProfile.GraphConfigs), Equals, 3)
	// FIXME: validate the expected content of the metric and graph configs
}
