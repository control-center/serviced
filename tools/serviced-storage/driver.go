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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/control-center/serviced/volume"
	"github.com/jessevdk/go-flags"

	log "github.com/Sirupsen/logrus"
)

// DriverInit is the subcommand for initializing a new driver
type DriverInit struct {
	Args struct {
		Path flags.Filename `description:"Path of the driver"`
		Type string         `description:"Type of driver to initialize (btrfs|devicemapper|rsync)"`
	} `positional-args:"yes" required:"yes"`
}

// DriverSet is the subcommand for setting the default driver
type DriverSet struct {
	Args struct {
		Path flags.Filename `description:"Path of the driver"`
	} `positional-args:"yes" required:"yes"`
}

// DriverUnset is the subcommand for unsetting the default driver
type DriverUnset struct{}

// DriverDisable is the subcommand to disable the driver
type DriverDisable struct {
	Args struct {
		Path flags.Filename `description:"Path of the driver"`
	} `positional-args:"yes" optional:"yes"`
}

// DriverStatus is the subcommand for displaying the status of the driver
type DriverStatus struct {
	Args struct {
		Path flags.Filename `description:"Path of the driver"`
	} `positional-args:"yes" optional:"yes"`
}

// DriverList is the subcommand for listing the volumes for a driver
type DriverList struct {
	Args struct {
		Path flags.Filename `description:"Path of the driver"`
	} `positional-args:"yes" optional:"yes"`
}

// CheckOrphans is the subcommand for displaying orphaned volumes
type CheckOrphans struct {
	Clean bool `description:"Indicates that the orphans should be removed" long:"clean" short:"c"`
	Args  struct {
		Path flags.Filename `long:"driver" short:"d" description:"Path of the driver"`
	} `positional-args:"yes"`
}

// Execute initializes a new storage driver
func (c *DriverInit) Execute(args []string) error {
	App.initializeLogging()
	driverType, err := volume.StringToDriverType(c.Args.Type)
	if err != nil {
		log.Fatal(err)
	}
	path := string(c.Args.Path)
	logger := log.WithFields(log.Fields{
		"directory": path,
		"type":      driverType,
	})
	logger.Info("Initializing storage driver")
	if err := volume.InitDriver(driverType, path, App.Options.Options); err != nil {
		log.Fatal(err)
	}
	logger.Info("Storage driver initialized successfully")
	return nil
}

// Execute sets the default driver for all proceeding commands
func (c *DriverSet) Execute(args []string) error {
	App.initializeLogging()
	path := string(c.Args.Path)
	if _, err := InitDriverIfExists(path); err == volume.ErrDriverNotInit {
		log.Fatalf("Driver not initialized. Use `%s driver init %s TYPE [OPTIONS]`", "."+App.name, path)
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(usr.HomeDir, App.name), []byte(path), 0644); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Execute unsets the default driver
func (c *DriverUnset) Execute(args []string) error {
	App.initializeLogging()
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(usr.HomeDir, App.name)); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
	return nil
}

// Execute disables a driver
func (c *DriverDisable) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Args.Path))
	logger := log.WithFields(log.Fields{
		"directory": directory,
	})
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger = logger.WithFields(log.Fields{
		"type": driver.DriverType(),
	})
	if err := driver.Cleanup(); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Disabled driver")
	return nil
}

// Execute displays the status of a driver
func (c *DriverStatus) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Args.Path))
	logger := log.WithFields(log.Fields{
		"directory": directory,
	})
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		logger.Fatal(err)
	}
	logger = logger.WithFields(log.Fields{
		"type": driver.DriverType(),
	})
	status, err := driver.Status()
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println(directory)
	fmt.Println(status.String())
	return nil
}

// Execute displays the volumes on a driver
func (c *DriverList) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Args.Path))
	logger := log.WithFields(log.Fields{
		"directory": directory,
	})
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		logger.Fatal(err)
	}
	for _, volname := range driver.List() {
		fmt.Println(volname)
	}
	return nil
}

