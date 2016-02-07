// Copyright 2015 The Serviced Authors.
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

package volume_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/control-center/serviced/volume"
	. "gopkg.in/check.v1"
)

func TestUtils(t *testing.T) { TestingT(t) }

type UtilsSuite struct{}

var _ = Suite(&UtilsSuite{})

func (s *UtilsSuite) TestIsDir(c *C) {
	root := c.MkDir()

	// Test a directory
	ok, err := IsDir(root)
	c.Assert(ok, Equals, true)
	c.Assert(err, IsNil)

	// Test a file
	file := filepath.Join(root, "afile")
	ioutil.WriteFile(file, []byte("hi"), 0664)
	ok, err = IsDir(file)
	c.Assert(ok, Equals, false)
	c.Assert(err, ErrorMatches, ErrNotADirectory.Error())

	// Test a nonexistent path
	ok, err = IsDir(root + "/notafile")
	c.Assert(ok, Equals, false)
	c.Assert(err, IsNil)
}

func (s *UtilsSuite) TestFileInfoSlice(c *C) {
	root := c.MkDir()

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
		time.Sleep(sleepTimeMsec * time.Millisecond)
	}

	// Append the FileInfos in non-sorted order
	slice = append(slice, stats[2])
	slice = append(slice, stats[0])
	slice = append(slice, stats[1])

	labels := slice.Labels()

	c.Assert(labels, DeepEquals, expected)
}
