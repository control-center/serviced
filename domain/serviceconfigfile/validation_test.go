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

// +build integration

package serviceconfigfile

import (
	"github.com/control-center/serviced/domain/servicedefinition"
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
