// Copyright 2015 The Serviced Authors.
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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/control-center/serviced/volume"
)

func getDefaultStorageOptions(driverType volume.DriverType) []string {
	var options []string
	switch driverType {
	case volume.DRIVER_RSYNC:
	case volume.DRIVER_BTRFS:
	case volume.DRIVER_DEVICEMAPPER:
		addStorageOption("SERVICED_DM_THINPOOLDEV", func(v string) {
			options = append(options, fmt.Sprintf("dm.thinpooldev=%s", v))
		})
		addStorageOption("SERVICED_DM_BASESIZE", func(v string) {
			options = append(options, fmt.Sprintf("dm.basesize=%s", v))
		})
		addStorageOption("SERVICED_DM_LOOPDATASIZE", func(v string) {
			options = append(options, fmt.Sprintf("dm.loopdatasize=%s", v))
		})
		addStorageOption("SERVICED_DM_LOOPMETADATASIZE", func(v string) {
			options = append(options, fmt.Sprintf("dm.loopmetadatasize=%s", v))
		})
		addStorageOption("SERVICED_DM_ARGS", func(v string) {
			options = append(options, strings.Split(v, " ")...)
		})
	}
	return options
}

func addStorageOption(k string, parse func(v string)) {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		parse(v)
	}
}
