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
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
)

// Override to allow loop back devices with devicemapper storage
var allowLoopBack = false

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

func thinPoolEnabled(storageOptions []string) bool {
	enabled := false
	for _, storageArg := range storageOptions {
		if strings.HasPrefix(storageArg, "dm.thinpooldev=") {
			enabled = true
			break
		}
	}
	return enabled
}

func loopBackOptionsFound(storageOptions []string) bool {
	found := false
	for _, storageArg := range storageOptions {
		if strings.HasPrefix(storageArg, "dm.loop") {
			found = true
			break
		}
	}
	return found
}

func validateStorageArgs() error {
	if options.Master {
		log := log.WithFields(logrus.Fields{
			"driver": options.FSType,
			"args":   strings.Join(options.StorageArgs, ","),
		})
		if options.FSType != volume.DriverTypeDeviceMapper {
			log.Warnf("Using a non-recommended storage driver. Recommended configuration is %s with an LVM thin pool", volume.DriverTypeDeviceMapper)
		} else if thinPoolEnabled(options.StorageArgs) {
			if loopBackOptionsFound(options.StorageArgs) {
				log.Warn("Ignoring loop device storage arguments because a thin pool is configured")
			}
		} else {
			allowLoopBack, err := strconv.ParseBool(options.AllowLoopBack)
			if err != nil {
				return fmt.Errorf("error parsing allow-loop-back value %v", err)
			}

			// if we're using devicemapper without a thin pool, and
			if allowLoopBack {
				log.Warn("Use of a loop-device-based thin pool is NOT recommended, but continuing because --allow-loop-back is true")
			} else {
				return fmt.Errorf("Use of devicemapper loop back device is not allowed unless --allow-loop-back=true")
			}
		}
	}
	return nil
}
