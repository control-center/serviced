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

package api

import (
	"fmt"
	"strings"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
)

func getDefaultStorageOptions(driverType volume.DriverType, config utils.ConfigReader) []string {
	var options []string
	switch driverType {
	case volume.DriverTypeRsync:
	case volume.DriverTypeBtrFS:
	case volume.DriverTypeDeviceMapper:
		addStorageOption(config, "DM_THINPOOLDEV", "", func(v string) {
			options = append(options, fmt.Sprintf("dm.thinpooldev=%s", v))
		})
		addStorageOption(config, "DM_BASESIZE", "100G", func(v string) {
			options = append(options, fmt.Sprintf("dm.basesize=%s", v))
		})
		addStorageOption(config, "DM_LOOPDATASIZE", "", func(v string) {
			options = append(options, fmt.Sprintf("dm.loopdatasize=%s", v))
		})
		addStorageOption(config, "DM_LOOPMETADATASIZE", "", func(v string) {
			options = append(options, fmt.Sprintf("dm.loopmetadatasize=%s", v))
		})
		addStorageOption(config, "DM_ARGS", "", func(v string) {
			options = append(options, strings.Split(v, " ")...)
		})
	}
	return options
}

func addStorageOption(config utils.ConfigReader, key, dflt string, parse func(v string)) {
	if v := strings.TrimSpace(config.StringVal(key, dflt)); v != "" {
		parse(v)
	}
}
