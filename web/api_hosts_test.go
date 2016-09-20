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

	"github.com/control-center/serviced/domain/host"
	. "gopkg.in/check.v1"
)

var apiHostsTestData = struct {
	firstHost  host.ReadHost
	secondHost host.ReadHost
}{
	firstHost: host.ReadHost{
		ID:            "firstHost",
		Name:          "FirstHost",
		PoolID:        "Pool",
		Cores:         12,
		Memory:        15000,
		RAMLimit:      "50%",
		KernelVersion: "1.1.1",
		KernelRelease: "1.2.3",
		ServiceD: host.ReadServiced{
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
		ServiceD: host.ReadServiced{
			Version: "1.2.3.4.5",
			Date:    "1/1/1999",
			Release: "Release",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	},
}

func (s *TestWebSuite) TestGetHostsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/hosts", "")

	s.mockFacade.
		On("GetReadHosts", s.ctx.getDatastoreContext()).
		Return([]host.ReadHost{apiHostsTestData.firstHost}, nil)

	getHosts(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestGetHostsForPoolShouldReturnBadRequestForInvalidPoolId(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/pools/inv%ZZlid/hosts", "")
	request.PathParams["poolId"] = "inv%ZZlid"
	getHostsForPool(&(s.writer), &request, s.ctx)
	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestGetHostsForPoolShouldReturnBadRequestForMissingPoolId(c *C) {
	request := s.buildRequest("GET", "http://www.example.com/pools/hosts", "")
	request.PathParams["poolId"] = ""
	getHostsForPool(&(s.writer), &request, s.ctx)
	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}
