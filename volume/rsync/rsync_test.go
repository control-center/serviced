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

// +build root,integration

package rsync_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/drivertest"
	// Register the rsync driver
	_ "github.com/control-center/serviced/volume/rsync"
)

var (
	rsyncArgs []string = make([]string, 0)
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

type RsyncSuite struct{}

var _ = Suite(&RsyncSuite{})

func (s *RsyncSuite) TestRsyncCreateEmpty(c *C) {
	drivertest.DriverTestCreateEmpty(c, "rsync", "", rsyncArgs)
}

func (s *RsyncSuite) TestRsyncCreateBase(c *C) {
	drivertest.DriverTestCreateBase(c, "rsync", "", rsyncArgs)
}

func (s *RsyncSuite) TestRsyncSnapshots(c *C) {
	drivertest.DriverTestSnapshots(c, "rsync", "", rsyncArgs)
}

func (s *RsyncSuite) TestRsyncSnapshotTags(c *C) {
	drivertest.DriverTestSnapshotTags(c, "rsync", "", rsyncArgs)
}

func (s *RsyncSuite) TestRsyncExportImport(c *C) {
	drivertest.DriverTestExportImport(c, "rsync", "", "", rsyncArgs)
}

func (s *RsyncSuite) TestRsyncBadSnapshots(c *C) {
	badsnapshot := func(label string, vol volume.Volume) error {
		//create an invalid snapshot by snapshotting and then removing .SnapshotInfo
		if err := vol.Snapshot(label, "", []string{}); err != nil {
			return err
		}
		filePath := filepath.Join(vol.Driver().Root(), ".rsync", "volumes", fmt.Sprintf("%s_%s", vol.Name(), label), ".SNAPSHOTINFO")
		err := os.Remove(filePath)
		return err
	}

	drivertest.DriverTestBadSnapshot(c, "rsync", "", badsnapshot, rsyncArgs)
}
