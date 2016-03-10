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

// +build linux,root,integration

package devicemapper_test

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/drivertest"
	// Register the devicemapper driver
	_ "github.com/control-center/serviced/volume/devicemapper"
	"github.com/zenoss/glog"
)

var (
	_                   = Suite(&DeviceMapperSuite{})
	devmapArgs []string = make([]string, 0)
)

func init() {
	// Reduce the size the the base fs and loopback for the tests
	devmapArgs = append(devmapArgs,
		fmt.Sprintf("dm.loopdatasize=%d", 300*1024*1024),
		fmt.Sprintf("dm.loopmetadatasize=%d", 199*1024*1024),
		fmt.Sprintf("dm.basesize=%d", 300*1024*1024),
		"dm.override_udev_sync_check=true")
	if err := initLoopbacks(); err != nil {
		panic(err)
	}
	// Set Docker's logger to debug level, so we can get interesting
	// information if -v
	logrus.SetLevel(logrus.DebugLevel)

	// Also enable glog verbosity so we get other interesting information if tests run with -v
	glog.SetToStderr(true)
	glog.SetVerbosity(2)

}

// getBaseLoopStats inspects /dev/loop0 to collect uid,gid, and mode for the
// loop0 device on the system.  If it does not exist we assume 0,0,0660 for the
// stat data
func getBaseLoopStats() (*syscall.Stat_t, error) {
	loop0, err := os.Stat("/dev/loop0")
	if err != nil {
		if os.IsNotExist(err) {
			return &syscall.Stat_t{
				Uid:  0,
				Gid:  0,
				Mode: 0660,
			}, nil
		}
		return nil, err
	}
	return loop0.Sys().(*syscall.Stat_t), nil
}

// initLoopbacks ensures that the loopback devices are properly created within
// the system running the device mapper tests.
func initLoopbacks() error {
	statT, err := getBaseLoopStats()
	if err != nil {
		return err
	}
	for i := 0; i < 8; i++ {
		loopPath := fmt.Sprintf("/dev/loop%d", i)
		// only create new loopback files if they don't exist
		if _, err := os.Stat(loopPath); err != nil {
			if mkerr := syscall.Mknod(loopPath,
				uint32(statT.Mode|syscall.S_IFBLK), int((7<<8)|(i&0xff)|((i&0xfff00)<<12))); mkerr != nil {
				return mkerr
			}
			os.Chown(loopPath, int(statT.Uid), int(statT.Gid))
		}
	}
	return nil
}

func Test(t *testing.T) { TestingT(t) }

type DeviceMapperSuite struct{}

func (s *DeviceMapperSuite) TestDeviceMapperCreateEmpty(c *C) {
	drivertest.DriverTestCreateEmpty(c, "devicemapper", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestDeviceMapperCreateBase(c *C) {
	drivertest.DriverTestCreateBase(c, "devicemapper", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestDeviceMapperSnapshots(c *C) {
	drivertest.DriverTestSnapshots(c, "devicemapper", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestDeviceMapperSnapshotTags(c *C) {
	drivertest.DriverTestSnapshotTags(c, "devicemapper", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestDeviceMapperExportImport(c *C) {
	drivertest.DriverTestExportImport(c, "devicemapper", "", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestSnapShotContainerMounts(c *C) {
	drivertest.DriverTestSnapshotContainerMounts(c, "devicemapper", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestResize(c *C) {
	drivertest.DriverTestResize(c, "devicemapper", "", devmapArgs)
}

func (s *DeviceMapperSuite) TestDeviceMapperBadSnapshots(c *C) {
	badsnapshot := func(label string, vol volume.Volume) error {
		//create an invalid snapshot by snapshotting and then removing .SnapshotInfo
		if err := vol.Snapshot(label, "", []string{}); err != nil {
			return err
		}
		filePath := filepath.Join(vol.Driver().Root(), ".devicemapper", "volumes", fmt.Sprintf("%s_%s", vol.Name(), label), ".SNAPSHOTINFO")
		err := os.Remove(filePath)
		return err
	}

	drivertest.DriverTestBadSnapshot(c, "devicemapper", "", badsnapshot, devmapArgs)
}
