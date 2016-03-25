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

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *TestAPISuite) TestGetService_works(c *C) {
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

func (s *TestAPISuite) TestGetService_fails(c *C) {
	errorStub := errors.New("errorStub: GetService() failed")
	serviceID := "test-service"
	s.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(errorStub)
	actual, err := s.api.GetService(serviceID)
	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}

type mockInputReader struct {
	Mock *mock.Mock
}

func (m mockInputReader) Read(p []byte) (n int, err error) {
	args := m.Mock.Called(p)
	return args.Int(0), args.Error(1)
}

func (s *TestAPISuite) TestGetEndpoints_fails(c *C) {
	errorStub := errors.New("errorStub: GetServiceEndpoints() failed")
	serviceID := "test-service"

	s.mockMasterClient.
		On("GetServiceEndpoints", []string{serviceID}, true, true, true).
		Return(nil, errorStub)

	actual, err := s.api.GetEndpoints(serviceID, true, true, true)

	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (s *TestAPISuite) TestGetEndpoints_works(c *C) {
	serviceID := "test-service"

	s.mockMasterClient.
		On("GetServiceEndpoints", []string{serviceID}, true, true, true).
		Return([]applicationendpoint.EndpointReport{}, nil)

	actual, err := s.api.GetEndpoints(serviceID, true, true, true)

	c.Assert(err, IsNil)
	c.Assert(actual, NotNil)
	c.Assert(len(actual), Equals, 0)
}
