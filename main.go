package main


import (
	"os"

	"github.com/zenoss/serviced/servicedversion"
	"github.com/zenoss/serviced/cli/api"
	"github.com/zenoss/serviced/cli/cmd"
)

var Version string
var Date string
var Gitcommit string
var Gitbranch string

func main() {
	servicedversion.Version = Version
	servicedversion.Date = Date
	servicedversion.Gitcommit = Gitcommit
	servicedversion.Gitbranch = Gitbranch
	cmd.New(api.New()).Run(os.Args)
}
