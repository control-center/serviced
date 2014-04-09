package cmd

import (
	"github.com/zenoss/serviced/serviced/client/api"
)

func ExampleServicedCli_cmdPoolList() {
	New(api.New()).Run([]string{"serviced", "pool", "list"})

	// Output:
	// serviced pool list
}

func ExampleServicedCli_cmdPoolAdd() {
	New(api.New()).Run([]string{"serviced", "pool", "add"})

	// Output:
	// serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY
}

func ExampleServicedCli_cmdPoolRemove() {
	New(api.New()).Run([]string{"serviced", "pool", "remove"})
	New(api.New()).Run([]string{"serviced", "pool", "rm"})

	// Output:
	// serviced pool remove POOLID
	// serviced pool remove POOLID
}

func ExampleServicedCli_cmdPoolListIPs() {
	New(api.New()).Run([]string{"serviced", "pool", "list-ips"})

	// Output:
	// serviced pool list-ips POOLID
}
