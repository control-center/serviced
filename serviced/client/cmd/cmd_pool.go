package cmd

import (
	"fmt"

	"github.com/codegangsta/cli"
)

// CmdPoolList is the command-line interaction for serviced pool list
// usage: serviced pool list
func CmdPoolList(c *cli.Context) {
	fmt.Println("serviced pool list")
}

// CmdPoolAdd is the command-line interaction for serviced pool add
// usage: serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY
func CmdPoolAdd(c *cli.Context) {
	fmt.Println("serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY")
}

// CmdPoolRemove is the command-line interaction for serviced pool remove
// usage: serviced pool remove POOLID
func CmdPoolRemove(c *cli.Context) {
	fmt.Println("serviced pool remove POOLID")
}

// CmdPoolListIPs is the command-line interaction for serviced pool list-ips
// usage: serviced pool list-ips POOLID
func CmdPoolListIPs(c *cli.Context) {
	fmt.Println("serviced pool list-ips POOLID")
}