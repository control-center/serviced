// Copyright 2014 The Serviced Authors.
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

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codegangsta/cli"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/script"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

// Initializer for serviced script
func (c *ServicedCli) initScript() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "script",
		Usage:       "serviced script",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:        "parse",
				Usage:       "Parse a script",
				Description: "serviced script parse FILE",
				Action:      c.cmdScriptParse,
			},
			{
				Name:        "run",
				Usage:       "Run a script",
				Description: "serviced script run FILE [--service SERVICEID] [-n]",
				Action:      c.cmdScriptRun,
				Flags: []cli.Flag{
					cli.StringFlag{"service", "", "Service to run this script against"},
					cli.BoolFlag{"no-op, n", "Run through script without modifying system"},
				},
			},
		},
	})
}

// cmdScriptRun serviced script run filename
func (c *ServicedCli) cmdScriptRun(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Incorrect Usage.\n\n")
		return
	}

	var svc *service.Service
	if svcID := ctx.String("service"); svcID != "" {
		//verify service or translate to ID
		var err error
		svc, err = c.searchForService(svcID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}
		if svc == nil {
			fmt.Fprintf(os.Stderr, "service %s not found\n", svcID)
			c.exit(1)
			return
		}
	}

	fileName := args[0]
	config := &script.Config{}
	if svc != nil {
		config.ServiceID = svc.ID
	}

	// exec unix script command to log output
	if isWithin := os.Getenv("IS_WITHIN_UNIX_SCRIPT"); isWithin != "TRUE" {
		os.Setenv("IS_WITHIN_UNIX_SCRIPT", "TRUE") // prevent inception problem

		// DO NOT EXIT ON ANY ERRORS - continue without logging
		if logdir, err := utils.UserHomeServicedDir(); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to log output - error getting home serviced dir: %s\n", err)
		} else if err := os.MkdirAll(logdir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to log output - error creating dir %s: %s", logdir, err)
		} else {
			logfile := time.Now().Format(fmt.Sprintf("%s/script.log-2006-01-02-150405", logdir))

			// unix exec ourselves
			cmd := []string{"/usr/bin/script", "--append", "--return", "--flush",
				"-c", strings.Join(os.Args, " "), logfile}

			fmt.Fprintf(os.Stderr, "Logging to logfile: %s\n", logfile)
			glog.V(1).Infof("syscall.exec unix script with command: %+v", cmd)
			if err := syscall.Exec(cmd[0], cmd[0:], os.Environ()); err != nil {
				fmt.Fprintf(os.Stderr, "Unable to log output with command:%+v err:%s\n", cmd, err)
			}
		}
	}

	glog.V(1).Infof("runScript filename:%s %+v\n", fileName, config)
	runScript(c, ctx, fileName, config)
}

// cmdScriptParse serviced script parse filename
func (c *ServicedCli) cmdScriptParse(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Incorrect Usage.\n\n")
		return
	}
	fileName := args[0]
	config := script.Config{}
	err := c.driver.ScriptParse(fileName, &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return
}

func runScript(c *ServicedCli, ctx *cli.Context, fileName string, config *script.Config) {
	stopChan := make(chan struct{})
	signalHandlerChan := make(chan os.Signal)
	signal.Notify(signalHandlerChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalHandlerChan
		glog.Infof("Received stop signal, stopping")
		close(stopChan)
	}()

	config.NoOp = ctx.Bool("no-op")
	config.DockerRegistry = docker.DEFAULT_REGISTRY
	err := c.driver.ScriptRun(fileName, config, stopChan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}
}
