// Copyright 2020 The Serviced Authors.
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
serviced-service COMMAND ARGS

Commands:
	compile PATH
		Compiles service templates in a directory into a single template file

	deploy TEMPLATE_FILE
		Converts a template into service definitions

	update TEMPLATE_FILE SOURCE_FILE CHANGES
		Applies the modifications in the CHANGES file to the SOURCE file.

	version
		Prints the version to stdout
*/

package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"

	"github.com/control-center/serviced/servicedversion"
	"github.com/zenoss/glog"
)

var (
	// Version is the version of this app
	Version string
	// GoVersion is the version of go compiler
	GoVersion string
	// Date is when the app was compiled
	Date string
	// Gitcommit is the git ref of source code that was compiled
	Gitcommit string
	// Gitbranch was the branch the source code is found in
	Gitbranch string
	// Buildtag is just some text identifying the build
	Buildtag string

	name = "serviced-service"
	// App is the root object of the application
	App = &ServicedService{
		name:   name,
		Parser: flags.NewNamedParser(name, flags.Default),
	}
)

func init() {
	servicedversion.Version = Version
	servicedversion.GoVersion = GoVersion
	servicedversion.Date = Date
	servicedversion.Gitcommit = Gitcommit
	servicedversion.Gitbranch = Gitbranch
	servicedversion.Buildtag = Buildtag

	App.version = fmt.Sprintf("%s - %s", servicedversion.Version, servicedversion.Gitcommit)

	App.Parser.AddGroup("Global Options", "Global options", &App.Options)
	App.Parser.AddCommand(
		"compile",
		"Convert a directory of templates into a single template",
		"Convert a directory of templates into a single template",
		&Compile{},
	)
	App.Parser.AddCommand(
		"deploy",
		"Convert a service template into service definitions",
		"Convert a service template into service definitions",
		&Deploy{},
	)
	App.Parser.AddCommand(
		"update",
		"Update service definitions",
		"Update service definitions from an update file",
		&Update{},
	)
	App.Parser.AddCommand(
		"version",
		"Print the version and exit",
		"Print the version and exit",
		&ServicedServiceVersion{},
	)
}

// ServicedServiceOptions are the options for ServicedService
type ServicedServiceOptions struct {
	Verbose []bool `short:"v" description:"Display verbose logging"`
}

// ServicedService is the root data structure for the application
type ServicedService struct {
	name    string
	version string
	Parser  *flags.Parser
	Options ServicedServiceOptions
}

// Run organizes the options for the application
func (s *ServicedService) Run() {
	// Set up some initial logging for the sake of parser errors
	s.initializeLogging()
	// if _, err := s.Parser.AddGroup("Basic Options", "Basic options", &s.Options); err != nil {
	// 	log.WithFields(log.Fields{"exitcode": 1}).Fatal("Unable to add option group")
	// 	os.Exit(1)
	// }
	s.Parser.Parse()
}

// initializeLogging initializes the logger for the application
func (s *ServicedService) initializeLogging() {
	log.SetOutput(os.Stderr)
	level := log.WarnLevel + log.Level(len(App.Options.Verbose))
	log.SetLevel(level)

	// Include glog output if verbosity is enabled
	if len(App.Options.Verbose) > 0 {
		glog.SetToStderr(true)
		glog.SetVerbosity(len(App.Options.Verbose))
	}
}

func main() {
	App.Run()
}
