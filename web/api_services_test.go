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

	"github.com/control-center/serviced/domain/service"
	. "gopkg.in/check.v1"
)

var serviceDetailsTestData = struct {
	firstDetails  service.ServiceDetails
	secondDetails service.ServiceDetails
}{

	firstDetails: service.ServiceDetails{
		ID:              "firstDetails",
		Name:            "firstDetailsName",
		Description:     "The first child service details",
		PoolID:          "pool",
		ParentServiceID: "parentService",
	},

	secondDetails: service.ServiceDetails{
		ID:              "secondDetails",
		Name:            "secondDetailsName",
		Description:     "The second child service details",
		PoolID:          "pool",
		ParentServiceID: "parentService",
	},
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/parentService/services", "")
	request.PathParams["serviceId"] = "parentService"

	s.mockFacade.
		On("GetChildServiceDetails", s.ctx.getDatastoreContext(), "parentService").
		Return([]service.ServiceDetails{serviceDetailsTestData.firstDetails}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnCorrectValueForTotal(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/parentService/services", "")
	request.PathParams["serviceId"] = "parentService"

	s.mockFacade.
		On("GetChildServiceDetails", s.ctx.getDatastoreContext(), "parentService").
		Return([]service.ServiceDetails{serviceDetailsTestData.firstDetails, serviceDetailsTestData.secondDetails}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	response := childServiceDetailsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Total, Equals, 2)
}

func (s *TestWebSuite) TestRestGetChildServiceDetailsShouldReturnCorrectLinkValues(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/services/parentService/services", "")
	request.PathParams["serviceId"] = "parentService"

	s.mockFacade.
		On("GetChildServiceDetails", s.ctx.getDatastoreContext(), "parentService").
		Return([]service.ServiceDetails{serviceDetailsTestData.firstDetails, serviceDetailsTestData.secondDetails}, nil)

	getChildServiceDetails(&(s.writer), &request, s.ctx)

	response := childServiceDetailsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Links[0].HRef, Equals, "/services/parentService/services")
	c.Assert(response.Links[0].Rel, Equals, "self")
	c.Assert(response.Links[0].Method, Equals, "GET")
}
