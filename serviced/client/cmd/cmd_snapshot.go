package cmd

import (
	"fmt"

	"github.com/codegangsta/cli"
)

// CmdSnapshotList is the command-line interaction for serviced snapshot list
// usage: serviced snapshot list [SERVICEID]
func CmdSnapshotList(c *cli.Context) {
	fmt.Println("serviced snapshot list [SERVICEID]")
}

// CmdSnapshotAdd is the command-line interaction for serviced snapshot add
// usage: serviced snapshot add SERVICEID
func CmdSnapshotAdd(c *cli.Context) {
	fmt.Println("serviced snapshot add SERVICEID")
}

// CmdSnapshotRemove is the command-line interaction for serviced snapshot remove
// usage: serviced snapshot remove SNAPSHOTID
func CmdSnapshotRemove(c *cli.Context) {
	fmt.Println("serviced snapshot remove SNAPSHOTID")
}

// CmdSnapshotCommit is the command-line interaction for serviced snapshot commit
// usage: serviced snapshot commit DOCKERID
func CmdSnapshotCommit(c *cli.Context) {
	fmt.Println("serviced snapshot commit DOCKERID")
}

// CmdSnapshotRollback is the command-line interaction for serviced snapshot rollback
// usage: serviced snapshot rollback SNAPSHOTID
func CmdSnapshotRollback(c *cli.Context) {
	fmt.Println("serviced snapshot rollback SNAPSHOTID")
}