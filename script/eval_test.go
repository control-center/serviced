// Copyright 2015, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"errors"

	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_evalSvcMigrate(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	testRunner.runnerObj.svcMigrate = testRunner.svcMigrate

	expectedServiceID := "TEST_SERVICE_ID_FROM_PATH"
	expectedScriptBody := testRunner.testNode.args[1]
	testRunner.Mock.
		On("svcMigrate", expectedServiceID, expectedScriptBody).
		Return(nil)

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, IsNil)
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfNoSvcFromPath(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	testRunner.runnerObj.svcFromPath = nil

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, ErrorMatches, "no service id lookup function for SVC_MIGRATE")
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfTenantNotFound(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	delete(testRunner.runnerObj.env, "TENANT_ID")

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, ErrorMatches, "no service tenant id specified for SVC_MIGRATE")
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfServiceFromPathNotFound(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	errorStub := errors.New("ErrorStub: service lookup failed")
	testRunner.runnerObj.svcFromPath = func(tenantID string, path string) (string, error) { return "", errorStub }

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, ErrorMatches, "ErrorStub: service lookup failed")
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfServiceNotFound(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	testRunner.runnerObj.svcFromPath = func(tenantID string, path string) (string, error) { return "", nil }

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, ErrorMatches, "no service id found for zope")
}

type EvalTestRunner struct {
	mock.Mock
	runnerObj *runner
	testNode  node
}

func (etr *EvalTestRunner) svcMigrate(serviceID string, scriptBody string) error {
	return etr.Mock.Called(serviceID, scriptBody).Error(0)
}

func setupRunner(testNode node) *EvalTestRunner {
	testConfig := Config{
		NoOp:          true,
		ServiceID:     "TEST_SERVICE_ID_12345",
		TenantLookup:  func(service string) (string, error) { return "TEST_TENANT_ID", nil },
		SvcIDFromPath: func(tenantID string, path string) (string, error) { return "TEST_SERVICE_ID_FROM_PATH", nil },
	}
	testRunner := runner{
		config:      &testConfig,
		env:         make(map[string]string),
		svcFromPath: testConfig.SvcIDFromPath,
	}

	// The tenant ID
	testRunner.env["TENANT_ID"] = string("TEST_TENANT_ID")

	evalRunner := EvalTestRunner{
		runnerObj: &testRunner,
		testNode:  testNode,
	}

	return &evalRunner
}

func setupMigrateNode() node {
	migrateNode := node{
		cmd:     "SVC_MIGRATE zope somescript.py",
		args:    []string{"zope", "somescript.py"},
		line:    "SVC_MIGRATE zope somescript.py",
		lineNum: 0,
	}
	return migrateNode
}
