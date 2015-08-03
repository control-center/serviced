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

package utils

import (
	"errors"
	"os/exec"
	"testing"
)

func TestGetExitStatus(t *testing.T) {
	t.Log("verifying nil error")
	if exitstatus, ok := GetExitStatus(nil); !ok {
		t.Errorf("nil exit status should be considered a valid exit status")
	} else if exitstatus != 0 {
		t.Errorf("Expected 0 exit code; actual %d", exitstatus)
	}

	t.Log("verifying invalid error type")
	invalidError := errors.New("not an ExitError")
	if _, ok := GetExitStatus(invalidError); ok {
		t.Errorf("error `%s` is not an ExitError", invalidError)
	}

	t.Log("verifying valid errors")
	exiterrors := map[int]error{
		0: exec.Command("/bin/bash", "-c", "exit 0").Run(),
		1: exec.Command("/bin/bash", "-c", "exit 1").Run(),
	}

	for expected, err := range exiterrors {
		if actual, ok := GetExitStatus(err); !ok {
			t.Errorf("error '%s' should be considered a valid exit error", err)
		} else if expected != actual {
			t.Errorf("Expected %d; Actual: %d", expected, actual)
		}
	}
}
