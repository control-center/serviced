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

// +build root,integration

package btrfs_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/drivertest"
	// Register the btrfs driver
	_ "github.com/control-center/serviced/volume/btrfs"
)

var (
	_                  = Suite(&BtrfsSuite{})
	btrfsArgs []string = []string{}
)

// Wire in gocheck
func Test(t *testing.T) { TestingT(t) }

type BtrfsSuite struct {
	root string
}

func (s *BtrfsSuite) SetUpSuite(c *C) {
	root := volume.CreateBtrfsTmpVolume(c, 32*1024*1024)
	s.root = root
}

func (s *BtrfsSuite) TearDownSuite(c *C) {
	volume.CleanupBtrfsTmpVolume(c, s.root)
}

func (s *BtrfsSuite) TestBtrfsCreateEmpty(c *C) {
	drivertest.DriverTestCreateEmpty(c, "btrfs", s.root, btrfsArgs)
}

func (s *BtrfsSuite) TestBtrfsCreateBase(c *C) {
	drivertest.DriverTestCreateBase(c, "btrfs", s.root, btrfsArgs)
}

func (s *BtrfsSuite) TestBtrfsSnapshots(c *C) {
	drivertest.DriverTestSnapshots(c, "btrfs", s.root, btrfsArgs)
}

func (s *BtrfsSuite) TestBtrfsExportImport(c *C) {
	other_root := volume.CreateBtrfsTmpVolume(c, 32*1024*1024)
	defer volume.CleanupBtrfsTmpVolume(c, other_root)
	drivertest.DriverTestExportImport(c, "btrfs", s.root, other_root, btrfsArgs)
}
