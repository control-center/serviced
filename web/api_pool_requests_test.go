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

	"github.com/control-center/serviced/domain/pool"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *TestWebSuite) TestReadPoolsWithIncorrectSortParameter(c *C) {
	request := s.buildRequest("GET", "/pools?skip=1&pull=20&order=asc&sort=incorrect", "")

	s.mockFacade.
		On("GetResourcePoolsByPage", s.ctx.getDatastoreContext(), mock.AnythingOfType("pool.ResourcePoolsQuery")).
		Return(&pool.ResourcePoolsResponse{Results: []pool.ResourcePool{}, Total: 0}, nil)

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestReadPoolsWithIncorrectOrderParameter(c *C) {
	request := s.buildRequest("GET", "/pools?skip=1&pull=20&order=incorrect&sort=ID", "")

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestReadPoolsWithNonIntegerSkipParameter(c *C) {
	request := s.buildRequest("GET", "/pools?skip=ABC&pull=20&order=asc&sort=ID", "")

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestReadPoolsWithSkipParameterLessThanZero(c *C) {
	request := s.buildRequest("GET", "/pools?skip=-1&pull=20&order=asc&sort=ID", "")

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestReadPoolsWithNonIntegerPullParameter(c *C) {
	request := s.buildRequest("GET", "/pools?skip=1&pull=incorrect&order=asc&sort=ID", "")

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestReadPoolsWithPullParameterLessThanZero(c *C) {
	request := s.buildRequest("GET", "/pools?skip=-1&pull=-1&order=asc&sort=ID", "")

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusBadRequest)
}

func (s *TestWebSuite) TestReadPoolsDefaultSkipToZero(c *C) {
	request := s.buildRequest("GET", "/apipools?&pull=10&order=asc&sort=ID", "")

	var query pool.ResourcePoolsQuery

	s.mockFacade.
		On("GetResourcePoolsByPage", s.ctx.getDatastoreContext(), mock.AnythingOfType("pool.ResourcePoolsQuery")).
		Return(&pool.ResourcePoolsResponse{Results: []pool.ResourcePool{}, Total: 0}, nil).
		Run(func(args mock.Arguments) { query = args.Get(1).(pool.ResourcePoolsQuery) })

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(query.Skip, Equals, 0)
}

func (s *TestWebSuite) TestReadPoolsDefaultPullTo50000(c *C) {
	request := s.buildRequest("GET", "/apipools?skip=1&order=asc&sort=ID", "")

	var query pool.ResourcePoolsQuery

	s.mockFacade.
		On("GetResourcePoolsByPage", s.ctx.getDatastoreContext(), mock.AnythingOfType("pool.ResourcePoolsQuery")).
		Return(&pool.ResourcePoolsResponse{Results: []pool.ResourcePool{}, Total: 0}, nil).
		Run(func(args mock.Arguments) { query = args.Get(1).(pool.ResourcePoolsQuery) })

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(query.Pull, Equals, 50000)
}

func (s *TestWebSuite) TestReadPoolsDefaultOrderToAsc(c *C) {
	request := s.buildRequest("GET", "/apipools?skip=1&pull=20&sort=ID", "")

	var query pool.ResourcePoolsQuery

	s.mockFacade.
		On("GetResourcePoolsByPage", s.ctx.getDatastoreContext(), mock.AnythingOfType("pool.ResourcePoolsQuery")).
		Return(&pool.ResourcePoolsResponse{Results: []pool.ResourcePool{}, Total: 0}, nil).
		Run(func(args mock.Arguments) { query = args.Get(1).(pool.ResourcePoolsQuery) })

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(query.Order, Equals, "asc")
}

func (s *TestWebSuite) TestReadPoolsDefaultSortToID(c *C) {
	request := s.buildRequest("GET", "/apipools?skip=1&pull=20&order=asc", "")

	var query pool.ResourcePoolsQuery

	s.mockFacade.
		On("GetResourcePoolsByPage", s.ctx.getDatastoreContext(), mock.AnythingOfType("pool.ResourcePoolsQuery")).
		Return(&pool.ResourcePoolsResponse{Results: []pool.ResourcePool{}, Total: 0}, nil).
		Run(func(args mock.Arguments) { query = args.Get(1).(pool.ResourcePoolsQuery) })

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(query.Sort, Equals, "ID")
}
