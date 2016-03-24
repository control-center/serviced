// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

// Initializer for serviced pool subcommands
func (c *ServicedCli) initVolume() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "volume",
		Usage:       "Administers volume data",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:        "status",
				Usage:       "Provides volume status for application storage",
				Description: "serviced volume status",
				Action:      c.cmdVolumeStatus,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:   "verbose, v",
						Usage:  "Show JSON format",
					},
				},
			},
		},
	})
}

// serviced volume status
func (c *ServicedCli) cmdVolumeStatus(ctx *cli.Context) {
	response, err := c.driver.GetVolumeStatus()
	if err != nil {
		glog.Errorf("error getting volume status: %v", err)
		return
	}
	if ctx.Bool("verbose") {
		printStatusesJson(response)
	} else {
		printStatuses(response)
	}
	return
}

func printStatuses(statuses *volume.Statuses) {
	for path, status := range statuses.StatusMap {
		fmt.Printf("Status for volume %s:\n", path)
		printStatusText(&status)
	}
}

func printStatusesJson(statuses *volume.Statuses) {
	if jsonStatuses, err := json.MarshalIndent(statuses, " ", "  "); err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal volume status list: %s", err)
	} else {
		fmt.Println(string(jsonStatuses))
	}
}

func printStatusText(status *volume.Status) {
	fmt.Printf("%s\n", status.String())
}
