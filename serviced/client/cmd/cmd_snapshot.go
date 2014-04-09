package cmd

import (
	"fmt"

	"github.com/zenoss/cli"
)

// initSnapshot is the initializer for serviced snapshot
func (c *ServicedCli) initSnapshot() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:   "snapshot",
		Usage:  "Administers environment snapshots.",
		Action: cmdDefault,
	})
	cmd.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "Lists all snapshots.",
			Action: c.cmdSnapshotList,
		}, {
			Name:   "add",
			Usage:  "Take a snapshot of an existing service.",
			Action: c.cmdSnapshotAdd,
		}, {
			Name:      "remove",
			ShortName: "rm",
			Usage:     "Removes an existing snapshot.",
			Action:    c.cmdSnapshotRemove,
		}, {
			Name:   "commit",
			Usage:  "Snapshots and commits a given service instance",
			Action: c.cmdSnapshotCommit,
		}, {
			Name:   "rollback",
			Usage:  "Restores the environment to the state of the given snapshot.",
			Action: c.cmdSnapshotRollback,
		},
	}
}

// cmdSnapshotList is the command-line interaction for serviced snapshot list
// usage: serviced snapshot list [SERVICEID]
func (c *ServicedCli) cmdSnapshotList(ctx *cli.Context) {
	fmt.Println("serviced snapshot list [SERVICEID]")
}

// cmdSnapshotAdd is the command-line interaction for serviced snapshot add
// usage: serviced snapshot add SERVICEID
func (c *ServicedCli) cmdSnapshotAdd(ctx *cli.Context) {
	fmt.Println("serviced snapshot add SERVICEID")
}

// cmdSnapshotRemove is the command-line interaction for serviced snapshot remove
// usage: serviced snapshot remove SNAPSHOTID
func (c *ServicedCli) cmdSnapshotRemove(ctx *cli.Context) {
	fmt.Println("serviced snapshot remove SNAPSHOTID")
}

// cmdSnapshotCommit is the command-line interaction for serviced snapshot commit
// usage: serviced snapshot commit DOCKERID
func (c *ServicedCli) cmdSnapshotCommit(ctx *cli.Context) {
	fmt.Println("serviced snapshot commit DOCKERID")
}

// cmdSnapshotRollback is the command-line interaction for serviced snapshot rollback
// usage: serviced snapshot rollback SNAPSHOTID
func (c *ServicedCli) cmdSnapshotRollback(ctx *cli.Context) {
	fmt.Println("serviced snapshot rollback SNAPSHOTID")
}