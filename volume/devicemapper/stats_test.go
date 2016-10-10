// Copyright 2016 The Serviced Authors.
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

package devicemapper

import (
  "os"
  "testing"
	. "gopkg.in/check.v1"
)

func TestStats(t *testing.T) { TestingT(t) }

type StatsSuite struct{}

var _ = Suite(&StatsSuite{})

// TestParseDumpe2fsOutput tests that sample dumpe2fs output parses correctly
func (s *StatsSuite) TestParseDumpe2fsOutput(c *C) {
	r, err := os.Open("testdata/dumpe2fs.out.data")
	if err != nil {
		panic(err)
	}
	defer r.Close()
	fs, err := parseDumpe2fsOutput(r)
	if err != nil {
		panic(err)
	}

  // Compare to hand picked values
	c.Check(fs, DeepEquals, &filesystemStats{
		BlocksTotal: 26214400,
		BlockSize:       4096,
		FreeBlocks:      25755051,
		UnusableBlocks: 444088,
		Superblocks: 15,
		GroupsTotal: 800,
		JournalLength: 32768,
		SuperblockSize: 1,
		GroupDescSize: 7,
		BitmapSize: 1,
		InodeBitmapSize: 1,
		InodeTableSize: 512,
	})
}

// TestParseDfOutput tests that sample df output parses correctly
func (s *StatsSuite) TestParseDfOutput(c *C) {
  r, err := os.Open("testdata/df.out.data")
	if err != nil {
		panic(err)
	}
	defer r.Close()
	dfSlice, err := parseDfOutput(r)
	if err != nil {
		panic(err)
	}

  // Our test data only has 1 element, compare to hand picked values
	c.Check(dfSlice[0], DeepEquals, &dfStats{
    BlockSize: 1024,
    FilesystemPath: "/dev/mapper/docker-8:1-12852375-1VbuL3AP0QC0ETsvpys9su",
    BlocksTotal: 103081248,
    BlocksUsed: 537608,
    BlocksAvailable: 97284376,
    MountPoint: "/exports/serviced_volumes_v2/e0fqzrnmnwgiytbznlqi0fy08",
  })
}

// TestConsistantStats tests that parseDumpe2fsOutput and parseDfOutput
// give us the same usable fs size
func (s *StatsSuite) TestConsistantStats(c *C) {
	rDe, err := os.Open("testdata/dumpe2fs.out.data")
	if err != nil {
		panic(err)
	}
	defer rDe.Close()
	rDf, err := os.Open("testdata/df.out.data")
	if err != nil {
		panic(err)
	}
	defer rDf.Close()
	fs, err := parseDumpe2fsOutput(rDe)
	if err != nil {
		panic(err)
	}
	dfSlice, err := parseDfOutput(rDf)
	if err != nil {
		panic(err)
	}
	sizeDe := (fs.BlocksTotal - fs.UnusableBlocks) * fs.BlockSize
	sizeDf := dfSlice[0].BlocksTotal * dfSlice[0].BlockSize
	c.Check(sizeDe, DeepEquals, sizeDf)
}
