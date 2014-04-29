package main

import (
	"os"

	"github.com/zenoss/serviced/cli/api"
	"github.com/zenoss/serviced/cli/cmd"
)

func main() {
	cmd.New(api.New()).Run(os.Args)
}
