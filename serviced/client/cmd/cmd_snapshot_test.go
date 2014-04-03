package cmd

func ExampleCmdSnapshotList() {
	Run("serviced", "snapshot", "list")

	// Output:
	// serviced snapshot list [SERVICEID]
}

func ExampleCmdSnapshotAdd() {
	Run("serviced", "snapshot", "add")

	// Output:
	// serviced snapshot add SERVICEID
}

func ExampleCmdSnapshotRemove() {
	Run("serviced", "snapshot", "remove")
	Run("serviced", "snapshot", "rm")

	// Output:
	// serviced snapshot remove SNAPSHOTID
	// serviced snapshot remove SNAPSHOTID
}

func ExampleCmdSnapshotCommit() {
	Run("serviced", "snapshot", "commit")

	// Output:
	// serviced snapshot commit DOCKERID
}

func ExampleCmdSnapshotRollback() {
	Run("serviced", "snapshot", "rollback")

	// Output:
	// serviced snapshot rollback SNAPSHOTID
}
