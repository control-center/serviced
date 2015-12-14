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
	"fmt"
	"strings"
	"testing"

	"github.com/control-center/serviced/dao"
	daomocks "github.com/control-center/serviced/dao/mocks"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/rpc/master/mocks"

	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var tGlobal *testing.T

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)

	// HACK ALERT: Some functions in testify.Mock require testing.T, but
	//	gocheck doesn't offer access to it, so we have to save a copy in
	//	a global variable :=(
	tGlobal = t
}

var _ = Suite(&serviceAPITest{})

type serviceAPITest struct {
	api API

	//  A mock implementation of the ControlPlane interface
	mockControlPlane *daomocks.ControlPlane

	mockMasterClient *mocks.ClientInterface
}

func (st *serviceAPITest) SetUpTest(c *C) {
	st.mockControlPlane = &daomocks.ControlPlane{}
	st.mockMasterClient = &mocks.ClientInterface{}

	apiObj := NewAPI(st.mockMasterClient, nil, nil, st.mockControlPlane)
	st.api = apiObj
}

func (st *serviceAPITest) TearDownTest(c *C) {
	// don't allow per-test-case values to be reused across test cases
	st.api = nil
	st.mockControlPlane = nil
}

func (st *serviceAPITest) TestGetService_works(c *C) {
	serviceID := "test-service"
	expected, _ := service.NewService()

	st.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(nil).Run(func(a mock.Arguments) {
		svc := a.Get(1).(*service.Service)
		*svc = *expected
	})
	actual, err := st.api.GetService(serviceID)
	c.Assert(err, IsNil)
	c.Assert(actual.ID, Equals, expected.ID)
}

func (st *serviceAPITest) TestGetService_fails(c *C) {
	errorStub := errors.New("errorStub: GetService() failed")
	serviceID := "test-service"
	st.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(errorStub)
	actual, err := st.api.GetService(serviceID)
	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (st *serviceAPITest) TestMigrateService_works(c *C) {
	serviceID := "test-service"
	scriptBody := "# no-op script"
	inputScript := strings.NewReader(scriptBody)
	expected, _ := service.NewService()
	sdkVersion := "a.b.c"
	st.mockControlPlane.On("GetService", serviceID, mock.AnythingOfType("*service.Service")).Return(nil).Run(func(a mock.Arguments) {
		svc := a.Get(1).(*service.Service)
		*svc = *expected
	})
	st.mockControlPlane.On("RunMigrationScript", mock.AnythingOfType("dao.RunMigrationScriptRequest"), mock.AnythingOfType("*int")).Return(nil).Run(func(a mock.Arguments) {
		req := a.Get(0).(dao.RunMigrationScriptRequest)
		c.Assert(req.ServiceID, Equals, serviceID)
		c.Assert(req.ScriptBody, Equals, scriptBody)
		c.Assert(req.DryRun, Equals, true)
		c.Assert(req.SDKVersion, Equals, sdkVersion)
	})
	actual, err := st.api.RunMigrationScript(serviceID, inputScript, true, sdkVersion)
	c.Assert(err, IsNil)
	c.Assert(actual.ID, Equals, expected.ID)
	st.mockControlPlane.AssertExpectations(c)
}

type mockInputReader struct {
	Mock *mock.Mock
}

func (m mockInputReader) Read(p []byte) (n int, err error) {
	args := m.Mock.Called(p)
	return args.Int(0), args.Error(1)
}

func (st *serviceAPITest) TestMigrateService_failsToReadScript(c *C) {
	serviceID := "test-service"
	errorStub := errors.New("errorStub: Read() failed")
	mockInput := mockInputReader{Mock: &mock.Mock{}}
	mockInput.Mock.
		On("Read", mock.Anything).
		Return(0, errorStub)

	actual, err := st.api.RunMigrationScript(serviceID, mockInput, false, "")

	c.Assert(actual, IsNil)
	expectedError := fmt.Errorf("could not read migration script: %s", errorStub)
	c.Assert(err.Error(), Equals, expectedError.Error())
	/*

		// RunMigrationScript should never be called if we can't read the script
		args := st.mockControlPlane.GetArgsForMockCall("MigrateServce")
		c.Assert(len(args), Equals, 0)
	*/
}

func (st *serviceAPITest) TestMigrateService_failsForEmptyScript(c *C) {
	serviceID := "test-service"
	scriptBody := ""
	inputScript := strings.NewReader(scriptBody)

	actual, err := st.api.RunMigrationScript(serviceID, inputScript, false, "")

	c.Assert(actual, IsNil)
	expectedError := fmt.Errorf("migration failed: script is empty")
	c.Assert(err.Error(), Equals, expectedError.Error())

	/*
		// RunMigrationScript should never be called if we can't read the script
		args := st.mockControlPlane.GetArgsForMockCall("MigrateServce")
		c.Assert(len(args), Equals, 0)
	*/
}

func (st *serviceAPITest) TestMigrateService_fails(c *C) {
	serviceID := "test-service"
	scriptBody := "# no-op script"
	inputScript := strings.NewReader(scriptBody)

	errorStub := errors.New("errorStub: migrate failed")
	st.mockControlPlane.
		On("RunMigrationScript", mock.Anything, mock.Anything).
		Return(errorStub)

	actual, err := st.api.RunMigrationScript(serviceID, inputScript, false, "")

	c.Assert(actual, IsNil)
	expectedError := fmt.Errorf("migration failed: %s", errorStub)
	c.Assert(err.Error(), Equals, expectedError.Error())
}

func (st *serviceAPITest) TestGetEndpoints_fails(c *C) {
	errorStub := errors.New("errorStub: GetServiceEndpoints() failed")
	serviceID := "test-service"

	st.mockMasterClient.
		On("GetServiceEndpoints", []string{serviceID}, true, true, true).
		Return(nil, errorStub)

	actual, err := st.api.GetEndpoints(serviceID, true, true, true)

	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (st *serviceAPITest) TestGetEndpoints_works(c *C) {
	serviceID := "test-service"

	st.mockMasterClient.
		On("GetServiceEndpoints", []string{serviceID}, true, true, true).
		Return([]applicationendpoint.EndpointReport{}, nil)

	actual, err := st.api.GetEndpoints(serviceID, true, true, true)

	c.Assert(err, IsNil)
	c.Assert(actual, NotNil)
	c.Assert(len(actual), Equals, 0)
}
