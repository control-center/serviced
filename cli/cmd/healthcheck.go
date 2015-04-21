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

package cmd

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/dao"
)

const (
	serviceNameWidth   = 23
	containerNameWidth = 38
	containerIDWidth   = 12
	checkNameWidth     = 13
)

// Initializer for serviced healthcheck subcommands
func (c *ServicedCli) initHealthCheck() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "healthcheck",
		Usage:       "Reports on health of serviced",
		Description: "serviced healthcheck [ISERVICENAME-1 [ISERVICENAME-2 ... [ISERVICENAME-N]]]",
		Before:      c.cmdHealthCheck,
	})
}

// serviced healthcheck list [ISERVICENAME]
func (c *ServicedCli) cmdHealthCheck(ctx *cli.Context) error {

	if results, err := c.driver.ServicedHealthCheck(ctx.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return c.exit(2)
	} else {
		printHeader()
		exitStatus := 0
		for _, serviceHealth := range results {
			serviceDescription := getServiceDescription(serviceHealth)
			for _, status := range serviceHealth.HealthStatuses {
				if status.Status != "passed" {
					exitStatus = 1
				}
				printResult(serviceDescription, status.Name, status.Status)
			}
		}
		return c.exit(exitStatus)
	}
}

func printHeader() {
	fmt.Printf("%-*.*s %-*.*s %-*.*s  %-*.*s %-s\n",
		serviceNameWidth,
		serviceNameWidth,
		"Service Name",
		containerNameWidth,
		containerNameWidth,
		"Container Name",
		containerIDWidth,
		containerIDWidth,
		"Container ID",
		checkNameWidth,
		checkNameWidth,
		"Health Check",
		"Status")
}

func getServiceDescription(serviceHealth dao.IServiceHealthResult) string {
	return fmt.Sprintf("%-*.*s %-*.*s %-*.*s",
		serviceNameWidth,
		serviceNameWidth,
		serviceHealth.ServiceName,
		containerNameWidth,
		containerNameWidth,
		serviceHealth.ContainerName,
		containerIDWidth,
		containerIDWidth,
		serviceHealth.ContainerID)
}

func printResult(serviceDescription, healthCheckName, healthCheckStatus string) {
	fmt.Printf("%s  %-*.*s %-s\n",
		serviceDescription,
		checkNameWidth,
		checkNameWidth,
		healthCheckName,
		healthCheckStatus)
}
