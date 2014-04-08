package cmd

import (
	"github.com/zenoss/serviced/serviced/client/api"
)

func ExampleServicedCli_cmdHostList() {
	New(api.New()).Run("serviced", "host", "list")

	// Output:
	// serviced host list
}

func ExampleServicedCli_cmdHostAdd() {
	New(api.New()).Run("serviced", "host", "add")

	// Output:
	// serviced host add HOST[:PORT] RESOURCE_POOL [[--ip IP] ...]
}

func ExampleServicedCli_cmdHostRemove() {
	New(api.New()).Run("serviced", "host", "remove")
	New(api.New()).Run("serviced", "host", "rm")

	// Output:
	// serviced host remove HOSTID
	// serviced host remove HOSTID
}
