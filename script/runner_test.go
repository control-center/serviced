// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"errors"

	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_Run(t *C) {
	config := Config{
		NoOp:          true,
		ServiceID:     "TEST_SERVICE_ID_12345",
		TenantLookup:  func(service string) (string, error) { return service, nil },
		SvcIDFromPath: func(tenantID string, path string) (string, error) { return tenantID, nil },
	}
	runner, err := NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	err = runner.Run()
	t.Assert(err, IsNil)

	runner, err = NewRunnerFromFile("bad_descriptor.txt", &config)
	t.Assert(err, ErrorMatches, "error parsing script")

	config = Config{
		NoOp:      true,
		ServiceID: "TEST_SERVICE_ID_12345",
		TenantLookup: func(service string) (string, error) {
			return "", errors.New("tenant error for test")
		},
	}
	runner, err = NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	err = runner.Run()
	t.Assert(err, ErrorMatches, "tenant error for test")

}
