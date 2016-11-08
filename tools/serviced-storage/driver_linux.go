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

package main

import (
// Need to do btrfs driver initializations
	_ "github.com/control-center/serviced/volume/btrfs"
// Need to do rsync driver initializations
	_ "github.com/control-center/serviced/volume/rsync"
// Need to do devicemapper driver initializations
	_ "github.com/control-center/serviced/volume/devicemapper"
)

// Execute displays orphaned volumes
func (c *CheckOrphans) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Args.Path))
	dmd, err := InitDriverIfExists(directory)
	if err != nil {
		return err
	}
	drv, ok := dmd.(*devicemapper.DeviceMapperDriver)
	if !ok {
		log.Fatal("This only works on devicemapper devices")
	}

	//  we retrieve the list of all devices that CC has access to
	var ccDevices []string
	for _, v := range drv.ListTenants() {
		_vol, err := drv.Get(v)
		if err != nil {
			return err
		}
		vol, ok := _vol.(*devicemapper.DeviceMapperVolume)
		if !ok {
			log.Fatal("Volume", v, "is not of DeviceMapper type.")
		}
		ccDevices = append(ccDevices, vol.Metadata.CurrentDevice())
		for _, dHash := range vol.Metadata.Snapshots {
			ccDevices = append(ccDevices, dHash)
		}
	}

	// next we compare CC-visible device hashes to the device hashes stored by the device driver
	var orphans []string
	for _, d := range drv.DeviceSet.List() {
		found := false
		for _, c := range ccDevices {
			// (*devicemapper.DeviceMapperDriver).DeviceSet.List() will contain an empty string for the base device
			if d == c || d == "" {
				found = true
				break
			}
		}
		if !found {
			orphans = append(orphans, d)
		}
	}

	if len(orphans) > 0 {
		fmt.Println("Orphaned devices were found")
		// delete the actual devices
		for _, v := range orphans {
			if c.Clean {
				drv.DeviceSet.UnmountDevice(v, drv.Root())
				drv.DeviceSet.Lock()
				if err := drv.DeactivateDevice(v); err != nil {
					log.Info("An error occurred while attempting to deactivate the device:", err)
				}
				drv.DeviceSet.Unlock()
				if err := drv.DeviceSet.DeleteDevice(v, false); err != nil {
					log.Info("An error occurred while attempting to remove the device:", err)
				}
				fmt.Println("Removed orphaned snapshot", v)
			} else {
				fmt.Println(v)
			}
		}
	} else {
		fmt.Println("No orphaned devices found.")
	}
	return nil
}
