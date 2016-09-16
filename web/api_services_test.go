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
	"net/http"

	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	. "gopkg.in/check.v1"
)

var serviceDetailsTestData = struct {
	tenant        service.ServiceDetails
	firstService  service.ServiceDetails
	secondService service.ServiceDetails
}{
	tenant: service.ServiceDetails{
		ID:              "tenant",
		Name:            "Tenant Name",
		Description:     "Tenant Description",
		PoolID:          "pool",
		ParentServiceID: "",
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     2,
			Default: 1,
		},
		Startup: "firstservice -start",
	},
	firstService: service.ServiceDetails{
		ID:              "firstService",
		Name:            "First Service Name",
		Description:     "The first child service",
		PoolID:          "pool",
		ParentServiceID: "tenant",
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     2,
			Default: 1,
		},
		Startup: "firstservice -start",
	},
	secondService: service.ServiceDetails{
		ID:              "secondService",
		Name:            "Second Service Name",
		Description:     "The second child service",
		PoolID:          "pool",
		ParentServiceID: "tenant",
		InstanceLimits: domain.MinMax{
			Min:     0,
			Max:     10,
			Default: 1,
		},
		Startup: "secondservice -start",
	},
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/tenant/services", "")
	request.PathParams["serviceId"] = "tenant"

	s.mockFacade.
		On("GetServiceDetailsByParentID", s.ctx.getDatastoreContext(), "tenant").
		Return([]service.ServiceDetails{serviceDetailsTestData.firstService}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnCorrectValueForTotal(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/tenant/services", "")
	request.PathParams["serviceId"] = "tenant"

	s.mockFacade.
		On("GetServiceDetailsByParentID", s.ctx.getDatastoreContext(), "tenant").
		Return([]service.ServiceDetails{serviceDetailsTestData.firstService, serviceDetailsTestData.secondService}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	response := serviceDetailsListResponse{}
	s.getResult(c, &response)

	c.Assert(response.Total, Equals, 2)
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnCorrectLinkValues(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/tenant/services", "")
	request.PathParams["serviceId"] = "tenant"

	s.mockFacade.
		On("GetServiceDetailsByParentID", s.ctx.getDatastoreContext(), "tenant").
		Return([]service.ServiceDetails{serviceDetailsTestData.firstService, serviceDetailsTestData.secondService}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	response := serviceDetailsListResponse{}
	s.getResult(c, &response)

	c.Assert(response.Links[0].HRef, Equals, "/services/tenant/services")
	c.Assert(response.Links[0].Rel, Equals, "self")
	c.Assert(response.Links[0].Method, Equals, "GET")
}

func (s *TestWebSuite) TestRestGetServiceDetailsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/firstservice/services", "")
	request.PathParams["serviceId"] = "firstservice"

	s.mockFacade.
		On("GetServiceDetails", s.ctx.getDatastoreContext(), "firstservice").
		Return(&serviceDetailsTestData.firstService, nil)

	getServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestGetServiceDetailsShouldReturnCorrectLinkValues(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/firstservice/services", "")
	request.PathParams["serviceId"] = "firstservice"

	s.mockFacade.
		On("GetServiceDetails", s.ctx.getDatastoreContext(), "firstservice").
		Return(&serviceDetailsTestData.firstService, nil)

	getServiceDetails(&(s.writer), &request, s.ctx)

	response := serviceDetailsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Links[0].HRef, Equals, "/services/firstservice/services")
	c.Assert(response.Links[0].Rel, Equals, "self")
	c.Assert(response.Links[0].Method, Equals, "GET")
}

func (s *TestWebSuite) TestRestGetServiceDetailsShouldReturnStatusNotFoundIfNoService(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/firstservice/services", "")
	request.PathParams["serviceId"] = "firstservice"

	s.mockFacade.
		On("GetServiceDetails", s.ctx.getDatastoreContext(), "firstservice").
		Return(nil, nil)

	getServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusNotFound)
}

func (s *TestWebSuite) TestRestGetAllServiceDetailsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services", "")

	s.mockFacade.
		On("GetAllServiceDetails", s.ctx.getDatastoreContext()).
		Return([]service.ServiceDetails{
			serviceDetailsTestData.firstService,
			serviceDetailsTestData.secondService,
			serviceDetailsTestData.tenant,
		}, nil)

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestGetAllServiceDetailsShouldReturnCorrectValueForTotal(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services", "")

	s.mockFacade.
		On("GetAllServiceDetails", s.ctx.getDatastoreContext()).
		Return([]service.ServiceDetails{
			serviceDetailsTestData.firstService,
			serviceDetailsTestData.secondService,
			serviceDetailsTestData.tenant,
		}, nil)

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	response := serviceDetailsListResponse{}
	s.getResult(c, &response)

	c.Assert(response.Total, Equals, 3)
}

func (s *TestWebSuite) TestRestGetAllServiceDetailsShouldReturnCorrectLinkValues(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services", "")

	s.mockFacade.
		On("GetAllServiceDetails", s.ctx.getDatastoreContext()).
		Return([]service.ServiceDetails{
			serviceDetailsTestData.firstService,
			serviceDetailsTestData.secondService,
			serviceDetailsTestData.tenant,
		}, nil)

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	response := serviceDetailsListResponse{}
	s.getResult(c, &response)

	c.Assert(response.Links[0].HRef, Equals, "/services")
	c.Assert(response.Links[0].Rel, Equals, "self")
	c.Assert(response.Links[0].Method, Equals, "GET")
}

func (s *TestWebSuite) TestRestGetAllServiceDetailsShouldOnlyReturnTenants(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services?tenants", "")

	s.mockFacade.
		On("GetServiceDetailsByParentID", s.ctx.getDatastoreContext(), "").
		Return([]service.ServiceDetails{
			serviceDetailsTestData.tenant,
		}, nil)

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	response := serviceDetailsListResponse{}
	s.getResult(c, &response)

	c.Assert(len(response.Results), Equals, 1)
	c.Assert(response.Results[0].ID, Equals, "tenant")
}
