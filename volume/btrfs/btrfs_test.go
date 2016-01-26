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
	"path/filepath"
	"sort"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/drivertest"
	// Register the btrfs driver
	. "github.com/control-center/serviced/volume/btrfs"
)

var (
	_                  = Suite(&BtrfsSuite{})
	btrfsArgs []string = []string{}
)

func arrayContains(array []string, element string) bool {
	for _, x := range array {
		if x == element {
			return true
		}
	}
	return false
}

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
	volume.CleanupTmpVolume(c, s.root)
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

func (s *BtrfsSuite) TestBtrfsBadSnapshots(c *C) {
	err := volume.InitDriver("btrfs", s.root, btrfsArgs)
	c.Assert(err, IsNil)
	d, err := volume.GetDriver(s.root)
	c.Assert(err, IsNil)
	c.Assert(d, NotNil)

	vol, err := d.Create("Base")
	c.Assert(err, IsNil)
	c.Assert(vol, NotNil)

	//create a subvolume that looks like a snapshot but is missing .SnapshotInfo
	badSnapshotPath := filepath.Join(s.root, "Base_badsnapshot")
	_, err = volume.RunBtrFSCmd(true, "subvolume", "snapshot", "-r", vol.Path(), badSnapshotPath)
	c.Assert(err, IsNil)

	// Make sure we can still list snapshots
	snaps, err := vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(len(snaps), Equals, 0)

	// Make sure we can still add another snapshot
	err = vol.Snapshot("Snap", "snapshot-message-0", []string{"SnapTag", "tagA"})
	c.Assert(err, IsNil)

	// And it shows up in the list
	snaps, err = vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(len(snaps), Equals, 1)
	c.Assert(arrayContains(snaps, "Base_Snap"), Equals, true)

	// Trying to get info on the first snapshot fails
	snapInfo, err := vol.SnapshotInfo("Base_badsnapshot")
	c.Assert(err, NotNil)
	c.Assert(snapInfo, IsNil)

	// Trying to get info on the second snapshot works
	snapInfo, err = vol.SnapshotInfo("Base_Snap")
	c.Assert(err, IsNil)
	c.Assert(snapInfo, NotNil)

}

func (s *BtrfsSuite) TestBtrfsSnapshotTags(c *C) {
	err := volume.InitDriver("btrfs", s.root, btrfsArgs)
	c.Assert(err, IsNil)
	d, err := volume.GetDriver(s.root)
	c.Assert(err, IsNil)
	c.Assert(d, NotNil)

	vol, err := d.Create("Base")
	c.Assert(err, IsNil)
	c.Assert(vol, NotNil)

	// Take a snapshot with tags
	err = vol.Snapshot("Snap", "snapshot-message-0", []string{"SnapTag", "tagA"})
	c.Assert(err, IsNil)
	snaps, err := vol.Snapshots()
	c.Assert(err, IsNil)
	sort.Strings(snaps)
	c.Assert(sort.SearchStrings(snaps, "Base_Snap") < len(snaps), Equals, true)

	// Verify the tags are set
	info, err := vol.SnapshotInfo("Base_Snap")
	c.Assert(err, IsNil)
	c.Assert(info, NotNil)
	c.Check(info.Name, Equals, "Base_Snap")
	c.Check(info.Label, Equals, "Snap")
	c.Check(info.TenantID, Equals, "Base")
	c.Check(info.Message, Equals, "snapshot-message-0")
	c.Check(info.Tags, DeepEquals, []string{"SnapTag", "tagA"})

	// Take another snapshot with an existing tag
	err = vol.Snapshot("Snap2", "snapshot-message-1", []string{"tagA"})
	c.Assert(err, Equals, volume.ErrTagAlreadyExists)

	// Add a tag to an existing snapshot
	err = vol.TagSnapshot("Base_Snap", "tagB")
	c.Assert(err, Equals, ErrBtrfsNotSupported)

	// Remove a tag from an existing snapshot
	label, err := vol.UntagSnapshot("tagA")
	c.Assert(err, Equals, ErrBtrfsNotSupported)
	c.Assert(label, Equals, "")

	c.Assert(d.Remove("Base"), IsNil)
	c.Assert(d.Exists("Base"), Equals, false)
}

func (s *BtrfsSuite) TestBtrfsExportImport(c *C) {
	other_root := volume.CreateBtrfsTmpVolume(c, 32*1024*1024)
	defer volume.CleanupTmpVolume(c, other_root)
	drivertest.DriverTestExportImport(c, "btrfs", s.root, other_root, btrfsArgs)
}
