package cmd

func ExampleCmdServiceList() {
	Run("serviced", "service", "list")

	// Output:
	// serviced service list
}

func ExampleCmdServiceAdd() {
	Run("serviced", "service", "add")

	// Output:
	// serviced service add [-p PORT] [-q REMOTE_PORT] NAME POOLID IMAGEID COMMAND
}

func ExampleCmdServiceRemove() {
	Run("serviced", "service", "remove")
	Run("serviced", "service", "rm")

	// Output:
	// serviced service remove SERVICEID
	// serviced service remove SERVICEID
}

func ExampleCmdServiceEdit() {
	Run("serviced", "service", "edit")

	// Output:
	// serviced service edit SERVICEID
}

func ExampleCmdServiceAutoIPs() {
	Run("serviced", "service", "auto-assign-ips")

	// Output:
	// serviced service auto-assign-ips SERVICEID
}

func ExampleCmdServiceManualIPs() {
	Run("serviced", "service", "manual-assign-ips")

	// Output:
	// serviced service manual-assign-ips SERVICEID IPADDRESS
}

func ExampleCmdServiceStart() {
	Run("serviced", "service", "start")

	// Output:
	// serviced service start SERVICEID
}

func ExampleCmdServiceStop() {
	Run("serviced", "service", "stop")

	// Output:
	// serviced service stop SERVICEID
}

func ExampleCmdServiceRestart() {
	Run("serviced", "service", "restart")

	// Output:
	// serviced service restart SERVICEID
}

func ExampleCmdServiceShell() {
	Run("serviced", "service", "shell")

	// Output:
	// serviced service shell SERVICEID [-rm=false] [-i] COMMAND [ARGS ...]
}

func ExampleCmdServiceListCmds() {
	Run("serviced", "service", "list-commands")

	// Output:
	// serviced service list-commands SERVICEID
}

func ExampleCmdServiceRun() {
	Run("serviced", "service", "run")

	// Output:
	// serviced service run SERVICEID PROGRAM [ARGS ...]
}
