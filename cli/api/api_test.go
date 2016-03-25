// Copyright 2015 The Serviced Authors.
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

package api

import (
	"testing"

	daomocks "github.com/control-center/serviced/dao/mocks"
	"github.com/control-center/serviced/rpc/master/mocks"

	. "gopkg.in/check.v1"
)

var tGlobal *testing.T

// Hook up gocheck into the "go test" runner.
func TestAPI(t *testing.T) {
	TestingT(t)

	// HACK ALERT: Some functions in testify.Mock require testing.T, but
	//	gocheck doesn't offer access to it, so we have to save a copy in
	//	a global variable :=(
	tGlobal = t
}

var _ = Suite(&TestAPISuite{})

type TestAPISuite struct {
	api API

	//  A mock implementation of the ControlPlane interface
	mockControlPlane *daomocks.ControlPlane

	mockMasterClient *mocks.ClientInterface
}

func (s *TestAPISuite) SetUpTest(c *C) {
	s.mockControlPlane = &daomocks.ControlPlane{}
	s.mockMasterClient = &mocks.ClientInterface{}

	apiObj := NewAPI(s.mockMasterClient, nil, nil, s.mockControlPlane)
	s.api = apiObj
}

func (s *TestAPISuite) TearDownTest(c *C) {
	// don't allow per-test-case values to be reused across test cases
	s.api = nil
	s.mockControlPlane = nil
}
