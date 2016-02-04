// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
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
	"time"

	"github.com/codegangsta/cli"
	sdcli "github.com/control-center/serviced/cli"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

const (
	outboundIPRetryDelay = 1
	outboundIPMaxWait    = 90
)

// Initializer for serviced server
func (c *ServicedCli) initServer() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "server",
		Usage:       "Starts serviced",
		Description: "serviced server",
		Action:      c.cmdServer,
	})
}

// serviced server
func (c *ServicedCli) cmdServer(ctx *cli.Context) {
	master := sdcli.GetOptionsMaster()
	agent := sdcli.GetOptionsAgent()

	// Make sure one of the configurations was specified
	if !master && !agent {
		fmt.Fprintf(os.Stderr, "serviced cannot be started: no mode (master or agent) was specified\n")
		return
	}

	// Make sure we have an endpoint to work with
	if endpoint := sdcli.GetOptionsRPCEndpoint(); len(endpoint) == 0 {
		if master {
			outboundIP, err := getOutboundIP()
			if err != nil {
				glog.Fatal(err)
			}
			endpoint := fmt.Sprintf("%s:%s", outboundIP, sdcli.GetOptionsRPCPort())
			sdcli.SetOptionsRPCEndpoint(endpoint)
		} else {
			glog.Fatal("No endpoint to master has been configured")
		}
	}

	if master {
		fmt.Println("This master has been configured to be in pool: " + sdcli.GetOptionsMasterPoolID())
	}

	// Start server mode
	rpcutils.RPC_CLIENT_SIZE = sdcli.GetOptionsMaxRPCClients()
	if err := c.driver.StartServer(); err != nil {
		glog.Fatalf("Could not start server: %s", err)
	}
}

// getOutboundIP queries the network configuration for an IP address suitable for reaching the outside world.
// Will retry for a while if a path to the outside world is not yet available.
func getOutboundIP() (string, error) {
	var outboundIP string
	var err error
	timeout := time.After(outboundIPMaxWait * time.Second)
	for {
		if outboundIP, err = utils.GetIPAddress(); err == nil {
			// Success
			return outboundIP, nil
		} else {
			select {
			case <-timeout:
				// Give up
				return "", fmt.Errorf("Gave up waiting for network (to determine our outbound IP address): %s", err)
			default:
				// Retry
				glog.Info("Waiting for network initialization...")
				time.Sleep(outboundIPRetryDelay * time.Second)
			}
		}
	}
}
