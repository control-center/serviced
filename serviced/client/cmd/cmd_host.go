package cmd

import (
	"fmt"

	"github.com/codegangsta/cli"
)

// CmdHostList is the command-line interaction for serviced host list
// usage: serviced host list
func CmdHostList(c *cli.Context) {
	fmt.Println("serviced host list")
}

// CmdHostAdd is the command-line interaction for serviced host add
// usage: serviced host add HOST[:PORT] RESOURCE_POOL [[--ip IP] ...]
func CmdHostAdd(c *cli.Context) {
	fmt.Println("serviced host add HOST[:PORT] RESOURCE_POOL [[--ip IP] ...]")
}

// CmdHostRemove is the command-line interaction for serviced host remove
// usage: serviced host remove HOSTID
func CmdHostRemove(c *cli.Context) {
	fmt.Println("serviced host remove HOSTID")
}