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

package volume_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	. "github.com/control-center/serviced/volume"
)

func TestIsDir(t *testing.T) {
	root, err := ioutil.TempDir("", "serviced-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	// Test a directory
	if ok, err := IsDir(root); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatalf("%s IS a directory", root)
	}
	// Test a file
	file := filepath.Join(root, "afile")
	ioutil.WriteFile(file, []byte("hi"), 0664)
	if ok, err := IsDir(file); err == nil {
		t.Fatal("IsDir didn't error on a file")
	} else if ok {
		t.Fatalf("IsDir returned true for a file", root)
	}
	// Test a nonexistent path
	if ok, err := IsDir(root + "/notafile"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatalf("IsDir said something existed that didn't", root)
	}

}

func TestFileInfoSlice(t *testing.T) {
	root, err := ioutil.TempDir("", "serviced-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	var (
		slice    FileInfoSlice
		stats    []os.FileInfo
		expected []string
		data     = []byte("hi")
	)

	expected = append(expected, "file1")
	expected = append(expected, "file2")
	expected = append(expected, "file3")

	for _, fname := range expected {
		file := filepath.Join(root, fname)
		ioutil.WriteFile(file, data, 0664)
		if fi, err := os.Stat(file); err == nil {
			stats = append(stats, fi)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Append the FileInfos in non-sorted order
	slice = append(slice, stats[2])
	slice = append(slice, stats[0])
	slice = append(slice, stats[1])

	labels := slice.Labels()
	if !reflect.DeepEqual(labels, expected) {
		t.Fatalf("Labels weren't sorted properly: expected %s but got %s", expected, labels)
	}
}
