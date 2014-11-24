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
	"github.com/codegangsta/cli"
	"fmt"
)

// Initializer for serviced metric
func (c *ServicedCli) initMetric() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "metric",
		Usage:       "Interact with metrics",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "push",
				Usage:        "Push a metric value",
				Description:  "serviced metric push METRICNAME VALUE",
				Before:       c.cmdMetric,
			},
		},
	})
}

// serviced metric
func (c *ServicedCli) cmdMetric(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Println("Insufficient number of arguments")
		return fmt.Errorf("Insufficient number of arguments")
	}
	metricName := args[0]
	metricValue := args[1]
	message, err := c.driver.PostMetric(metricName, metricValue)
	if err != nil {
		return fmt.Errorf("could not post stats: %s", err)
	}
	return fmt.Errorf(message)
}

