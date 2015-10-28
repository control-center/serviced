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
					cli.BoolFlag{"show-tags, t", "shows tags associated with each snapshot"},
				},
			}, {
				Name:         "add",
				Usage:        "Take a snapshot of an existing service",
				Description:  "serviced snapshot add SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdSnapshotAdd,
				Flags: []cli.Flag{
					cli.StringFlag{"description, d", "", "a description of the snapshot"},
					cli.StringFlag{"tags, t", "", "a comma-delimited list of tags for the snapshot"},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing snapshot",
				Description:  "serviced snapshot remove [SNAPSHOTID | TAG-NAME] ...",
				BashComplete: c.printSnapshotsAndTagsAll,
				Action:       c.cmdSnapshotRemove,
				Flags: []cli.Flag{
					cli.BoolFlag{"force, f", "deletes all matching snapshots without prompt, required for deleting all snapshots or deleting by tag"},
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
					cli.BoolFlag{"force-restart", "restarts running services during rollback"},
				},
				BashComplete: c.printSnapshotsFirst,
				Action:       c.cmdSnapshotRollback,
			}, {
				Name:        "tag",
				Usage:       "Tags an existing snapshot with 1 or more TAG-NAMEs",
				Description: "serviced snapshot tag SNAPSHOTID TAG-NAME ...",
				BashComplete: c.printSnapshotsFirstThenTags,
				Action:       c.cmdSnapshotTag,
			 },{
				Name:        "removetags",
				ShortName:	 "rmtags",
				Usage:       "Removes tags from an existing snapshot",
				Description: "serviced snapshot removetags SNAPSHOTID [TAG-NAME ...]",
				BashComplete: c.printSnapshotsFirstThenTags,
				Action:       c.cmdSnapshotRemoveTags,
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

// Bash-completion command that prints all the snapshot ids and all used tag names as all arguments.
func (c *ServicedCli) printSnapshotsAndTagsAll(ctx *cli.Context) {
	args := ctx.Args()

	suggestionList := []string{}
	for _, snap := range c.snapshots("") {
		suggestionList = append(suggestionList, snap.SnapshotID)
		suggestionList = append(suggestionList, snap.Tags...)
	}

	for _, s := range suggestionList {
		for _, a := range args {
			if s == a {
				goto next
			}
			fmt.Println(s)
		next:
		}
	}
}

// Bash-completion command that prints all the snapshot ids as the first argument and all used tag names as all other arguments.
func (c *ServicedCli) printSnapshotsFirstThenTags(ctx *cli.Context) {
	args := ctx.Args()
	snapshots := c.snapshots("")

	for _, s := range snapshots {
		if len(args) == 0 {
			fmt.Println(s.SnapshotID)
		} else {
			for _, t := range s.Tags{
				for _, a := range args {
					if t == a {
						goto next
					}
				}
				fmt.Println(t)
			next:
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
				//built a comma-delimted list of the tags
				var tags string

				for _, tag := range s.Tags {
					tags += tag + ","
				}
				tags = strings.Trim(tags, ",")

				//make the row and add it to the table
				row := make(map[string]interface{})
				row["Snapshot"] = s.SnapshotID
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

	//get the tags (if any)
	tags := ctx.String("tags")
	//Remove spaces around commas and at the end
	tags = strings.Replace(tags, ", ", ",", -1)
	tags = strings.Replace(tags, " ,", ",", -1)
	tags = strings.Trim(tags, " ")
	tags = strings.Trim(tags, ",")

	var tagList []string
	if len(tags) > 0 {
		tagList = strings.Split(tags, ",")
	} else {
		tagList = []string{}
	}

	cfg := api.SnapshotConfig{
		ServiceID: 	ctx.Args().First(),
		Message:   	ctx.String("description"),
		Tags:		tagList,
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

// serviced snapshot remove [SNAPSHOTID | TAG-NAME] ...
func (c *ServicedCli) cmdSnapshotRemove(ctx *cli.Context) {
	var (
		snapshots 			[]dao.SnapshotInfo
		snapshotsToDelete	[]string
		err 				error
	)

	args := ctx.Args()
	force := ctx.Bool("force")

	if snapshots, err = c.driver.GetSnapshots(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	deleteAll := false

	if len(args) < 1 {
		deleteAll = true
		if !force {
			fmt.Printf("Incorrect usage:\nUse\n   serviced snapshot remove -f\nto delete all snapshots, or\n   serviced snapshot remove -h\nfor help with this command.\n")
			return
		}
	} 

	for _, snapshot := range snapshots {
		if deleteAll {
			snapshotsToDelete = append(snapshotsToDelete, snapshot.SnapshotID)
		} else {
			//compare to args
			for _, arg := range args {
				if arg == snapshot.SnapshotID {
					snapshotsToDelete = append(snapshotsToDelete, snapshot.SnapshotID)
					break
				} else if snapshot.HasTag(arg) {
					if !force {
						fmt.Printf("Incorrect Usage.  '-f' required to force deletion based on tags\n")
						return
					}
					snapshotsToDelete = append(snapshotsToDelete, snapshot.SnapshotID)
					break
				}
			}
		}
	}

	if len(snapshotsToDelete) == 0 {
		fmt.Println("No matching snapshots found.")
	}

	//Delete the chosen snapshots
	for _, id := range snapshotsToDelete {
		if err = c.driver.RemoveSnapshot(id); err != nil {
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

// serviced snapshot tag SNAPSHOTID TAG-NAME ...
func (c *ServicedCli) cmdSnapshotTag(ctx *cli.Context) {
	args := ctx.Args()

	var (
		newTags []string
		err      error
	)

	if len(args) < 2{
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "tag")
		return
	}

	if newTags, err = c.driver.TagSnapshot(args[0], args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Printf("%v TAGS: %v\n", args[0], newTags)
}

// serviced snapshot removetags SNAPSHOTID [TAG-NAME ...]
func (c *ServicedCli) cmdSnapshotRemoveTags(ctx *cli.Context) {
	args := ctx.Args()

	var (
		newTags []string
		err      error
	)

	if len(args) < 1{
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "removetags")
		return
	} else if len(args) == 1 {
		//remove all tags
		if err = c.driver.RemoveAllSnapshotTags(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		} 
	} else {
		//remove specified tags
		if newTags, err = c.driver.RemoveSnapshotTags(args[0], args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}

	fmt.Printf("%v TAGS: %v\n", args[0], newTags)
}
