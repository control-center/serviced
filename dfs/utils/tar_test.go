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
	"archive/tar"
	"fmt"
	"os"
	"os/exec"
	"testing"

	. "github.com/control-center/serviced/dfs/utils"
	. "gopkg.in/check.v1"
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

func TestTar(t *testing.T) { TestingT(t) }

func tarfile(dir, size string, numfiles int) string {
	i += 1
	tardir := fmt.Sprintf("%s/test-%d", dir, i)
	os.Mkdir(tardir, os.ModePerm)
	for j := 0; j < numfiles; j++ {
		fname := getFileName(tardir)
		cmd := fmt.Sprintf("cat /dev/urandom | head -c%s > %s", size, fname)
		exec.Command("/bin/bash", "-c", cmd).Run()
	}
	tar := fmt.Sprintf("%s.tar", tardir)
	exec.Command("tar", "cf", tar, tardir).Run()
	return tar
}

func countFilesInTar(fname string) int {
	tarfile, _ := os.Open(fname)
	defer tarfile.Close()
	reader := tar.NewReader(tarfile)
	var j int
	for {
		if _, err := reader.Next(); err != nil {
			fmt.Println(err)
			break
		}
		j += 1
	}
	return j
}

func (s *TarTestSuite) TestSingleTar(c *C) {
	dir := c.MkDir()
	testfile := tarfile(dir, "100KB", 2)
	instream, _ := os.Open(testfile)
	outfile := fmt.Sprintf("%s/out.tar", dir)
	outstream, _ := os.Create(outfile)

	m := NewTarStreamMerger(outstream)
	m.Append(instream)
	m.Close()
	outstream.Close()

	c.Assert(countFilesInTar(outfile), Equals, 2)

}
