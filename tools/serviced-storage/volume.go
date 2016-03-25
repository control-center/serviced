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
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/volume"
	"github.com/docker/go-units"
	"github.com/jessevdk/go-flags"
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

// DriverSync is the subcommand for syncing two volumes
type DriverSync struct {
	Create bool   `description:"Indicates that the destination driver should be created" long:"create" short:"c"`
	Type   string `description:"Type of the destination driver (btrfs|devicemapper|rsync)" long:"type" short:"t"`
	Args   struct {
		SourcePath      flags.Filename `description:"Path of the source driver"`
		DestinationPath flags.Filename `description:"Path of the destionation"`
	} `positional-args:"yes" required:"yes"`
}

//Execute syncs to volume
func (c *DriverSync) Execute(args []string) error {
	App.initializeLogging()
	destinationPath := string(c.Args.DestinationPath)
	sourcePath := string(c.Args.SourcePath)
	logger := log.WithFields(log.Fields{
		"destination": destinationPath,
		"source":      sourcePath})
	if c.Create {
		logger = logger.WithFields(log.Fields{
			"type": c.Type,
		})
		logger.Info("Determining driver type for destination")
		destinationDriverType, err := volume.StringToDriverType(c.Type)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Info("Creating driver for destination")
		initStatus := volume.InitDriver(destinationDriverType, destinationPath, App.Options.Options)
		if initStatus != nil {
			logger.Fatal(initStatus)
		}
	}
	destinationDirectory := GetDefaultDriver(destinationPath)
	logger.Info("Getting driver for destination")
	destinationDriver, err := InitDriverIfExists(destinationDirectory)
	if err != nil {
		logger.Fatal(err)
	}
	sourceDirectory := GetDefaultDriver(string(sourcePath))
	logger.Info("Getting driver for source")
	sourceDriver, err := InitDriverIfExists(sourceDirectory)
	if err != nil {
		logger.Fatal(err)
	}
	sourceVolumes := sourceDriver.List()
	logger = logger.WithFields(log.Fields{
		"numberOfVolumes": len(sourceVolumes),
	})
	for i := 0; i < len(sourceVolumes); i++ {
		volumeName := sourceVolumes[i]
		volumeLogger := logger.WithFields(log.Fields{
			"volumeName": volumeName,
		})
		volumeLogger.Info("Syncing data from source volume")
		sourceVolume, err := sourceDriver.Get(volumeName)
		if err != nil {
			volumeLogger.Fatal(err)
		}
		if !destinationDriver.Exists(volumeName) {
			logger.Info("Creating destination volume")
			createVolume(string(destinationPath), volumeName)
		}
		volumeLogger = volumeLogger.WithFields(log.Fields{
			"sourcePath": sourceVolume.Path(),
		})
		volumeLogger.Info("using rsync to sync source to destination")
		rsync(sourceVolume.Path(), string(c.Args.DestinationPath))
	}
	return nil
}

func rsync(sourcePath string, destinationPath string) {
	rsyncBin, err := exec.LookPath("rsync")
	if err != nil {
		log.Fatal(err)
	}
	rsyncArgv := []string{"-a", "--progress", "--stats", "--human-readable", sourcePath, destinationPath}
	rsync := exec.Command(rsyncBin, rsyncArgv...)
	log.Info("Starting rsync command")
	rsync.Stdout = os.Stdout
	rsync.Stderr = os.Stderr
	rsync.Run()
}

//CreateVolume creates a volume at path with name of name
func createVolume(path string, name string) {
	directory := GetDefaultDriver(path)
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.WithFields(log.Fields{
		"directory": driver.Root(),
		"type":      driver.DriverType(),
		"volume":    name,
	})
	logger.Info("Creating volume")
	vol, err := driver.Create(name)
	if err != nil {
		logger.Fatal(err)
	}
	log.WithFields(log.Fields{
		"mount": vol.Path(),
	}).Info("Volume Mounted")
}

// VolumeResize is the subcommand for enlarging an existing devicemapper volume
type VolumeResize struct {
	Path flags.Filename `long:"driver" short:"d" description:"Path of the driver"`
	Args struct {
		Name string `description:"Name of the volume to mount"`
		Size string `description:"New size of the volume"`
	} `positional-args:"yes" required:"yes"`
}

// Execute creates a new volume on a driver
func (c *VolumeCreate) Execute(args []string) error {
	App.initializeLogging()
	createVolume(string(c.Path), c.Args.Name)
	return nil
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
	logger.Info("Removing volume")
	if err := driver.Remove(c.Args.Name); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Volume deleted")
	return nil
}

// Resize increases the space available to an existing volume
func (c *VolumeResize) Execute(args []string) error {
	App.initializeLogging()
	directory := GetDefaultDriver(string(c.Path))
	driver, err := InitDriverIfExists(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.WithFields(log.Fields{
		"directory": driver.Root(),
		"volume":    c.Args.Name,
		"type":      driver.DriverType(),
	})
	if driver.DriverType() != volume.DriverTypeDeviceMapper {
		logger.Fatal("Only devicemapper volumes can be resized")
	}
	if !driver.Exists(c.Args.Name) {
		logger.Fatal("Volume does not exist")
	}
	size, err := units.RAMInBytes(c.Args.Size)
	if err != nil {
		return err
	}
	if err := driver.Resize(c.Args.Name, uint64(size)); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Volume resized")
	return nil
}
