package cmd

import (
	"github.com/zenoss/serviced/serviced/client/api"
)

func ExampleServicedCli_cmdProxy() {
	New(api.New()).Run("serviced", "proxy")

	// Output:
	// serviced proxy ...
}
