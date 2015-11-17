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

package api

import (
	"errors"

	"strings"

//	"github.com/control-center/serviced/dao"
//	"github.com/control-center/serviced/domain/service"

	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *CliAPITestSuite) TestGetService_works(c *C) {
	serviceID := "test-service"
	expected, _ := service.NewService()

	s.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(nil).Run(func(a mock.Arguments) {
		svc := a.Get(1).(*service.Service)
		*svc = *expected
	})
	actual, err := s.api.GetService(serviceID)
	c.Assert(err, IsNil)
	c.Assert(actual.ID, Equals, expected.ID)
}

func (s *CliAPITestSuite) TestGetService_fails(c *C) {
	errorStub := errors.New("errorStub: GetService() failed")
	serviceID := "test-service"
	s.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(errorStub)
	actual, err := s.api.GetService(serviceID)
	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (s *CliAPITestSuite) TestMigrateService_works(c *C) {
	serviceID := "test-service"
	scriptBody := "# no-op script"
	inputScript := strings.NewReader(scriptBody)
	expected, _ := service.NewService()
	sdkVersion := "a.b.c"
	s.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(nil).Run(func(a mock.Arguments) {
		svc := a.Get(1).(*service.Service)
		*svc = *expected
	})
	s.mockControlPlane.On("RunMigrationScript", mock.AnythingOfType("dao.RunMigrationScriptRequest"), mock.AnythingOfType("*int")).Return(nil).Run(func(a mock.Arguments) {
		req := a.Get(0).(dao.RunMigrationScriptRequest)
		c.Assert(req.ServiceID, Equals, serviceID)
		c.Assert(req.ScriptBody, Equals, scriptBody)
		c.Assert(req.DryRun, Equals, true)
		c.Assert(req.SDKVersion, Equals, sdkVersion)
	})
	actual, err := s.api.RunMigrationScript(serviceID, inputScript, true, sdkVersion)
	c.Assert(err, IsNil)
	c.Assert(actual.ID, Equals, expected.ID)
	s.mockControlPlane.AssertExpectations(c)
}

type mockInputReader struct {
	Mock *mock.Mock
}

func (m mockInputReader) Read(p []byte) (n int, err error) {
	args := m.Mock.Called(p)
	return args.Int(0), args.Error(1)
}

func (s *CliAPITestSuite) TestGetEndpoints_fails(c *C) {
	errorStub := errors.New("errorStub: GetServiceEndpoints() failed")
	serviceID := "test-service"

	s.mockMasterClient.
		On("GetServiceEndpoints", []string{serviceID}, true, true, true).
		Return(nil, errorStub)

	actual, err := s.api.GetEndpoints(serviceID, true, true, true)

	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (s *CliAPITestSuite) TestGetEndpoints_works(c *C) {
	serviceID := "test-service"

	s.mockMasterClient.
		On("GetServiceEndpoints", []string{serviceID}, true, true, true).
		Return([]applicationendpoint.EndpointReport{}, nil)

	actual, err := s.api.GetEndpoints(serviceID, true, true, true)

	c.Assert(err, IsNil)
	c.Assert(actual, NotNil)
	c.Assert(len(actual), Equals, 0)
}
