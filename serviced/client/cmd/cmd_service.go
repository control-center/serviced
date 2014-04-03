package cmd

import (
	"fmt"

	"github.com/codegangsta/cli"
)

// CmdServiceList is the command-line interaction for serviced service list
// usage: serviced service list
func CmdServiceList(c *cli.Context) {
	fmt.Println("serviced service list")
}

// CmdServiceAdd is the command-line interaction for serviced service add
// usage: serviced service add [-p PORT] [-q REMOTE_PORT] NAME POOLID IMAGEID COMMAND
func CmdServiceAdd(c *cli.Context) {
	fmt.Println("serviced service add [-p PORT] [-q REMOTE_PORT] NAME POOLID IMAGEID COMMAND")
}

// CmdServiceRemove is the command-line interaction for serviced service remove
// usage: serviced service remove SERVICEID
func CmdServiceRemove(c *cli.Context) {
	fmt.Println("serviced service remove SERVICEID")
}

// CmdServiceEdit is the command-line interaction for serviced service edit
// usage: serviced service edit SERVICEID
func CmdServiceEdit(c *cli.Context) {
	fmt.Println("serviced service edit SERVICEID")
}

// CmdServiceAutoIPs is the command-line interaction for serviced service auto-assign-ips
// usage: serviced service auto-assign-ips SERVICEID
func CmdServiceAutoIPs(c *cli.Context) {
	fmt.Println("serviced service auto-assign-ips SERVICEID")
}

// CmdServiceManualIPs is the command-line interaction for serviced service manual-assign-ips
// usage: serviced service manual-assign-ips SERVICEID IPADDRESS
func CmdServiceManualIPs(c *cli.Context) {
	fmt.Println("serviced service manual-assign-ips SERVICEID IPADDRESS")
}

// CmdServiceStart is the command-line interaction for serviced service start
// usage: serviced service start SERVICEID
func CmdServiceStart(c *cli.Context) {
	fmt.Println("serviced service start SERVICEID")
}

// CmdServiceStop is the command-line interaction for serviced service stop
// usage: serviced service stop SERVICEID
func CmdServiceStop(c *cli.Context) {
	fmt.Println("serviced service stop SERVICEID")
}

// CmdServiceRestart is the command-line interaction for serviced service restart
// usage: serviced service restart SERVICEID
func CmdServiceRestart(c *cli.Context) {
	fmt.Println("serviced service restart SERVICEID")
}

// CmdServiceShell is the command-line interaction for serviced service shell
// usage: serviced service shell SERVICEID [-rm=false] [-i] COMMAND [ARGS ...]
func CmdServiceShell(c *cli.Context) {
	fmt.Println("serviced service shell SERVICEID [-rm=false] [-i] COMMAND [ARGS ...]")
}

// CmdServiceListCmds is the command-line interaction for serviced service list-commands
// usage: serviced service list-commands SERVICEID
func CmdServiceListCmds(c *cli.Context) {
	fmt.Println("serviced service list-commands SERVICEID")
}

// CmdServiceRun is the command-line interaction for serviced service run
// usage: serviced service run SERVICEID PROGRAM [ARGS ...]
func CmdServiceRun(c *cli.Context) {
	fmt.Println("serviced service run SERVICEID PROGRAM [ARGS ...]")
}