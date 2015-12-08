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
	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"os/exec"
)

// VolumeCreate is the subcommand for creating a new volume on a driver
type VolumeCreate struct {
	Path flags.Filename `long:"driver" short:"d" description:"Path of the driver"`
	Args struct {
		Name string `description:"Name of the volume to create"`
	} `positional-args:"yes" required:"yes"`
}

// VolumeMount is the subcommand for mounting an existing volume from a driver
type VolumeMount struct {
	Path flags.Filename `long:"driver" short:"d" description:"Path of the driver"`
	Args struct {
		Name string `description:"Name of the volume to mount"`
	} `positional-args:"yes" required:"yes"`
}

// VolumeRemove is the subcommand for deleting an existing volume from a driver
type VolumeRemove struct {
	Path flags.Filename `long:"driver" short:"d" description:"Path of the driver"`
	Args struct {
		Name string `description:"Name of the volume to remove"`
	} `positional-args:"yes" required:"yes"`
}
// VolumeSync is the subcommand for syncing two volumes
type VolumeSync struct {
	Args struct {
		DestinationPath flags.Filename	`description:"Path of the destionation" long:"destination-path" short:"p" required:"yes" `
		SourcePath flags.Filename				`description:"Path of the source driver" long:"source" short:"s" required:"yes"`
	}
}
/*
DestinationName string	 				`description:"Destination volume name" long:"destination-name" short:"dn" required:"yes"`
DestinationPath flags.Filename	`description:"Path of the destionation" long:"destination-path" short:"dp" required:"yes" `
SourcePath flags.Filename				`description:"Path of the source driver" long:"source" short:"s" required:"yes"`
Create bool											`description:"Should we create the destination volume" long:"create" short:"c"`

*/


//Execute syncs to volume
func (c *VolumeSync) Execute(args []string) error {
	App.initializeLogging()
	sourceDirectory := GetDefaultDriver(string(c.Args.SourcePath))
	sourceDriver, err := InitDriverIfExists(sourceDirectory)
	if err != nil {
		log.Fatal(err)
	}
	destinationDirectory := GetDefaultDriver(string(c.Args.DestinationPath))
	destinationDriver, err := InitDriverIfExists(destinationDirectory)
	if err != nil {
		log.Fatal(err)
	}
	sourceVolumes := sourceDriver.List()
	for i := 0; i < len(sourceVolumes); i++ {
		volumeName := sourceVolumes[i]
		sourceVolume, err := sourceDriver.Get(volumeName)
		if err != nil {
			log.Fatal(err)
		}
		if destinationDriver.Exists(volumeName) {
			createVolume(string(c.Args.DestinationPath), volumeName)
		}
		rsync(sourceVolume.Path(), string(c.Args.DestinationPath))
	}
	return nil
}

func rsync(sourcePath string, destinationPath string) {
	rsyncBin, err := exec.LookPath("rsync")
	if err != nil {
		log.Fatal(err)
	}
	rsyncArgv := []string{"-a", sourcePath , destinationPath}
	rsync := exec.Command(rsyncBin, rsyncArgv...)
	output, err := rsync.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Ran rysnc and got the following output %s", output)
}

// Execute creates a new volume on a driver
func (c *VolumeCreate) Execute(args []string) error {
	App.initializeLogging()
	createVolume(string(c.Path), c.Args.Name)
	return nil
}

//CreateVolume creates a volume at path with name of name
func createVolume(path string, name string) {
	directory := GetDefaultDriver(path)
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.WithFields(log.Fields {
		"directory": 	driver.Root(),
		"type": 			driver.DriverType(),
		"volume":			name,
	})
	logger.Info("Creating volume")
	vol, err := driver.Create(name)
	if err != nil {
		logger.Fatal(err)
	}
	log.WithFields(log.Fields{
		"mount":	vol.Path(),
	}).Info("Volume Mounted")
}

// Execute mounts an existing volume from a driver
func (c *VolumeMount) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Path))
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.WithFields(log.Fields{
		"directory": driver.Root(),
		"type":      driver.DriverType(),
		"volume":    c.Args.Name,
	})
	logger.Info("Mounting volume")
	vol, err := driver.Get(c.Args.Name)
	if err != nil {
		logger.Fatal(err)
	}
	logger.WithFields(log.Fields{
		"mount": vol.Path(),
	}).Info("Volume mounted")
	return nil
}

// Execute removes an existing volume from a driver
func (c *VolumeRemove) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Path))
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.WithFields(log.Fields{
		"directory": driver.Root(),
		"type":      driver.DriverType(),
		"volume":    c.Args.Name,
	})
	if !driver.Exists(c.Args.Name) {
		logger.Fatal("Volume does not exist")
	}
	logger.Info("Removeing volume")
	if err := driver.Remove(c.Args.Name); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Volume deleted")
	return nil
}
