// Copyright 2015, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package api

import (
	"errors"
	"io"

	"github.com/control-center/serviced/domain/service"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (st *serviceAPITest) Test_script_cliServiceMigrate(t *C) {
	mockAPI := new(mockAPI)
	mockAPI.Mock.
		On("MigrateService", "serviceID", mock.Anything, false, "").
		Return(nil, nil)
	serviceMigrateFunc := cliServiceMigrate(mockAPI)

	err := serviceMigrateFunc("serviceID", "testMigrate.txt", "")

	t.Assert(err, IsNil)
}

func (st *serviceAPITest) Test_script_cliServiceMigrateWithVersion(t *C) {
	mockAPI := new(mockAPI)
	mockAPI.Mock.
		On("MigrateService", "serviceID", mock.Anything, false, "1.2.3").
		Return(nil, nil)
	serviceMigrateFunc := cliServiceMigrate(mockAPI)

	err := serviceMigrateFunc("serviceID", "testMigrate.txt", "1.2.3")

	t.Assert(err, IsNil)
}

func (st *serviceAPITest) Test_script_cliServiceMigrate_fails(t *C) {
	errorStub := errors.New("errorStub: migration failed")
	mockAPI := new(mockAPI)
	mockAPI.Mock.
		On("MigrateService", "serviceID", mock.Anything, false, "").
		Return(nil, errorStub)
	serviceMigrateFunc := cliServiceMigrate(mockAPI)

	err := serviceMigrateFunc("serviceID", "testMigrate.txt", "")

	t.Assert(err, ErrorMatches, "Migration failed for service serviceID: errorStub: migration failed")
}

func (st *serviceAPITest) Test_script_cliServiceMigrate_failsForInvalidFile(t *C) {
	serviceMigrateFunc := cliServiceMigrate(st.api)

	err := serviceMigrateFunc("serviceID", "path/to/bogus/file", "")

	t.Assert(err, ErrorMatches, "Could not open migration script: open path/to/bogus/file: no such file or directory")
}

type mockAPI struct {
	mock.Mock
	API
}

func (mock *mockAPI) MigrateService(serviceID string, input io.Reader, dryRun bool, sdkVersion string) (*service.Service, error) {
	args := mock.Mock.Called(serviceID, input, dryRun, sdkVersion)

	var svc *service.Service
	if arg0 := args.Get(0); arg0 != nil {
		svc = arg0.(*service.Service)
	}
	return svc, args.Error(1)
}
