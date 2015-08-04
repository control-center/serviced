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

package retry

import (
	"testing"
	"time"
)

func TestNTimes(t *testing.T) {

	n := NTimes(0, time.Millisecond*10)
	if retry, _ := n.AllowRetry(0, time.Second); retry {
		t.Logf("expected 0 retries")
		t.FailNow()
	}
	n = NTimes(1, time.Millisecond*10)
	if retry, _ := n.AllowRetry(0, time.Second); !retry {
		t.Logf("expected 1 retries")
		t.FailNow()
	}

	// check if elapsed means anything
	if retry, _ := n.AllowRetry(0, time.Second*10000); !retry {
		t.Logf("expected 1 retries")
		t.FailNow()
	}
}
