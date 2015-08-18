package main

import (
	"github.com/control-center/serviced/volume"

	_ "github.com/control-center/serviced/volume/btrfs"
	_ "github.com/control-center/serviced/volume/devicemapper"
	_ "github.com/control-center/serviced/volume/rsync"

	log "github.com/Sirupsen/logrus"
)

func init() {
	App.Parser.AddCommand("driver", "Driver subcommands", "Driver subcommands", &Driver{})
}

type Driver struct {
	Init     DriverInit     `command:"init" description:"Initialize a driver"`
	Shutdown DriverShutdown `command:"shutdown" description:"Release any system resources held by a driver"`
}

type DriverInit struct {
	Args struct {
		Type    string   `description:"Type of driver to initialize (btrfs, devicemapper, rsync)"`
		Options []string `description:"name=value pairs of storage options"`
	} `positional-args:"yes" required:"yes"`
}

func (c *DriverInit) Execute(args []string) error {
	driverType, err := volume.StringToDriverType(c.Args.Type)
	if err != nil {
		return err
	}
	path := string(App.Options.Directory)
	log.WithFields(log.Fields{
		"path": path,
		"type": driverType,
	}).Info("Initializing storage driver")

	if err := volume.InitDriver(driverType, path, c.Args.Options); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"path": path,
		"type": driverType,
	}).Info("Storage driver initialized successfully")
	return nil
}

func InitDriverIfExists(directory string) error {
	driverType, err := volume.DetectDriverType(directory)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"directory": directory,
		"type":      driverType,
	}).Info("Found existing storage")
	if err := volume.InitDriver(driverType, directory, []string{}); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"directory": directory,
		"type":      driverType,
	}).Info("Initialized storage driver")
	return nil
}