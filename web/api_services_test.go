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
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	"github.com/stretchr/testify/mock"
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

var allServices = []service.ServiceDetails{
	serviceDetailsTestData.firstService,
	serviceDetailsTestData.secondService,
	serviceDetailsTestData.tenant,
}

func (s *TestWebSuite) TestRestPutServiceDetailsShouldReturnStatusOK(c *C) {
	body := `
	{
		"Name": "Zenoss.core",
		"Description": "Zenoss Core",
		"PoolID": "default",
		"Instances": 1,
		"RAMCommitment": "128M",
		"Startup": "redis-server /etc/redis.conf"
	}`

	request := s.buildRequest("PUT", "http://www.example.com/services/1a2b3c", body)
	request.PathParams["serviceId"] = "1a2b3c"

	s.mockFacade.
		On("GetService", s.ctx.getDatastoreContext(), "1a2b3c").
		Return(&service.Service{Name: "Service"}, nil)

	s.mockFacade.
		On("UpdateService", s.ctx.getDatastoreContext(), mock.AnythingOfType("service.Service")).
		Return(nil)

	putServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestPutServiceDetailsShouldReturnBadRequestForInvalidMessageBody(c *C) {
	body := `
	{
		"Description": "Zenoss Core",
		"PoolID": "default",
		"Instances": 1,
		"RAMCommitment": "128M",
		"Startup": "redis-server /etc/redis.conf"
	}`

	request := s.buildRequest("PUT", "http://www.example.com/services/1a2b3c", body)
	request.PathParams["serviceId"] = "1a2b3c"

	s.mockFacade.
		On("GetService", s.ctx.getDatastoreContext(), "1a2b3c").
		Return(&service.Service{Name: "Service"}, nil)

	s.mockFacade.
		On("UpdateService", s.ctx.getDatastoreContext(), mock.AnythingOfType("service.Service")).
		Return(nil)

	putServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestRestPutServiceDetailsShouldSetValuesCorrectly(c *C) {
	body := `
	{
		"Name": "Zenoss.core",
		"Description": "Zenoss Core",
		"PoolID": "default",
		"Instances": 1,
		"RAMCommitment": "1000000",
		"Startup": "redis-server /etc/redis.conf"
	}`

	request := s.buildRequest("PUT", "http://www.example.com/services/1a2b3c", body)
	request.PathParams["serviceId"] = "1a2b3c"

	s.mockFacade.
		On("GetService", s.ctx.getDatastoreContext(), "1a2b3c").
		Return(&service.Service{ID: "1a2b3c"}, nil)

	var calledService service.Service

	s.mockFacade.
		On("UpdateService", s.ctx.getDatastoreContext(), mock.AnythingOfType("service.Service")).
		Return(nil).
		Run(func(a mock.Arguments) {
			calledService = a.Get(1).(service.Service)
		})

	putServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(calledService.ID, Equals, "1a2b3c")
	c.Assert(calledService.Name, Equals, "Zenoss.core")
	c.Assert(calledService.Description, Equals, "Zenoss Core")
	c.Assert(calledService.PoolID, Equals, "default")
	c.Assert(calledService.Instances, Equals, 1)
	c.Assert(calledService.RAMCommitment.Value, Equals, uint64(1000000))
	c.Assert(calledService.Startup, Equals, "redis-server /etc/redis.conf")
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/tenant/services", "")
	request.PathParams["serviceId"] = "tenant"

	s.mockFacade.
		On("GetServiceDetailsByParentID", s.ctx.getDatastoreContext(), "tenant", time.Duration(0)).
		Return([]service.ServiceDetails{serviceDetailsTestData.firstService}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
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

func (s *TestWebSuite) TestRestGetServiceDetailsShouldReturnStatusNotFoundIfNoService(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/firstservice/services", "")
	request.PathParams["serviceId"] = "firstservice"

	expectedError := datastore.ErrNoSuchEntity{Key: datastore.NewKey("service", "firstservice")}
	s.mockFacade.
		On("GetServiceDetails", s.ctx.getDatastoreContext(), "firstservice").
		Return(nil, expectedError)

	getServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusNotFound)
}

func (s *TestWebSuite) TestRestGetAllServiceDetailsShouldReturnStatusOK(c *C) {
	s.mockFacade.On("QueryServiceDetails",
		mock.Anything,
		mock.AnythingOfType("service.Query")).
		Return(allServices, nil)

	request := s.buildRequest("GET", "http://www.example.com/services", "")

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	response := []service.ServiceDetails{}
	s.getResult(c, &response)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestQueryServiceDetailsShouldQueryForAll(c *C) {
	s.mockFacade.On("QueryServiceDetails",
		mock.Anything,
		mock.AnythingOfType("service.Query")).
		Return(allServices, nil)

	request := s.buildRequest("GET", "http://www.example.com/services", "")

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	query := service.Query{Tags: []string{}}
	s.mockFacade.AssertCalled(c, "QueryServiceDetails", s.ctx.getDatastoreContext(), query)
}

func (s *TestWebSuite) TestRestQueryServiceDetailsShouldQueryWithName(c *C) {
	s.mockFacade.On("QueryServiceDetails",
		mock.Anything,
		mock.AnythingOfType("service.Query")).
		Return(allServices, nil)

	request := s.buildRequest("GET", "http://www.example.com/services?name=Service", "")

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	query := service.Query{Name: "Service", Tags: []string{}}
	s.mockFacade.AssertCalled(c, "QueryServiceDetails", s.ctx.getDatastoreContext(), query)
}

func (s *TestWebSuite) TestRestQueryServiceDetailsShouldQueryForTenants(c *C) {
	s.mockFacade.On("QueryServiceDetails",
		mock.Anything,
		mock.AnythingOfType("service.Query")).
		Return(allServices, nil)

	request := s.buildRequest("GET", "http://www.example.com/services?tenants", "")

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	query := service.Query{Tenants: true, Tags: []string{}}
	s.mockFacade.AssertCalled(c, "QueryServiceDetails", s.ctx.getDatastoreContext(), query)
}

func (s *TestWebSuite) TestRestQueryServiceDetailsShouldQueryWithTags(c *C) {
	s.mockFacade.On("QueryServiceDetails",
		mock.Anything,
		mock.AnythingOfType("service.Query")).
		Return(allServices, nil)

	request := s.buildRequest("GET", "http://www.example.com/services?tags=daemon,collector", "")

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	query := service.Query{Tags: []string{"daemon", "collector"}}
	s.mockFacade.AssertCalled(c, "QueryServiceDetails", s.ctx.getDatastoreContext(), query)
}

func (s *TestWebSuite) TestRestQueryServiceDetailsShouldQueryForUpdated(c *C) {
	s.mockFacade.On("QueryServiceDetails",
		mock.Anything,
		mock.AnythingOfType("service.Query")).
		Return(allServices, nil)

	request := s.buildRequest("GET", "http://www.example.com/services?since=5000", "")

	getAllServiceDetails(&(s.writer), &request, s.ctx)

	query := service.Query{Tags: []string{}, Since: time.Duration(5 * time.Second)}
	s.mockFacade.AssertCalled(c, "QueryServiceDetails", s.ctx.getDatastoreContext(), query)
}
