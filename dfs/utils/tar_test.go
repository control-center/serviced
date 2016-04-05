// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package utils_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/control-center/serviced/dfs/utils"
	. "gopkg.in/check.v1"
	"gopkg.in/pipe.v2"
)

type TarTestSuite struct{}

var (
	_ = Suite(&TarTestSuite{})
	i int
)

func getFileName(dir string) string {
	i += 1
	return fmt.Sprintf("%s/%d.file", dir, i)
}

func tarfile(dir, size string, numfiles int) string {
	i += 1
	tardir := filepath.Join(dir, fmt.Sprintf("test-%d", i))
	os.Mkdir(tardir, os.ModePerm)
	for j := 0; j < numfiles; j++ {
		fname := getFileName(tardir)
		cmd := fmt.Sprintf("cat /dev/urandom | head -c%s > %s", size, fname)
		exec.Command("/bin/bash", "-c", cmd).Run()
	}
	tar := fmt.Sprintf("%s.tar", tardir)
	exec.Command("tar", "-C", tardir, "-cf", tar, ".").Run()
	return tar
}

func (s *TarTestSuite) TestPrefixPath(c *C) {
	dir := c.MkDir()
	testfile := tarfile(dir, "10KB", 2)
	outfile := filepath.Join(dir, "out.tar")
	p := pipe.Line(
		pipe.ReadFile(testfile),
		PrefixPath("prefix"),
		pipe.WriteFile(outfile, 0644),
	)
	err := pipe.Run(p)
	c.Assert(err, IsNil)
}

func (s *TarTestSuite) TestCatPipe(c *C) {
	p := Cat(
		pipe.Println("line 1"),
		pipe.Line(
			// Add a level of indirection
			pipe.Println("line 2"),
			pipe.Tee(ioutil.Discard),
		),
		pipe.Println("line 3"),
		pipe.Println("line 4"),
		pipe.Println("line 5"),
	)
	out, err := pipe.CombinedOutput(p)
	c.Assert(err, IsNil)
	c.Assert(string(out), Equals, "line 1\nline 2\nline 3\nline 4\nline 5\n")
}

func (s *TarTestSuite) TestConcatTarPipe(c *C) {
	dir := c.MkDir()
	testfile1 := tarfile(dir, "1K", 2)
	testfile2 := tarfile(dir, "1K", 3)
	testfile3 := tarfile(dir, "1K", 1)
	outfile := filepath.Join(dir, "out.tar")

	p := pipe.Line(
		ConcatTarStreams(
			pipe.Line(
				pipe.ReadFile(testfile1),
				PrefixPath("1"),
			),
			pipe.ReadFile(testfile2),
			pipe.ReadFile(testfile3),
		),
		pipe.WriteFile(outfile, 0644),
	)
	err := pipe.Run(p)
	c.Assert(err, IsNil)
}
