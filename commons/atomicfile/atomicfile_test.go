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
)

func TestWriteFile(t *testing.T) {

	f, err := ioutil.TempFile("", "TestWriteFile")
	if err != nil {
		t.Fatalf("unexpected error creating tempfile: %s", err)
	}
	defer os.Remove(f.Name())
	if err := f.Close(); err != nil {
		t.Fatalf("error closing tempfile")
	}

	expectedBytes := []byte("foobar")
	if err := WriteFile(f.Name(), expectedBytes, 0660); err != nil {
		t.Fatalf("unexpected error writing to atomic file: %s", err)
	}

	data, err := ioutil.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("trouble reading tempfile: %s", err)
	}
	if !reflect.DeepEqual(data, expectedBytes) {
		t.Fatalf("got %+v expected %+v", data, expectedBytes)
	}
	stats, err := os.Stat(f.Name())
	if err != nil {
		t.Fatalf("error getting stats on file %s: %s", f.Name(), err)
	}
	newMode := stats.Mode()
	if 0660 != newMode {
		t.Fatalf("desired file mode (%04o) not successfully found (%04o)", 0660, newMode)
	}
}
