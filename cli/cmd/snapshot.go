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

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/dao"
)

// initSnapshot is the initializer for serviced snapshot
func (c *ServicedCli) initSnapshot() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "snapshot",
		Usage:       "Administers environment snapshots",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Lists all snapshots",
				Description:  "serviced snapshot list [SERVICEID]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdSnapshotList,
			}, {
				Name:         "add",
				Usage:        "Take a snapshot of an existing service",
				Description:  "serviced snapshot add SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdSnapshotAdd,
				Flags: []cli.Flag{
					cli.StringFlag{"description, d", "", "a description of the snapshot"},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing snapshot",
				Description:  "serviced snapshot remove SNAPSHOTID ...",
				BashComplete: c.printSnapshotsAll,
				Action:       c.cmdSnapshotRemove,
			}, {
				Name:        "commit",
				Usage:       "Snapshots and commits a given service instance",
				Description: "serviced snapshot commit DOCKERID",
				Action:      c.cmdSnapshotCommit,
			}, {
				Name:        "rollback",
				Usage:       "Restores the environment to the state of the given snapshot",
				Description: "serviced snapshot rollback SNAPSHOTID",
				Flags: []cli.Flag{
					cli.BoolFlag{"force-restart", "restarts running services during rollback"},
				},
				BashComplete: c.printSnapshotsFirst,
				Action:       c.cmdSnapshotRollback,
			},
		},
	})
}

// Returns a list of snapshots as specified by the service ID.  If no service
// ID is set, then returns a list of all snapshots.
func (c *ServicedCli) snapshots(id string) []dao.SnapshotInfo {
	var (
		snapshots []dao.SnapshotInfo
		err       error
	)

	if id != "" {
		snapshots, err = c.driver.GetSnapshotsByServiceID(id)
	} else {
		snapshots, err = c.driver.GetSnapshots()
	}

	if err != nil || snapshots == nil || len(snapshots) == 0 {
		return []dao.SnapshotInfo{}
	}

	return snapshots
}

// Bash-completion command that prints all the snapshot ids as the first
// argument
func (c *ServicedCli) printSnapshotsFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	for _, s := range c.snapshots("") {
		fmt.Println(s)
	}
}

// Bash-completion command that prints all the snapshot ids as all arguments.
func (c *ServicedCli) printSnapshotsAll(ctx *cli.Context) {
	args := ctx.Args()

	for _, s := range c.snapshots("") {
		for _, a := range args {
			if s.SnapshotID == a {
				goto next
			}
			fmt.Println(s)
		next:
		}
	}
}

// serviced snapshot list [SERVICEID]
func (c *ServicedCli) cmdSnapshotList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		serviceID := ctx.Args().First()
		if snapshots, err := c.driver.GetSnapshotsByServiceID(serviceID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if snapshots == nil || len(snapshots) == 0 {
			fmt.Fprintln(os.Stderr, "no snapshots found")
		} else {
			for _, s := range snapshots {
				fmt.Println(s)
			}
		}
		return
	}

	if snapshots, err := c.driver.GetSnapshots(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if snapshots == nil || len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "no snapshots found")
	} else {
		for _, s := range snapshots {
			fmt.Println(s)
		}
	}
}

// serviced snapshot add SERVICEID
func (c *ServicedCli) cmdSnapshotAdd(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	description := ctx.String("description")
	if snapshot, err := c.driver.AddSnapshot(ctx.Args().First(), description); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if snapshot == "" {
		fmt.Fprintln(os.Stderr, "received nil snapshot")
	} else {
		fmt.Println(snapshot)
	}
}

// serviced snapshot remove SNAPSHOTID ...
func (c *ServicedCli) cmdSnapshotRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	for _, id := range args {
		if err := c.driver.RemoveSnapshot(id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", id, err)
		} else {
			fmt.Println(id)
		}
	}
}

// serviced snapshot commit DOCKERID
func (c *ServicedCli) cmdSnapshotCommit(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "commit")
		return
	}

	if snapshot, err := c.driver.Commit(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if snapshot == "" {
		fmt.Fprintln(os.Stderr, "received nil snapshot")
	} else {
		fmt.Println(snapshot)
	}
}

// serviced snapshot rollback SNAPSHOTID [--force-restart]
func (c *ServicedCli) cmdSnapshotRollback(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "rollback")
		return
	}

	if err := c.driver.Rollback(args[0], ctx.Bool("force-restart")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println(args[0])
	}
}
