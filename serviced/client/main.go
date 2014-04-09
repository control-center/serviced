package main

import (
	"os"

	"github.com/zenoss/serviced/serviced/client/api"
	"github.com/zenoss/serviced/serviced/client/cmd"
)

func main() {
	cmd.New(api.New()).Run(os.Args)
}