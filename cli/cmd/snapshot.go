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
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
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
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "show-tags, t",
						Usage: "shows tags associated with each snapshot",
					},
				},
			}, {
				Name:         "add",
				Usage:        "Take a snapshot of an existing service",
				Description:  "serviced snapshot add SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdSnapshotAdd,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "description, d",
						Value: "",
						Usage: "a description of the snapshot",
					},
					cli.StringFlag{
						Name:  "tag, t",
						Value: "",
						Usage: "a unique tag for the snapshot",
					},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing snapshot",
				Description:  "serviced snapshot remove [SNAPSHOTID | SERVICED TAG-NAME]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdSnapshotRemove,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "force, f",
						Usage: "required for deleting all snapshots",
					},
				},
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
					cli.BoolFlag{
						Name:  "force-restart",
						Usage: "restarts running services during rollback",
					},
				},
				BashComplete: c.printSnapshotsFirst,
				Action:       c.cmdSnapshotRollback,
			}, {
				Name:         "tag",
				Usage:        "Tags an existing snapshot with TAG-NAME",
				Description:  "serviced snapshot tag SNAPSHOTID TAG-NAME",
				BashComplete: c.printSnapshotsFirst,
				Action:       c.cmdSnapshotTag,
			}, {
				Name:         "untag",
				Usage:        "Removes a tag from an existing snapshot",
				Description:  "serviced snapshot untag SERVICEID TAG-NAME",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdSnapshotRemoveTag,
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

// Bash-completion command that prints all the snapshot ids as the first argument and all tags associated with the snapshot ID in the first argument as the second.
func (c *ServicedCli) printSnapshotsFirstThenTags(ctx *cli.Context) {
	args := ctx.Args()

	if len(args) > 2 {
		return
	}

	snapshots := c.snapshots("")

	for _, s := range snapshots {
		if len(args) == 0 {
			fmt.Println(s.SnapshotID)
		} else if s.SnapshotID == args[0] {
			for _, t := range s.Tags {
				fmt.Println(t)
			}
		}
	}
}

// serviced snapshot list [SERVICEID] [--show-tags]
func (c *ServicedCli) cmdSnapshotList(ctx *cli.Context) {
	showTags := ctx.Bool("show-tags")
	var (
		snapshots []dao.SnapshotInfo
		err       error
	)
	if len(ctx.Args()) > 0 {
		serviceID := ctx.Args().First()
		if snapshots, err = c.driver.GetSnapshotsByServiceID(serviceID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	} else {
		if snapshots, err = c.driver.GetSnapshots(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}

	if snapshots == nil || len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "no snapshots found")
	} else {
		if showTags { //print a table of snapshot, description, tag list
			t := NewTable("Snapshot,Description,Tags")
			for _, s := range snapshots {
				//build a comma-delimited list of the tags
				tags := strings.Join(s.Tags, ",")
				snapshotID := s.SnapshotID
				if s.Invalid {
					snapshotID += " [DEPRECATED]"
				}

				//make the row and add it to the table
				row := make(map[string]interface{})
				row["Snapshot"] = snapshotID
				row["Description"] = s.Description
				row["Tags"] = tags
				t.Padding = 6
				t.AddRow(row)
			}
			//print the table
			t.Print()
		} else { //just print a list of snapshots
			for _, s := range snapshots {
				fmt.Println(s)
			}
		}
	}
	return
}

// serviced snapshot add SERVICEID [--tags=<tag1>,<tag2>...]
func (c *ServicedCli) cmdSnapshotAdd(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	cfg := api.SnapshotConfig{
		ServiceID: ctx.Args().First(),
		Message:   ctx.String("description"),
		Tag:       ctx.String("tag"),
	}
	if snapshot, err := c.driver.AddSnapshot(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if snapshot == "" {
		fmt.Fprintln(os.Stderr, "received nil snapshot")
		c.exit(1)
	} else {
		fmt.Println(snapshot)
	}
}

// serviced snapshot remove [SNAPSHOTID | SERVICED TAG-NAME]
func (c *ServicedCli) cmdSnapshotRemove(ctx *cli.Context) {

	args := ctx.Args()
	force := ctx.Bool("force")

	snapshotsToDelete := []string{}

	if len(args) > 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	if len(args) == 0 {
		// user wants to delete all snapshots, but check for -f first
		if !force {
			fmt.Printf("Incorrect Usage.\nUse\n   serviced snapshot remove -f\nto delete all snapshots, or\n   serviced snapshot remove -h\nfor help with this command.\n")
			return
		} else {
			//delete all snapshots
			snapshots, err := c.driver.GetSnapshots()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			for _, snapshot := range snapshots {
				snapshotsToDelete = append(snapshotsToDelete, snapshot.SnapshotID)
			}
		}
	} else if len(args) == 1 {
		//Delete the snapshot specified in the argument
		snapshotsToDelete = []string{args[0]}
	} else {
		//Find the snapshot with the given tag and service
		if snapshot, err := c.driver.GetSnapshotByServiceIDAndTag(args[0], args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		} else if snapshot != "" {
			snapshotsToDelete = []string{snapshot}
		}
	}

	if len(snapshotsToDelete) == 0 {
		fmt.Println("No matching snapshots found.")
		return
	}

	//Delete the chosen snapshots
	for _, id := range snapshotsToDelete {
		if err := c.driver.RemoveSnapshot(id); err != nil {
			fmt.Fprintln(os.Stderr, err)
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
	cfg := api.SnapshotConfig{
		DockerID: args[0],
	}

	if snapshot, err := c.driver.AddSnapshot(cfg); err != nil {
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
		c.exit(1)
	} else {
		fmt.Println(args[0])
	}
}

// serviced snapshot tag SNAPSHOTID TAG-NAME
func (c *ServicedCli) cmdSnapshotTag(ctx *cli.Context) {
	args := ctx.Args()

	var (
		err error
	)

	if len(args) != 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "tag")
		return
	}

	if err = c.driver.TagSnapshot(args[0], args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

// serviced snapshot untag SERVICEID TAG-NAME
func (c *ServicedCli) cmdSnapshotRemoveTag(ctx *cli.Context) {
	args := ctx.Args()

	var (
		snapshotID string
		err        error
	)

	if len(args) != 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "untag")
		return
	}

	//remove specified tag
	if snapshotID, err = c.driver.RemoveSnapshotTag(args[0], args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Printf("%s\n", snapshotID)
}
