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
	"net/rpc"
	"os"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/servicedversion"
	"github.com/zenoss/glog"
)

const defaultHostName = "defaultHost"

type Options struct {
	HostName   string
	ConfigFile string
	IPAddress  string
}

type Agent struct {
	app        *cli.App
	options    Options
	hostConfig HostConfig
}

func newAgent(app *cli.App) *Agent {
	agent := &Agent {
		app: app,
	}

	agent.app.Name = "mockAgent"
	agent.app.Usage = "mock implementation of a serviced agent"
	agent.app.Version = fmt.Sprintf("%s - %s ", servicedversion.Version, servicedversion.Gitcommit)
	agent.app.Flags = []cli.Flag {
		cli.StringFlag{"host", defaultHostName, "hostid to use"},
		cli.StringFlag{"config-file", "", "path to config file to use"},
		cli.StringFlag{"address", "", "IP address to use"},
	}
	agent.app.Before = agent.initialize
	agent.app.Action = agent.startAction

	return agent
}

func (agent *Agent) run(args []string) {
	if err := agent.app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}

func (agent *Agent) initialize(ctx *cli.Context) error {
	agent.options.HostName = ctx.GlobalString("host")
	agent.options.ConfigFile = ctx.GlobalString("config-file")
	agent.options.IPAddress = ctx.GlobalString("address")
	return nil
}

func (agent *Agent) startAction(ctx *cli.Context) {
	if err := agent.validateOptions(ctx); err != nil {
		os.Exit(1)
	}

	if err := agent.readConfiguration(); err != nil {
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Starting mock agent ...\n")
	if err := agent.runDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "Can't start daemon: %s\n", err)
		os.Exit(1)
	}
}

func (agent *Agent) validateOptions(ctx *cli.Context) error {
	if agent.options.HostName == "" || agent.options.ConfigFile == "" {
		err := fmt.Errorf("missing one of the required options --host and/or --config-file")
		fmt.Fprintf(os.Stderr, "Incorrect usage\n\n")
		cli.ShowAppHelp(ctx)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return err
	}

	return nil
}

func (agent *Agent) readConfiguration() error {
	file, err := ioutil.ReadFile(agent.options.ConfigFile)
	if err != nil {
		err = fmt.Errorf("Unable to read configuration file %q: %v\n", agent.options.ConfigFile, err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return err
	}

	configFile := make(map[string](map[string]HostConfig))
	if err = json.Unmarshal(file, &configFile); err != nil {
		err = fmt.Errorf("Unable to parse json from config file %q: %v\n", agent.options.ConfigFile, err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return err
	}

	var ok bool
	agent.hostConfig, ok = configFile["hosts"][agent.options.HostName]
	if !ok {
		err = fmt.Errorf("Unable to find host %q in config file %q\n", agent.options.HostName, agent.options.ConfigFile)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return err
	}

	return nil
}

func (agent *Agent) runDaemon() error {
	daemon, err := newDaemon(&agent.hostConfig, rpc.NewServer())
	if err != nil {
		glog.Fatalf("could not create server: %v", err)
	}

	err = daemon.run(agent.options.IPAddress)
	if err != nil {
		glog.Fatalf("could not start server: %v", err)
	}

	return nil
}
