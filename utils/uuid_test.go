// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
