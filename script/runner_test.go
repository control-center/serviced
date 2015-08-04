// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// +build unit

package script

import (
	"errors"

	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_Run(t *C) {
	stopChan := make(chan struct{})
	config := Config{
		NoOp:          true,
		ServiceID:     "TEST_SERVICE_ID_12345",
		TenantLookup:  func(service string) (string, error) { return service, nil },
		SvcIDFromPath: func(tenantID string, path string) (string, error) { return tenantID, nil },
	}
	runner, err := NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	err = runner.Run(stopChan)
	t.Assert(err, IsNil)

	// Test sending a stop signal to the runner
	runner, err = NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	newStopChan := make(chan struct{})
	close(newStopChan)
	err = runner.Run(newStopChan)
	t.Assert(err, NotNil)

	runner, err = NewRunnerFromFile("bad_descriptor.txt", &config)
	t.Assert(err, ErrorMatches, "error parsing script")

	config = Config{
		NoOp:          true,
		ServiceID:     "TEST_SERVICE_ID_12345",
		TenantLookup:  func(service string) (string, error) { return service, nil },
		SvcIDFromPath: func(tenant, path string) (string, error) { return "", errors.New("test error id from path") },
	}
	runner, err = NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	err = runner.Run(stopChan)
	t.Assert(err, ErrorMatches, "test error id from path")

}
