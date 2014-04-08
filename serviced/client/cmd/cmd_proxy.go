package cmd

import (
	"fmt"

	"github.com/zenoss/cli"
)

// initProxy is the initializer for serviced proxy.
func (c *ServicedCli) initProxy() {
	cmd := cli.Command{
		Name:   "proxy",
		Usage:  "Starts a proxy on the foreground",
		Action: c.cmdProxy,
	}
	c.app.Commands = append(c.app.Commands, cmd)
}

// cmdProxy is the command-line interaction for serviced proxy.
func (c *ServicedCli) cmdProxy(ctx *cli.Context) {
	fmt.Println("serviced proxy ... ")
}