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
	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
	"fmt"
	"github.com/pivotal-golang/bytefmt"
	"encoding/json"
	"os"
)

// Initializer for serviced pool subcommands
func (c *ServicedCli) initVolume() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "volume",
		Usage:       "Administers volume data",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "status",
				Usage:        "Provides volume status for specified tenants",
				Description:  "serviced volume status [TENANT [TENANT ...]]",
				Action:       c.cmdVolumeStatus,
				Flags:  []cli.Flag{
					cli.BoolFlag{"verbose, v", "Show JSON format"},
				},
			},
		},
	})
}

// serviced volume status
func (c *ServicedCli) cmdVolumeStatus(ctx *cli.Context) {
	glog.V(2).Info("cmd.cmdVolumeStatus()")
	args := ctx.Args()
	glog.V(2).Infof("\tctx.Args() = %+v\n", args)

	response, err := c.driver.GetVolumeStatus(args)
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
	fmt.Printf("Driver:                 %s\n", status.Driver)
	fmt.Printf("PoolName:               %s\n", status.PoolName)
	fmt.Printf("DataFile:               %s\n", status.DataFile)
	fmt.Printf("DataLoopback:           %s\n", status.DataLoopback)
	fmt.Printf("MetadataFile:           %s\n", status.MetadataFile)
	fmt.Printf("MetadataLoopback:       %s\n", status.MetadataLoopback)
	fmt.Printf("SectorSize:             %s\n", bytefmt.ByteSize(status.SectorSize))
	fmt.Printf("UdevSyncSupported:      %t\n", status.UdevSyncSupported)
	fmt.Printf("Usage Data:\n")
	for _, usage := range status.UsageData {
		fmt.Printf("\t%s %s: %d\n", usage.Label, usage.Type, usage.Value)
	}
}