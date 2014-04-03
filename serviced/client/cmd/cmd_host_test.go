package cmd

func ExampleCmdHostList() {
	Run("serviced", "host", "list")

	// Output:
	// serviced host list
}

func ExampleCmdHostAdd() {
	Run("serviced", "host", "add")

	// Output:
	// serviced host add HOST[:PORT] RESOURCE_POOL [[--ip IP] ...]
}

func ExampleCmdHostRemove() {
	Run("serviced", "host", "remove")
	Run("serviced", "host", "rm")

	// Output:
	// serviced host remove HOSTID
	// serviced host remove HOSTID
}
