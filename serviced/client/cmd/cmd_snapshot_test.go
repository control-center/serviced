package cmd

import (
	"github.com/zenoss/serviced/serviced/client/api"
)

func ExampleServicedCli_cmdSnapshotList() {
	New(api.New()).Run("serviced", "snapshot", "list")

	// Output:
	// serviced snapshot list [SERVICEID]
}

func ExampleServicedCli_cmdSnapshotAdd() {
	New(api.New()).Run("serviced", "snapshot", "add")

	// Output:
	// serviced snapshot add SERVICEID
}

func ExampleServicedCli_cmdSnapshotRemove() {
	New(api.New()).Run("serviced", "snapshot", "remove")
	New(api.New()).Run("serviced", "snapshot", "rm")

	// Output:
	// serviced snapshot remove SNAPSHOTID
	// serviced snapshot remove SNAPSHOTID
}

func ExampleServicedCli_cmdSnapshotCommit() {
	New(api.New()).Run("serviced", "snapshot", "commit")

	// Output:
	// serviced snapshot commit DOCKERID
}

func ExampleServicedCli_cmdSnapshotRollback() {
	New(api.New()).Run("serviced", "snapshot", "rollback")

	// Output:
	// serviced snapshot rollback SNAPSHOTID
}
