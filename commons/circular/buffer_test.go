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

package circular

import (
	"testing"
	
	. "gopkg.in/check.v1"
)

func TestCircular(t *testing.T) { TestingT(t) }

type CircularSuite struct{}

var _ = Suite(&CircularSuite{})

// TODO: Write a performance benchmark to show improvements to impl.
// TODO: Write test to read from empty buffer, read from full buffer, etc

func (s *CircularSuite) TestBuffer(c *C) {

	const circularBufferSize = 5
	b := NewBuffer(circularBufferSize)

	testbytes := []byte{99, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	if n, err := b.Write(testbytes); err != nil {
		c.Logf("Unexpected error when writing to circular buffer: %s", err)
		c.FailNow()
	} else {
		if n != len(testbytes) {
			c.Logf("expected %d bytes written, only %d were written", len(testbytes), n)
			c.FailNow()
		}
	}

	results := make([]byte, circularBufferSize)

	if n, err := b.Read(results); err != nil {
		c.Logf("Unexpected error when reading from circular buffer: %s", err)
		c.FailNow()
	} else {
		if n != circularBufferSize {
			c.Logf("expected %d bytes read, only %d were read", circularBufferSize, n)
			c.Logf("buffer: %v", b)
			c.Logf("results: %v", results)
			c.FailNow()
		}
	}
}
