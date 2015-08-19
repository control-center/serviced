package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/control-center/serviced/volume"
	"github.com/jessevdk/go-flags"
	"github.com/pivotal-golang/bytefmt"

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
	Status   DriverStatus   `command:"status" description:"Print driver status"`
	List     DriverList     `command:"list" alias:"ls" description:"List volumes maintained by this driver"`
	Set      DriverSet      `command:"set" description:"Use a particular driver for volume operations"`
	Unset    DriverUnset    `command:"unset" description:"Clear driver cache"`
}

type DriverInit struct {
	Args struct {
		Type    string   `description:"Type of driver to initialize (btrfs, devicemapper, rsync)"`
		Options []string `description:"name=value pairs of storage options"`
	} `positional-args:"yes" required:"yes"`
}

type DriverShutdown struct{}
type DriverStatus struct{}
type DriverList struct{}

type DriverSet struct {
	Args struct {
		Path flags.Filename `description:"Driver directory to set"`
	} `positional-args:"yes" required:"yes"`
}

type DriverUnset struct{}

// If a driver exists in the given directory, initialize and return it
func InitDriverIfExists(directory string) (volume.Driver, error) {
	driverType, err := volume.DetectDriverType(directory)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.WithFields(log.Fields{
		"directory": directory,
		"type":      driverType,
	})
	logger.Debug("Found existing storage")
	if err := volume.InitDriver(driverType, directory, []string{}); err != nil {
		log.Fatal(err)
	}
	logger.Debug("Initialized storage driver")
	driver, err := volume.GetDriver(directory)
	if err != nil {
		log.Fatal(err)
	}
	return driver, nil
}

// Get the appropriate driver required by command line args
func GetDriver() (volume.Driver, *log.Entry) {
	// get the directory as stored in the state file
	var directory string
	if usr, err := user.Current(); err == nil {
		state, _ := ioutil.ReadFile(filepath.Join(usr.HomeDir, App.name))
		directory = string(state)
	}
	// overwrite state if -d flag is set
	if dir := string(App.Options.Directory); dir != "" {
		directory = dir
	}
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
	return driver, logger
}

func (c *DriverInit) Execute(args []string) error {
	App.initializeLogging()
	driverType, err := volume.StringToDriverType(c.Args.Type)
	if err != nil {
		log.Fatal(err)
	}
	path := string(App.Options.Directory)
	logger := log.WithFields(log.Fields{
		"directory": path,
		"type":      driverType,
	})
	logger.Info("Initializing storage driver")
	if err := volume.InitDriver(driverType, path, c.Args.Options); err != nil {
		log.Fatal(err)
	}
	logger.Info("Storage driver initialized successfully")
	return nil
}

func (d *DriverShutdown) Execute(args []string) error {
	App.initializeLogging()
	driver, logger := GetDriver()
	if err := driver.Cleanup(); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Shut down driver")
	return nil
}

func (d *DriverStatus) Execute(args []string) error {
	App.initializeLogging()
	driver, logger := GetDriver()
	status, err := driver.Status()
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("Driver:                 %s\n", status.Driver)
	fmt.Printf("PoolName:               %s\n", status.PoolName)
	fmt.Printf("DataFile:               %s\n", status.DataFile)
	fmt.Printf("DataLoopback:           %s\n", status.DataLoopback)
	fmt.Printf("DataSpaceAvailable:     %s\n", bytefmt.ByteSize(status.DataSpaceAvailable))
	fmt.Printf("DataSpaceUsed:          %s\n", bytefmt.ByteSize(status.DataSpaceUsed))
	fmt.Printf("DataSpaceTotal:         %s\n", bytefmt.ByteSize(status.DataSpaceTotal))
	fmt.Printf("MetadataFile:           %s\n", status.MetadataFile)
	fmt.Printf("MetadataLoopback:       %s\n", status.MetadataLoopback)
	fmt.Printf("MetadataSpaceAvailable: %s\n", bytefmt.ByteSize(status.MetadataSpaceAvailable))
	fmt.Printf("MetadataSpaceUsed:      %s\n", bytefmt.ByteSize(status.MetadataSpaceUsed))
	fmt.Printf("MetadataSpaceTotal:     %s\n", bytefmt.ByteSize(status.MetadataSpaceTotal))
	fmt.Printf("SectorSize:             %s\n", bytefmt.ByteSize(status.SectorSize))
	fmt.Printf("UdevSyncSupported:      %t\n", status.UdevSyncSupported)
	return nil
}

func (d *DriverList) Execute(args []string) error {
	App.initializeLogging()
	driver, _ := GetDriver()
	for _, volname := range driver.List() {
		fmt.Println(volname)
	}
	return nil
}

func (d *DriverSet) Execute(args []string) error {
	App.initializeLogging()
	directory := string(d.Args.Path)
	if _, err := volume.DetectDriverType(directory); err == volume.ErrDriverNotInit {
		log.Fatalf("Driver not initialized.  Use `%s driver init`.", App.name)
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(usr.HomeDir, App.name), []byte(directory), 0644); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *DriverUnset) Execute(args []string) error {
	App.initializeLogging()
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(usr.HomeDir, App.name)); err != nil && os.IsNotExist(err) {
		log.Fatal(err)
	}
	return nil
}
