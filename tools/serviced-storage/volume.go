package main

import (
	log "github.com/Sirupsen/logrus"
)

func init() {
	App.Parser.AddCommand("volume", "Volume subcommands", "Driver subcommands", &Volume{})
}

type Volume struct {
	Create VolumeCreate `command:"create" description:"Create a new volume"`
	Mount  VolumeMount  `command:"mount" description:"Mount an existing volume"`
}

type VolumeCreate struct {
	Args struct {
		Name string `description:"Name of the volume to create"`
	} `positional-args:"yes" required:"yes"`
}

type VolumeMount struct {
	Args struct {
		Name string `description:"Name of the volume to mount"`
	} `positional-args:"yes" required:"yes"`
}

func (c *VolumeCreate) Execute(args []string) error {
	driver, err := InitDriverIfExists(string(App.Options.Directory))
	if err != nil {
		return err
	}
	volumeName := c.Args.Name
	logger := log.WithFields(log.Fields{
		"directory": driver.Root(),
		"type":      driver.DriverType(),
		"volume":    volumeName,
	})
	logger.Info("Creating a volume")
	vol, err := driver.Create(volumeName)
	if err != nil {
		return err
	}
	logger.WithFields(log.Fields{
		"mount": vol.Path(),
	}).Info("Volume mounted")
	return nil
}

func (c *VolumeMount) Execute(args []string) error {
	driver, err := InitDriverIfExists(string(App.Options.Directory))
	if err != nil {
		return err
	}
	volumeName := c.Args.Name
	logger := log.WithFields(log.Fields{
		"directory": driver.Root(),
		"type":      driver.DriverType(),
		"volume":    volumeName,
	})
	logger.Info("Mounting volume")
	vol, err := driver.Get(volumeName)
	if err != nil {
		return err
	}
	logger.WithFields(log.Fields{
		"mount": vol.Path(),
	}).Info("Volume mounted")
	return nil
}
