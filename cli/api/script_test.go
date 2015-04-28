// Copyright 2015, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package api

import (
	"errors"

	"github.com/control-center/serviced/domain/service"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (st *serviceAPITest) Test_script_cliServiceMigrate(t *C) {
	mockAPI := new(mockAPI)
	mockAPI.Mock.
		On("RunEmbeddedMigrationScript", "serviceID", "testMigrate.txt", false, "").
		Return(nil, nil)
	serviceMigrateFunc := cliServiceMigrate(mockAPI)

	err := serviceMigrateFunc("serviceID", "testMigrate.txt", "")

	t.Assert(err, IsNil)
}

func (st *serviceAPITest) Test_script_cliServiceMigrateWithVersion(t *C) {
	mockAPI := new(mockAPI)
	mockAPI.Mock.
		On("RunEmbeddedMigrationScript", "serviceID", "testMigrate.txt", false, "1.2.3").
		Return(nil, nil)
	serviceMigrateFunc := cliServiceMigrate(mockAPI)

	err := serviceMigrateFunc("serviceID", "testMigrate.txt", "1.2.3")

	t.Assert(err, IsNil)
}

func (st *serviceAPITest) Test_script_cliServiceMigrate_fails(t *C) {
	errorStub := errors.New("errorStub: migration failed")
	mockAPI := new(mockAPI)
	mockAPI.Mock.
		On("RunEmbeddedMigrationScript", "serviceID", "testMigrate.txt", false, "").
		Return(nil, errorStub)
	serviceMigrateFunc := cliServiceMigrate(mockAPI)

	err := serviceMigrateFunc("serviceID", "testMigrate.txt", "")

	t.Assert(err, ErrorMatches, "Migration failed for service serviceID: errorStub: migration failed")
}

type mockAPI struct {
	mock.Mock
	API
}

func (mock *mockAPI) RunEmbeddedMigrationScript(serviceID string, scriptFile string, dryRun bool, sdkVersion string) (*service.Service, error) {
	args := mock.Mock.Called(serviceID, scriptFile, dryRun, sdkVersion)

	var svc *service.Service
	if arg0 := args.Get(0); arg0 != nil {
		svc = arg0.(*service.Service)
	}
	return svc, args.Error(1)
}
