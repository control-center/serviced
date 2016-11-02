// Copyright 2016 The Serviced Authors.
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
	"github.com/codegangsta/cli"
	"fmt"
)

// Initializer for serviced debug
func (c *ServicedCli) initDebug() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "debug",
		Usage:       "manage debugging",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "enable-metrics",
				Usage:        "Enable debug metrics",
				Description:  "serviced debug enable-metrics",
				Before:       c.cmdEnableDebugMetrics,
			},
			{
				Name:         "disable-metrics",
				Usage:        "Disable debug metrics",
				Description:  "serviced debug disable-metrics",
				Before:       c.cmdDisableDebugMetrics,
			},		},
	})
}

// serviced debug enable-metrics
func (c *ServicedCli) cmdEnableDebugMetrics(ctx *cli.Context) error {
	message, err := c.driver.DebugEnableMetrics()
	if err != nil {
		return fmt.Errorf("could not enable debug metrics: %s", err)
	}
	return fmt.Errorf(message)
}

// serviced debug disable-metrics
func (c *ServicedCli) cmdDisableDebugMetrics(ctx *cli.Context) error {
	message, err := c.driver.DebugDisableMetrics()
	if err != nil {
		return fmt.Errorf("could not disable debug metrics: %s", err)
	}
	return fmt.Errorf(message)
}

