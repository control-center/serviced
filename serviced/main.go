package main

import (
	"os"

	"github.com/zenoss/serviced/serviced/api"
	"github.com/zenoss/serviced/serviced/cmd"
)

func main() {
	cmd.New(api.New()).Run(os.Args)
}