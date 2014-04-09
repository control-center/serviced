package cmd

import (
	"github.com/zenoss/serviced/serviced/client/api"
)

func ExampleServicedCli_cmdProxy() {
	New(api.New()).Run([]string{"serviced", "proxy"})

	// Output:
	// serviced proxy ...
}