// Execute displays orphaned volumes
func (c *CheckOrphans) Execute(args []string) error {
	App.initializeLogging()
	type Metadata struct {
		CurrentDevice string
		Snapshots     interface{}
	}

	directory := GetDefaultDriver(string(c.Args.Path))
	volDir := directory + "/.devicemapper/volumes"
	fsVolDir := directory + "/.devicemapper/metadata"

	files, err := ioutil.ReadDir(volDir)
	if err != nil {
		return err
	}

	var ccVols []string
	var snaps []string
	var orphans []string
	for _, f := range files {
		if strings.Contains(f.Name(), ".") {
			snaps = append(snaps, f.Name())
			orphans = append(orphans, f.Name())
		} else {
			ccVols = append(ccVols, f.Name())
		}
	}

	// This will store CC-visible device identifiers for comparing with the filesystem's metadata
	var known []string
	for _, v := range ccVols {
		dat, _ := ioutil.ReadFile(volDir + "/" + v + "/metadata.json")
		var m Metadata
		json.Unmarshal(dat, &m)
		known = append(known, m.CurrentDevice)
		for k, vv := range m.Snapshots.(map[string]interface{}) {
			known = append(known, vv.(string))
			// Check if this snapshot is in our metadata
			for i, s := range orphans {
				if s == k {
					orphans = append(orphans[:i], orphans[i+1:]...)
				}
			}
		}
	}
	if len(orphans) > 0 {
		fmt.Println("Orphans were found:")
		for _, o := range orphans {
			fmt.Println("-", o)
		}
	} else {
		fmt.Println("No orphaned device volumes found")
		return nil
	}

	// now we compare these against the devices on the filesystem metadata to divine the needed "device_id"
	filez, err := ioutil.ReadDir(fsVolDir)
	if err != nil {
		log.Fatal(err)
	}
	// create a clean map of filesystem IDs, each mapped to their respective hash IDs
	var fsVols map[string]float64
	fsVols = map[string]float64{}
	for _, f := range filez {
		d := fsVolDir + "/" + f.Name()
		dat, err := ioutil.ReadFile(d)
		if err != nil {
			return err
		}
		var m map[string]interface{}
		json.Unmarshal(dat, &m)
		for k, v := range m {
			if k == "initialized" && v.(bool) == false {
				fsVols[f.Name()] = m["device_id"].(float64)
			}
		}
	}

	// store only the device ids of the devices we care about
	var zorphans map[string]float64
	zorphans = map[string]float64{}
	for k, v := range fsVols {
		found := false
		for _, vv := range known {
			if k == vv {
				found = true
			}
		}
		if found == false {
			zorphans[k] = v
		}
	}

	if len(zorphans) > 0 {
		for k, v := range zorphans {
			fmt.Println(k, v)
			if c.Clean {
				driver, err := InitDriverIfExists(directory)
				if err != nil {
					log.Fatal(err)
				}
				if err := driver.Remove(k); err != nil {
					log.Fatal(err)
				}
				fmt.Println("Removed orphaned snapshot", k)
			}
		}
	}
	return nil
}

// InitDriverIfExists returns a driver if it has been initialized in the given
// directory.
func InitDriverIfExists(directory string) (volume.Driver, error) {
	log.WithFields(log.Fields{
		"directory": directory,
	}).Debug("Checking driver")
	driverType, err := volume.DetectDriverType(directory)
	if err != nil {
		return nil, err
	}
	logger := log.WithFields(log.Fields{
		"directory": directory,
		"type":      driverType,
	})
	logger.Debug("Found existing storage")
	if err := volume.InitDriver(driverType, directory, App.Options.Options); err != nil {
		return nil, err
	}
	logger.Debug("Loaded storage driver")
	return volume.GetDriver(directory)
}

// GetDefaultDriver returns the path of the default directory as written in the
// state file.
func GetDefaultDriver(path string) string {
	if path != "" {
		return path
	}
	if usr, err := user.Current(); err == nil {
		state, _ := ioutil.ReadFile(filepath.Join(usr.HomeDir, "."+App.name))
		return string(state)
	}
	return ""
}
