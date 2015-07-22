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

package devicemapper_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/volume/drivertest"
	// Register the btrfs driver
	_ "github.com/control-center/serviced/volume/devicemapper"
)

var (
	_ = Suite(&DeviceMapperSuite{})
)

func Test(t *testing.T) { TestingT(t) }

type DeviceMapperSuite struct{}

func (s *DeviceMapperSuite) TestDeviceMapperCreateEmpty(c *C) {
	drivertest.DriverTestCreateEmpty(c, "devicemapper", "")
}

func (s *DeviceMapperSuite) TestDeviceMapperCreateBase(c *C) {
	drivertest.DriverTestCreateBase(c, "devicemapper", "")
}

func (s *DeviceMapperSuite) TestDeviceMapperSnapshots(c *C) {
	drivertest.DriverTestSnapshots(c, "devicemapper", "")
}

func (s *DeviceMapperSuite) TestDeviceMapperExportImport(c *C) {
	drivertest.DriverTestExportImport(c, "devicemapper", "", "")
}
