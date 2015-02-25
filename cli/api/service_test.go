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

package api

import (
	"errors"
	"testing"
	"unsafe"

	daotest "github.com/control-center/serviced/dao/test"
	"github.com/control-center/serviced/domain/service"

	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&serviceAPITest{})

type serviceAPITest struct {
	api API

	//  A mock implementation of the ControlPlane interface
	mockControlPlane *daotest.MockControlPlane
}

func (st *serviceAPITest) SetUpTest(c *C) {
	st.mockControlPlane = daotest.New()

	apiObj := NewAPI(nil, nil, nil, st.mockControlPlane)
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

	st.mockControlPlane.Responses["GetService"] = (unsafe.Pointer)(expected)
	st.mockControlPlane.
		On("GetService", serviceID, mock.Anything).
		Return(nil)

	actual, err := st.api.GetService(serviceID)

	c.Assert(err, IsNil)
	c.Assert(actual.ID, Equals, expected.ID)
}

func (st *serviceAPITest) TestGetService_fails(c *C) {
	errorStub := errors.New("errorStub: GetService() failed")
	serviceID := "test-service"

	st.mockControlPlane.
		On("GetService", serviceID, mock.Anything).
		Return(errorStub)

	actual, err := st.api.GetService(serviceID)

	c.Assert(actual, IsNil)
	c.Assert(err, Equals, errorStub)
}
