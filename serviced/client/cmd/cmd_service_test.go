package cmd

import (
	"github.com/zenoss/serviced/serviced/client/api"
)

func ExampleServicedCli_cmdServiceList() {
	New(api.New()).Run([]string{"serviced", "service", "list"})

	// Output:
	// serviced service list
}

func ExampleServicedCli_cmdServiceAdd() {
	New(api.New()).Run([]string{"serviced", "service", "add"})

	// Output:
	// serviced service add [-p PORT] [-q REMOTE_PORT] NAME POOLID IMAGEID COMMAND
}

func ExampleServicedCli_cmdServiceRemove() {
	New(api.New()).Run([]string{"serviced", "service", "remove"})
	New(api.New()).Run([]string{"serviced", "service", "rm"})

	// Output:
	// serviced service remove SERVICEID
	// serviced service remove SERVICEID
}

func ExampleServicedCli_cmdServiceEdit() {
	New(api.New()).Run([]string{"serviced", "service", "edit"})

	// Output:
	// serviced service edit SERVICEID
}

func ExampleServicedCli_cmdServiceAutoIPs() {
	New(api.New()).Run([]string{"serviced", "service", "auto-assign-ips"})

	// Output:
	// serviced service auto-assign-ips SERVICEID
}

func ExampleServicedCli_cmdServiceManualIPs() {
	New(api.New()).Run([]string{"serviced", "service", "manual-assign-ips"})

	// Output:
	// serviced service manual-assign-ips SERVICEID IPADDRESS
}

func ExampleServicedCli_cmdServiceStart() {
	New(api.New()).Run([]string{"serviced", "service", "start"})

	// Output:
	// serviced service start SERVICEID
}

func ExampleServicedCli_cmdServiceStop() {
	New(api.New()).Run([]string{"serviced", "service", "stop"})

	// Output:
	// serviced service stop SERVICEID
}

func ExampleServicedCli_cmdServiceRestart() {
	New(api.New()).Run([]string{"serviced", "service", "restart"})

	// Output:
	// serviced service restart SERVICEID
}

func ExampleServicedCli_cmdServiceShell() {
	New(api.New()).Run([]string{"serviced", "service", "shell"})

	// Output:
	// serviced service shell SERVICEID [-rm=false] [-i] COMMAND [ARGS ...]
}

func ExampleServicedCli_cmdServiceListCmds() {
	New(api.New()).Run([]string{"serviced", "service", "list-commands"})

	// Output:
	// serviced service list-commands SERVICEID
}

func ExampleServicedCli_cmdServiceRun() {
	New(api.New()).Run([]string{"serviced", "service", "run"})

	// Output:
	// serviced service run SERVICEID PROGRAM [ARGS ...]
}
