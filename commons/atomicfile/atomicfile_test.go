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

package atomicfile

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	. "gopkg.in/check.v1"
)

func TestAtomicFile(t *testing.T) { TestingT(t) }

type AtomicFileSuite struct{}

var _ = Suite(&AtomicFileSuite{})

func (s *AtomicFileSuite) TestWriteFile(c *C) {
	f, err := ioutil.TempFile("", "TestWriteFile")
	if err != nil {
		c.Fatalf("unexpected error creating tempfile: %s", err)
	}
	defer os.Remove(f.Name())
	if err := f.Close(); err != nil {
		c.Fatalf("error closing tempfile")
	}

	expectedBytes := []byte("foobar")
	if err := WriteFile(f.Name(), expectedBytes, 0660); err != nil {
		c.Fatalf("unexpected error writing to atomic file: %s", err)
	}

	data, err := ioutil.ReadFile(f.Name())
	if err != nil {
		c.Fatalf("trouble reading tempfile: %s", err)
	}
	if !reflect.DeepEqual(data, expectedBytes) {
		c.Fatalf("got %+v expected %+v", data, expectedBytes)
	}
	stats, err := os.Stat(f.Name())
	if err != nil {
		c.Fatalf("error getting stats on file %s: %s", f.Name(), err)
	}
	newMode := stats.Mode()
	if 0660 != newMode {
		c.Fatalf("desired file mode (%04o) not successfully found (%04o)", 0660, newMode)
	}
}
