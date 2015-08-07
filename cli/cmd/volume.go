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
	"strings"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
	"fmt"
	"strconv"
	"github.com/pivotal-golang/bytefmt"
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
				BashComplete: c.printVolumesAll,
				Action:       c.cmdVolumeStatus,
			},
			/* {
				Name:  "resize",
				Usage: "Resizes a volume",
				Description:  "serviced volume resize TENANT SIZE",
				BashComplete: c.printVolumesFirst,
				Action:       c.cmdVolumeResize,
			},  */
		},
	})
}

// Bash-completion command that prints a list of available services as the
// first argument
func (c *ServicedCli) printVolumesFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}
	glog.V(2).Infof("%s", strings.Join(c.volumes(), "\n"))   // TODO: remove or add V level
}

// Bash-completion command that prints a list of available services as all
// arguments
func (c *ServicedCli) printVolumesAll(ctx *cli.Context) {
	glog.V(2).Infof("volume.go: printVolumesAll(%+v).\n", ctx) // TODO: remove or add V level
	args := ctx.Args()
	svcs := c.volumes()

	// If arg is a service don't add to the list
	for _, s := range svcs {

		for _, a := range args {
			if s == a {
				goto next
			}
		}
		glog.V(2).Infof("%s",s)   // TODO: remove or add V level
		next:
	}
}

// Returns a list of all the available service IDs
func (c *ServicedCli) volumes() (data []string) {
	data = []string {"test1", "test2", "test3"}
	return
}
// serviced volume status
func (c *ServicedCli) cmdVolumeStatus(ctx *cli.Context) {
	glog.V(2).Info("cmd.cmdVolumeStatus()")    // TODO: remove or add V level
	args := ctx.Args()
	glog.V(2).Infof("\tctx.Args() = %+v\n", args) // TODO: remove or add V level

	//response := &volume.Statuses{}
	response, err := c.driver.GetVolumeStatus(args)
	//states, err := c.driver.GetVolumeStatus(args[0])
	if err != nil {
		glog.Errorf("error getting volume status: %v", err)     // TODO: remove or add V level
		return
	}
	printStatuses(response)
	return
}

// serviced volume resize
func (c *ServicedCli) cmdVolumeResize(ctx *cli.Context) {
	glog.V(2).Infof("cmdVolumeResize()")      // TODO: remove or add V level
	args := ctx.Args()
	glog.V(2).Infof("\tctx.Args() = %+v\n", args) // TODO: remove or add V level
	volumeName := args[0]
	newSize, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		glog.Errorf("Error parsing argument %s as size: %v", args[1], err)
		return
	}

	response, err := c.driver.ResizeVolume(volumeName, newSize)
	if err != nil {
		glog.Errorf("Error resizing volume: %v", err)     // TODO: remove or add V level
		return
	}

	fmt.Printf("New status for volume %s", volumeName)
	printStatus(response)
	return
}

func printStatuses(statuses *volume.Statuses) {
	for path, status := range statuses.StatusMap {
		fmt.Printf("Status for volume %s:\n", path)
		printStatus(&status)
	}
}

func printStatus(status *volume.Status) {
	fmt.Printf("Driver:                 %s\n", status.Driver)
	fmt.Printf("PoolName:               %s\n", status.PoolName)
	fmt.Printf("DataFile:               %s\n", status.DataFile)
	fmt.Printf("DataLoopback:           %s\n", status.DataLoopback)
	fmt.Printf("DataSpaceAvailable:     %s\n", bytefmt.ByteSize(status.DataSpaceAvailable))
	fmt.Printf("DataSpaceUsed:          %s\n", bytefmt.ByteSize(status.DataSpaceUsed))
	fmt.Printf("DataSpaceTotal:         %s\n", bytefmt.ByteSize(status.DataSpaceTotal))
	fmt.Printf("MetadataFile:           %s\n", status.MetadataFile)
	fmt.Printf("MetadataLoopback:       %s\n", status.MetadataLoopback)
	fmt.Printf("MetadataSpaceAvailable: %s\n", bytefmt.ByteSize(status.MetadataSpaceAvailable))
	fmt.Printf("MetadataSpaceUsed:      %s\n", bytefmt.ByteSize(status.MetadataSpaceUsed))
	fmt.Printf("MetadataSpaceTotal:     %s\n", bytefmt.ByteSize(status.MetadataSpaceTotal))
	fmt.Printf("SectorSize:             %s\n", bytefmt.ByteSize(status.SectorSize))
	fmt.Printf("UdevSyncSupported:      %t\n", status.UdevSyncSupported)
}