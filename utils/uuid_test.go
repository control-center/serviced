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
	"os"
	"testing"
)

type testRandT struct{}

func (t testRandT) Read(p []byte) (n int, err error) {
	f, err := os.Open("../testfiles/urandom_bytes")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Read(p)
}

// Test newUuid()
func TestNewUUID(t *testing.T) {
	randSource = testRandT{}

	uuid, err := NewUUID()
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		t.Fail()
	}
	expectedUuid := "1102c395-e94b-0a08-d1e9-307e31a5213e"
	if uuid != expectedUuid {
		t.Errorf("uuid: expected %s, got %s", expectedUuid, uuid)
		t.Fail()
	}
}

// Test newUuid62()
func TestNewUUID62(t *testing.T) {
	randSource = testRandT{}

	uuid, err := NewUUID62()
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		t.Fail()
	}
	expectedUuid := "w68e9g0vxRwx0gA9lQBxs"
	if uuid != expectedUuid {
		t.Errorf("uuid: expected %s, got %s", expectedUuid, uuid)
		t.Fail()
	}
}

func TestConvertUp(t *testing.T) {
	orig := "123456789abcdef"
	conv := ConvertUp(orig, "0123456789abcdef")

	if orig != conv {
		t.Errorf("got %s, expected %s", orig, conv)
	}
}
