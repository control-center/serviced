// Copyright 2015 The Serviced Authors.
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
	"testing"
)

func TestBase62(t *testing.T) {
	if Base62(238327) != "ZZZ" {
		t.Errorf("238327 in base 62 should be ZZZ.")
	}
	if Base62(0) != "0" {
		t.Errorf("0 in base 62 should be 0.")
	}
	if Base62(10) != "a" {
		t.Errorf("10 in base 62 should be a.")
	}
}
