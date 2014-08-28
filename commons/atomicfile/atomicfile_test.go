// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
}
