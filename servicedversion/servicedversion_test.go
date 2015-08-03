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

package servicedversion

import (
	"reflect"
	"testing"

	"github.com/control-center/serviced/utils"
)

func TestGetPackageRelease(t *testing.T) {
	t.Log("verifying command to get package release")

	actual := getCommandToGetPackageRelease("serviced")

	expected := []string{}
	if utils.Platform == utils.Rhel {
		expected = []string{"bash", "-c", "rpm -q --qf '%{VERSION}-%{Release}\n' serviced"}
	} else {
		expected = []string{"bash", "-o", "pipefail", "-c", "dpkg -s serviced | awk '/^Version/{print $NF;exit}'"}
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected: %+v != actual: %+v", expected, actual)
	}
}
