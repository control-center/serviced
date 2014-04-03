package cmd

func ExampleCmdPoolList() {
	Run("serviced", "pool", "list")

	// Output:
	// serviced pool list
}

func ExampleCmdPoolAdd() {
	Run("serviced", "pool", "add")

	// Output:
	// serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY
}

func ExampleCmdPoolRemove() {
	Run("serviced", "pool", "remove")
	Run("serviced", "pool", "rm")

	// Output:
	// serviced pool remove POOLID
	// serviced pool remove POOLID
}

func ExampleCmdPoolListIPs() {
	Run("serviced", "pool", "list-ips")

	// Output:
	// serviced pool list-ips POOLID
}
