// Copyright 2015, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	// "errors"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_evalSvcMigrate(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	testRunner.runnerObj.svcMigrate = testRunner.svcMigrate

	expectedServiceID := "TEST_TENANT_ID"
	expectedScriptBody := testRunner.testNode.args[0]
	expectedSDKVersion := ""
	testRunner.Mock.
		On("svcMigrate", expectedServiceID, expectedScriptBody, expectedSDKVersion).
		Return(nil)

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, IsNil)
}

func (vs *ScriptSuite) Test_evalSvcMigrate_withSDKVersion(t *C) {
	testRunner := setupRunner(setupMigrateNodeWithSDKVer())
	testRunner.runnerObj.svcMigrate = testRunner.svcMigrate

	expectedServiceID := "TEST_TENANT_ID"
	expectedScriptBody := testRunner.testNode.args[1]
	expectedSDKVersion := "1.2.3"
	testRunner.Mock.
		On("svcMigrate", expectedServiceID, expectedScriptBody, expectedSDKVersion).
		Return(nil)

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, IsNil)
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfTenantNotFound(t *C) {
	testRunner := setupRunner(setupMigrateNode())
	delete(testRunner.runnerObj.env, "TENANT_ID")

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	t.Assert(err, ErrorMatches, "no service tenant id specified for SVC_MIGRATE")
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfSDKMissingValue(t *C) {
	migrateNode := node{
		cmd:     "SVC_MIGRATE SDK= somescript.py",
		args:    []string{"SDK=", "somescript.py"},
		line:    "SVC_MIGRATE SDK= somescript.py",
		lineNum: 0,
	}
	testRunner := setupRunner(migrateNode)

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	// We can't use ErrorMatches because it wants to do some kind of regex on our regex
	t.Assert(err, NotNil)
	t.Assert(err.Error(), Equals, "arg SDK= did not match ^SDK=([a-zA-Z0-9.\\-_]+)$")
}

func (vs *ScriptSuite) Test_evalSvcMigrate_failsIfFirstArgInvalid(t *C) {
	migrateNode := node{
		cmd:     "SVC_MIGRATE SDK=!@#$= somescript.py",
		args:    []string{"SDK=!@#$=", "somescript.py"},
		line:    "SVC_MIGRATE SDK=!@#$= somescript.py",
		lineNum: 0,
	}
	testRunner := setupRunner(migrateNode)

	err := evalSvcMigrate(testRunner.runnerObj, testRunner.testNode)

	// We can't use ErrorMatches because it wants to do some kind of regex on our regex
	t.Assert(err, NotNil)
	t.Assert(err.Error(), Equals, "arg SDK=!@#$= did not match ^SDK=([a-zA-Z0-9.\\-_]+)$")
}

type EvalTestRunner struct {
	mock.Mock
	runnerObj *runner
	testNode  node
}

func (etr *EvalTestRunner) svcMigrate(serviceID string, scriptBody string, sdkVersion string) error {
	return etr.Mock.Called(serviceID, scriptBody, sdkVersion).Error(0)
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
		cmd:     "SVC_MIGRATE somescript.py",
		args:    []string{"somescript.py"},
		line:    "SVC_MIGRATE somescript.py",
		lineNum: 0,
	}
	return migrateNode
}

func setupMigrateNodeWithSDKVer() node {
	migrateNode := node{
		cmd:     "SVC_MIGRATE SDK=1.2.3 somescript.py",
		args:    []string{"SDK=1.2.3", "somescript.py"},
		line:    "SVC_MIGRATE SDK=1.2.3 somescript.py",
		lineNum: 0,
	}
	return migrateNode
}
