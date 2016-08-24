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
	"net/http"
	"time"

	"github.com/control-center/serviced/domain/host"
	. "gopkg.in/check.v1"
)

type apiHostsTestData struct {
	firstHost  host.ReadHost
	secondHost host.ReadHost
	selfLink   APILink
}

func newAPIHostsTestData() apiHostsTestData {
	return apiHostsTestData{
		firstHost: host.ReadHost{
			ID:            "firstHost",
			Name:          "FirstHost",
			PoolID:        "Pool",
			Cores:         12,
			Memory:        15000,
			RAMLimit:      "50%",
			KernelVersion: "1.1.1",
			KernelRelease: "1.2.3",
			ServiceD: host.ReadServiceD{
				Version: "1.2.3.4.5",
				Date:    "1/1/1999",
				Release: "Release",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		secondHost: host.ReadHost{
			ID:            "secondHost",
			Name:          "SecondHost",
			PoolID:        "Pool",
			Cores:         7,
			Memory:        10000,
			RAMLimit:      "70%",
			KernelVersion: "1.1.1",
			KernelRelease: "1.2.3",
			ServiceD: host.ReadServiceD{
				Version: "1.2.3.4.5",
				Date:    "1/1/1999",
				Release: "Release",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		selfLink: APILink{
			HRef:   "/hosts",
			Rel:    "self",
			Method: "GET",
		},
	}
}

func (s *TestWebSuite) TestRestGetHostsShouldReturnStatusOK(c *C) {
	data := newAPIHostsTestData()
	request := s.buildRequest("GET", "http://www.example.com/hosts", "")

	s.mockFacade.
		On("GetReadHosts", s.ctx.getDatastoreContext()).
		Return([]host.ReadHost{data.firstHost}, nil)

	getHosts(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestGetHostsShouldReturnCorrectValuesForReadPool(c *C) {
	data := newAPIHostsTestData()
	request := s.buildRequest("GET", "http://www.example.com/hosts", "")

	s.mockFacade.
		On("GetReadHosts", s.ctx.getDatastoreContext()).
		Return([]host.ReadHost{data.firstHost}, nil)

	getHosts(&(s.writer), &request, s.ctx)

	response := hostsResponse{}
	s.getResult(c, &response)

	host := response.Results[0]

	c.Assert(host.ID, Equals, data.firstHost.ID)
	c.Assert(host.Name, Equals, data.firstHost.Name)
	c.Assert(host.PoolID, Equals, data.firstHost.PoolID)
	c.Assert(host.Cores, Equals, data.firstHost.Cores)
	c.Assert(host.Memory, Equals, data.firstHost.Memory)
	c.Assert(host.RAMLimit, Equals, data.firstHost.RAMLimit)
	c.Assert(host.KernelVersion, Equals, data.firstHost.KernelVersion)
	c.Assert(host.KernelRelease, Equals, data.firstHost.KernelRelease)
	c.Assert(host.ServiceD.Date, Equals, data.firstHost.ServiceD.Date)
	c.Assert(host.ServiceD.Release, Equals, data.firstHost.ServiceD.Release)
	c.Assert(host.ServiceD.Version, Equals, data.firstHost.ServiceD.Version)
	c.Assert(host.CreatedAt, Equals, data.firstHost.CreatedAt)
	c.Assert(host.UpdatedAt, Equals, data.firstHost.UpdatedAt)
}

func (s *TestWebSuite) TestRestGetHostsShouldReturnCorrectValueForTotal(c *C) {
	data := newAPIHostsTestData()
	request := s.buildRequest("GET", "http://www.example.com/hosts", "")

	s.mockFacade.
		On("GetReadHosts", s.ctx.getDatastoreContext()).
		Return([]host.ReadHost{data.firstHost, data.secondHost}, nil)

	getHosts(&(s.writer), &request, s.ctx)

	response := hostsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Total, Equals, 2)
}

func (s *TestWebSuite) TestRestGetHostsShouldReturnCorrectLinkValues(c *C) {
	data := newAPIHostsTestData()
	request := s.buildRequest("GET", "http://www.example.com/hosts", "")

	s.mockFacade.
		On("GetReadHosts", s.ctx.getDatastoreContext()).
		Return([]host.ReadHost{data.firstHost, data.secondHost}, nil)

	getHosts(&(s.writer), &request, s.ctx)

	response := hostsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Links[0].HRef, Equals, data.selfLink.HRef)
	c.Assert(response.Links[0].Rel, Equals, data.selfLink.Rel)
	c.Assert(response.Links[0].Method, Equals, data.selfLink.Method)
}
