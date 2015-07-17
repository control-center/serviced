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

package subprocess

import (
	"testing"
	"time"
)

func TestSubprocess(t *testing.T) {
	s, exited, err := New(time.Second*5, nil, "sleep", "1")
	if err != nil {
		t.Fatalf("expected subprocess to start: %s", err)
	}
	select {
	case <-time.After(time.Millisecond * 1200):
		t.Fatal("expected sleep to finish")
	case <-exited:

	}

	timeout := time.AfterFunc(time.Millisecond*500, func() {
		t.Fatal("Should have closed subprocess already!")
	})
	s.Close()
	timeout.Stop()
}
