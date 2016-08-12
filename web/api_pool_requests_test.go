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

	"github.com/control-center/serviced/domain/pool"
	. "gopkg.in/check.v1"
)

type apiPoolsTestData struct {
	firstPool  pool.ResourcePool
	secondPool pool.ResourcePool
	selfLink   Link
}

func newAPIPoolsTestData() apiPoolsTestData {
	return apiPoolsTestData{
		firstPool: pool.ResourcePool{
			ID:                "firstPool",
			Description:       "The first pool",
			MemoryCapacity:    10000,
			MemoryCommitment:  5000,
			CoreCapacity:      15,
			ConnectionTimeout: 10,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},

		secondPool: pool.ResourcePool{
			ID:                "secondPool",
			Description:       "The second pool",
			MemoryCapacity:    20000,
			MemoryCommitment:  15000,
			CoreCapacity:      10,
			ConnectionTimeout: 2,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},

		selfLink: Link{
			HRef:   "/pools",
			Rel:    "self",
			Method: "GET",
		},
	}
}

func (s *TestWebSuite) TestRestGetPoolsShouldReturnStatusOK(c *C) {
	data := newAPIPoolsTestData()
	request := s.buildRequest("GET", "/pools", "")

	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return([]pool.ResourcePool{data.firstPool}, nil)

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}

func (s *TestWebSuite) TestRestGetPoolsShouldReturnCorrectValuesForReadPool(c *C) {
	data := newAPIPoolsTestData()
	request := s.buildRequest("GET", "/pools", "")

	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return([]pool.ResourcePool{data.firstPool}, nil)

	getPools(&(s.writer), &request, s.ctx)

	response := poolsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Results[0].ID, Equals, data.firstPool.ID)
	c.Assert(response.Results[0].Description, Equals, data.firstPool.Description)
	c.Assert(response.Results[0].MemoryCapacity, Equals, data.firstPool.MemoryCapacity)
	c.Assert(response.Results[0].MemoryCommitment, Equals, data.firstPool.MemoryCommitment)
	c.Assert(response.Results[0].CoreCapacity, Equals, data.firstPool.CoreCapacity)
	c.Assert(response.Results[0].ConnectionTimeout, Equals, data.firstPool.ConnectionTimeout)

	createdEquals := response.Results[0].CreatedAt.Equal(data.firstPool.CreatedAt)
	c.Assert(createdEquals, Equals, true)

	updateEquals := response.Results[0].UpdatedAt.Equal(data.firstPool.UpdatedAt)
	c.Assert(updateEquals, Equals, true)
}

func (s *TestWebSuite) TestRestGetPoolsShouldReturnCorrectValueForTotal(c *C) {
	data := newAPIPoolsTestData()
	request := s.buildRequest("GET", "/pools", "")

	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return([]pool.ResourcePool{data.firstPool, data.secondPool}, nil)

	getPools(&(s.writer), &request, s.ctx)

	response := poolsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Total, Equals, 2)
}

func (s *TestWebSuite) TestRestGetPoolsShouldReturnCorrectLinkValues(c *C) {
	data := newAPIPoolsTestData()
	request := s.buildRequest("GET", "http://www.example.com/pools", "")

	s.mockFacade.
		On("GetResourcePools", s.ctx.getDatastoreContext()).
		Return([]pool.ResourcePool{data.firstPool, data.secondPool}, nil)

	getPools(&(s.writer), &request, s.ctx)

	response := poolsResponse{}
	s.getResult(c, &response)

	c.Assert(response.Links[0].HRef, Equals, data.selfLink.HRef)
	c.Assert(response.Links[0].Rel, Equals, data.selfLink.Rel)
	c.Assert(response.Links[0].Method, Equals, data.selfLink.Method)
}
