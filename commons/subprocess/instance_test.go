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
	"time"

	. "gopkg.in/check.v1"
)

func (s *TestSubprocessSuite) TestSubprocess(c *C) {
	subprocess, exited, err := New(time.Second*5, nil, "sleep", "1")
	c.Assert(err, IsNil)

	select {
	case <-time.After(time.Millisecond * 1200):
		c.Fatal("expected sleep to finish")
	case <-exited:

	}

	timeout := time.AfterFunc(time.Millisecond*500, func() {
		c.Fatal("Should have closed subprocess already!")
	})
	subprocess.Close()
	timeout.Stop()
}
