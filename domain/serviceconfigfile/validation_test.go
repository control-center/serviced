// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package serviceconfigfile

import (
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

type testcase struct {
	svcConfFile SvcConfigFile
	valid       bool
}

var testcases = []testcase{
	testcase{
		svcConfFile: SvcConfigFile{},
		valid:       false,
	},
	testcase{
		svcConfFile: SvcConfigFile{
			ID: "id",
		},
		valid: false,
	},
	testcase{
		svcConfFile: SvcConfigFile{
			ID:              "id",
			ServiceTenantID: "tenant",
		},
		valid: false,
	},
	testcase{
		svcConfFile: SvcConfigFile{
			ID:              "id",
			ServiceTenantID: "tenant",
			ServicePath:     "/path",
		},
		valid: false,
	},
	testcase{
		svcConfFile: SvcConfigFile{
			ID:              "id",
			ServiceTenantID: "tenant",
			ServicePath:     "/path",
			ConfFile:        servicedefinition.ConfigFile{Content: "Test content"},
		},
		valid: false,
	},
	testcase{
		svcConfFile: SvcConfigFile{
			ID:              "id",
			ServiceTenantID: "tenant",
			ServicePath:     "badpath",
			ConfFile:        servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"},
		},
		valid: false,
	},
	testcase{
		svcConfFile: SvcConfigFile{
			ID:              "id",
			ServiceTenantID: "tenant",
			ServicePath:     "/path",
			ConfFile:        servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"},
		},
		valid: true,
	},
}

func (s *S) Test_Validation(t *C) {
	for idx, tc := range testcases {
		err := tc.svcConfFile.ValidEntity()
		if tc.valid && err != nil {
			t.Errorf("expected valid testcase for %v, got: %v", idx, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("expected invalid testcase for %v, got success", idx)
		}
	}

}
