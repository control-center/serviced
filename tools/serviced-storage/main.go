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

/*
serviced-storage COMMAND ARGS

Commands:
	init     TYPE PATH [OPTIONS...]
	set      PATH
	unset
	disable  [PATH]
	status   [PATH]
	list     [PATH]
	create   NAME [-d PATH]
	mount    NAME [-d PATH]
	remove   NAME [-d PATH]
	version
*/
package main

import (
	"fmt"
	"os"
	"os/user"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/servicedversion"
	"github.com/jessevdk/go-flags"
	"github.com/zenoss/glog"
)

var Version string
var Date string
var Gitcommit string
var Gitbranch string
var Giturl string
var Buildtag string

var (
	name    = "serviced-storage"
	App     = &ServicedStorage{
		name:    name,
		Parser:  flags.NewNamedParser(name, flags.Default),
	}
)

func init() {
	servicedversion.Version = Version
	servicedversion.Date = Date
	servicedversion.Gitcommit = Gitcommit
	servicedversion.Gitbranch = Gitbranch
	servicedversion.Giturl = Giturl
	servicedversion.Buildtag = Buildtag

	App.version = fmt.Sprintf("%s - %s", servicedversion.Version, servicedversion.Gitcommit)

	App.Parser.AddCommand("init", "Initialize a driver", "Initialize a driver", &DriverInit{})
	App.Parser.AddCommand("set", "Set the default driver", "Set the default driver", &DriverSet{})
	App.Parser.AddCommand("unset", "Unset the default driver", "Unset the default driver", &DriverUnset{})
	App.Parser.AddCommand("disable", "Disable a driver", "Disable a driver", &DriverDisable{})
	App.Parser.AddCommand("status", "Print the driver status", "Print the driver status", &DriverStatus{})
	App.Parser.AddCommand("list", "Print volumes on a driver", "Print volumes on a driver", &DriverList{})
	App.Parser.AddCommand("create", "Create a volume on a driver", "Create a volume on a driver", &VolumeCreate{})
	App.Parser.AddCommand("mount", "Mount an existing volume from a driver", "Mount an existing volume from a driver", &VolumeMount{})
	App.Parser.AddCommand("remove", "Remove an existing volume from a driver", "Remove an existing volume from a driver", &VolumeRemove{})
	App.Parser.AddCommand("resize", "Resize an existing volume", "Resize an existing volume", &VolumeResize{})
	App.Parser.AddCommand("sync", "Sync data from a volume to another volume", "Sync data from a volume to another volume", &DriverSync{})
	App.Parser.AddCommand("version", "Print the version and exit", "Print the version and exit", &ServicedStorageVersion{})
	App.Parser.AddCommand("create-thin-pool", "Create thin pool", "Create a thin pool from a set of block devices", &ThinPoolCreate{})
}

// ServicedStorageOptions are the options for ServicedStorage
type ServicedStorageOptions struct {
	Verbose []bool   `short:"v" description:"Display verbose logging"`
	Options []string `short:"o" description:"name=value pairs of storage options"`
}

// ServicedStorage is the root client for the application
type ServicedStorage struct {
	name    string
	version string
	Parser  *flags.Parser
	Options ServicedStorageOptions
}

// ServicedStorageVersion is the versioning command for the application
type ServicedStorageVersion struct{}

// Run organizes the options for the application
func (s *ServicedStorage) Run() {
	// Set up some initial logging for the sake of parser errors
	s.initializeLogging()
	if usr, err := user.Current(); err != nil {
		log.Fatal(err)
	} else if usr.Uid != "0" {
		log.Fatalf("%s must be run by root", App.name)
	}
	if _, err := s.Parser.AddGroup("Basic Options", "Basic options", &s.Options); err != nil {
		log.WithFields(log.Fields{"exitcode": 1}).Fatal("Unable to add option group")
		os.Exit(1)
	}
	s.Parser.Parse()
}

// initializeLogging initializes the logger for the application
func (s *ServicedStorage) initializeLogging() {
	log.SetOutput(os.Stderr)
	level := log.WarnLevel + log.Level(len(App.Options.Verbose))
	log.SetLevel(level)

	// Include glog output if verbosity is enabled
	if len(App.Options.Verbose) > 0 {
		glog.SetToStderr(true)
		glog.SetVerbosity(len(App.Options.Verbose))
	}
}

// Execute prints the application version to stdout and exits
func (c *ServicedStorageVersion) Execute(args []string) error {
	fmt.Println(App.version)
	return nil
}

func main() {
	App.Run()
}
