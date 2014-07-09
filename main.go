package main

import (
	"os"
	"github.com/zenoss/serviced/servicedversion"
	"github.com/zenoss/serviced/cli/api"
	"github.com/zenoss/serviced/cli/cmd"
)

var Version string
var Gitcommit string

func main() {
	servicedversion.Version = Version
	servicedversion.Gitcommit = Gitcommit
	cmd.New(api.New()).Run(os.Args)
}
