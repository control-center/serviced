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

var apiPoolsTestData = struct {
	firstPool  pool.ReadPool
	secondPool pool.ReadPool
}{

	firstPool: pool.ReadPool{
		ID:                "firstPool",
		Description:       "The first pool",
		MemoryCapacity:    10000,
		MemoryCommitment:  5000,
		CoreCapacity:      15,
		ConnectionTimeout: 10,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	},

	secondPool: pool.ReadPool{
		ID:                "secondPool",
		Description:       "The second pool",
		MemoryCapacity:    20000,
		MemoryCommitment:  15000,
		CoreCapacity:      10,
		ConnectionTimeout: 2,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	},
}

func (s *TestWebSuite) TestRestGetPoolsShouldReturnStatusOK(c *C) {
	request := s.buildRequest("GET", "/pools", "")

	s.mockFacade.
		On("GetReadPools", s.ctx.getDatastoreContext()).
		Return([]pool.ReadPool{apiPoolsTestData.firstPool}, nil)

	getPools(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
}
