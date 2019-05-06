// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

var unstartedTime = time.Date(1999, 12, 31, 23, 59, 0, 0, time.UTC)

// Initializer for serviced service subcommands
func (c *ServicedCli) initService() {

	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "service",
		Usage:       "Administers services",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Lists all services",
				Description:  "serviced service list [SERVICEID]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceList,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "Show JSON format",
					},
					cli.BoolFlag{
						Name:  "ascii, a",
						Usage: "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)",
					},
					cli.StringFlag{
						Name:  "format",
						Value: "",
						Usage: "format the output using the given go template",
					},
					cli.StringFlag{
						Name:  "show-fields",
						Value: "Name,ServiceID,Inst,ImageID,Pool,DState,Launch,DepID",
						Usage: "Comma-delimited list describing which fields to display",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:        "status",
				Usage:       "Displays the status of deployed services",
				Description: "serviced service status { SERVICEID | SERVICENAME | [POOL/]...PARENTNAME.../SERVICENAME }",
				Action:      c.cmdServiceStatus,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "ascii, a",
						Usage: "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)",
					},
					cli.StringFlag{
						Name:  "show-fields",
						Value: "Name,ServiceID,Status,HC Fail,Healthcheck,Healthcheck Status,Uptime,RAM,Cur/Max/Avg,Hostname,InSync,DockerID",
						Usage: "Comma-delimited list describing which fields to display",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:        "add",
				Usage:       "Adds a new service",
				Description: "serviced service add NAME IMAGEID COMMAND",
				Action:      c.cmdServiceAdd,
				Flags: []cli.Flag{
					cli.GenericFlag{
						Name:  "p",
						Value: &api.PortMap{},
						Usage: "Expose a port for this service (e.g. -p tcp:3306:mysql)",
					},
					cli.GenericFlag{
						Name:  "q",
						Value: &api.PortMap{},
						Usage: "Map a remote service port (e.g. -q tcp:3306:mysql)",
					},
					cli.StringFlag{
						Name:  "parent-id",
						Value: "",
						Usage: "Parent service ID for which this service relates",
					},
				},
			}, {
				Name:        "clone",
				Usage:       "Clones a new service",
				Description: "serviced service clone { SERVICEID | SERVICENAME | [DEPLOYMENTID/]...PARENTNAME.../SERVICENAME }",
				Action:      c.cmdServiceClone,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "suffix",
						Value: "",
						Usage: "name to append to service name, volumes, endpoints",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing service",
				Description:  "serviced service remove SERVICEID",
				BashComplete: c.printServicesAll,
				Action:       c.cmdServiceRemove,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "edit",
				Usage:        "Edits an existing service in a text editor",
				Description:  "serviced service edit SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceEdit,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "editor, e",
						Value: os.Getenv("EDITOR"),
						Usage: "Editor used to update the service definition",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "assign-ip",
				Usage:        "Assigns an IP address to a service's endpoints requiring an explicit IP address",
				Description:  "serviced service assign-ip SERVICEID [IPADDRESS]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceAssignIP,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "start",
				Usage:        "Starts one or more services",
				Description:  "serviced service start SERVICEID ...",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStart,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
					},
					cli.BoolFlag{
						Name:  "sync, s",
						Usage: "Schedules services synchronously",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "restart",
				Usage:        "Restarts one or more services",
				Description:  "serviced service restart { SERVICEID | INSTANCEID } ...",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceRestart,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
					},
					cli.BoolFlag{
						Name:  "sync, s",
						Usage: "Schedules services synchronously",
					},
					cli.BoolFlag{
						Name:  "rebalance",
						Usage: "Stops all instances before restarting them, instead of performing a rolling restart",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "stop",
				Usage:        "Stops one or more services",
				Description:  "serviced service stop SERVICEID ...",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStop,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
					},
					cli.BoolFlag{
						Name:  "sync, s",
						Usage: "Schedules services synchronously",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "pause",
				Usage:        "Pauses one or more services",
				Description:  "serviced service pause SERVICEID ...",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServicePause,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
					},
					cli.BoolFlag{
						Name:  "sync, s",
						Usage: "Schedules services synchronously",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "shell",
				Usage:        "Starts a service instance",
				Description:  "serviced service shell SERVICEID [COMMAND]",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceShell,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "saveas, s",
						Value: "",
						Usage: "saves the service instance with the given name",
					},
					cli.BoolFlag{
						Name:  "interactive, i",
						Usage: "runs the service instance as a tty",
					},
					cli.StringSliceFlag{
						Name:  "mount",
						Value: &cli.StringSlice{},
						Usage: "bind mount: HOST_PATH[,CONTAINER_PATH]",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "run",
				Usage:        "Runs a service command in a service instance",
				Description:  "serviced service run SERVICEID COMMAND [ARGS]",
				BashComplete: c.printServiceRun,
				Before:       c.cmdServiceRun,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "interactive, i",
						Usage: "runs the service instance as a tty",
					},
					cli.BoolFlag{
						Name:  "logtostderr",
						Usage: "enable/disable detailed serviced run logging (false by default)",
					},
					cli.BoolTFlag{
						Name:  "logstash",
						Usage: "enable/disable log stash (true by default)",
					},
					cli.StringFlag{
						Name:  "logstash-settle-time",
						Value: "5s",
						Usage: "time duration to wait for logstash to flush log messages before closing",
					},
					cli.StringSliceFlag{
						Name:  "mount",
						Value: &cli.StringSlice{},
						Usage: "bind mount: HOST_PATH[,CONTAINER_PATH]",
					},
					cli.StringFlag{
						Name:  "user",
						Value: "",
						Usage: "container username used to run command",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "attach",
				Usage:        "Run an arbitrary command in a running service container",
				Description:  "serviced service attach { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME/INSTANCE } [COMMAND]",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceAttach,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "action",
				Usage:        "Run a predefined action in a running service container",
				Description:  "serviced service action { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME/INSTANCE } ACTION",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceAction,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "logs",
				Usage:        "Output the logs of a running service container - calls docker logs",
				Description:  "serviced service logs { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME/INSTANCE }",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceLogs,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "list-snapshots",
				Usage:        "Lists the snapshots for a service",
				Description:  "serviced service list-snapshots SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceListSnapshots,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "show-tags, t",
						Usage: "shows the tags associated with each snapshot",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "snapshot",
				Usage:        "Takes a snapshot of the service",
				Description:  "serviced service snapshot SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceSnapshot,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "description, d",
						Value: "",
						Usage: "a description of the snapshot",
					},
					cli.StringFlag{
						Name:  "tag, t",
						Value: "",
						Usage: "a unique tag for the snapshot",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:         "endpoints",
				Usage:        "List the endpoints defined for the service",
				Description:  "serviced service endpoints SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceEndpoints,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "imports, i",
						Usage: "include only imported endpoints",
					},
					cli.BoolFlag{
						Name:  "all, a",
						Usage: "include all endpoints (imports and exports)",
					},
					cli.BoolFlag{
						Name:  "verify, v",
						Usage: "verify endpoints",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			}, {
				Name:        "public-endpoints",
				Usage:       "Manage public endpoints for a service",
				Description: "serviced service public-endpoints",
				Subcommands: []cli.Command{
					{
						Name:        "list",
						Usage:       "Lists public endpoints for a service",
						Description: "serviced service public-endpoints list [SERVICEID] [ENDPOINTNAME]",
						Action:      c.cmdPublicEndpointsListAll,
						Flags: []cli.Flag{
							cli.BoolFlag{
								Name:  "ascii, a",
								Usage: "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)",
							},
							cli.BoolFlag{
								Name:  "ports",
								Usage: "Show port public endpoints",
							},
							cli.BoolFlag{
								Name:  "vhosts",
								Usage: "Show vhost public endpoints",
							},
							cli.StringFlag{
								Name:  "show-fields",
								Value: "Service,ServiceID,Endpoint,Type,Protocol,Name,Enabled",
								Usage: "Comma-delimited list describing which fields to display",
							},
							cli.BoolFlag{
								Name:  "verbose, v",
								Usage: "Show JSON format",
							},
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
					{
						Name:        "port",
						Usage:       "Manages port public endpoints for a service",
						Description: "serviced service public-endpoints port",
						Subcommands: []cli.Command{
							{
								Name:        "list",
								Usage:       "List port public endpoints for a service",
								Description: "serviced service public-endpoints port list [SERVICEID] [ENDPOINTNAME]",
								Action:      c.cmdPublicEndpointsPortList,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "ascii, a",
										Usage: "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)",
									},
									cli.StringFlag{
										Name:  "show-fields",
										Value: "Service,ServiceID,Endpoint,Type,Protocol,Name,Enabled",
										Usage: "Comma-delimited list describing which fields to display",
									},
									cli.BoolFlag{
										Name:  "verbose, v",
										Usage: "Show JSON format",
									},
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
							{
								Name:        "add",
								Usage:       "Add a port public endpoint to a service",
								Description: "serviced service public-endpoints port add <SERVICEID> <ENDPOINTNAME> <PORTADDR> <PROTOCOL> <ENABLED>",
								Action:      c.cmdPublicEndpointsPortAdd,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "restart, r",
										Usage: "Restart the service after adding the port if the service is currently running",
									},
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
							{
								Name:        "remove",
								ShortName:   "rm",
								Usage:       "Remove a port public endpoint from a service",
								Description: "serviced service public-endpoints port remove <SERVICEID> <ENDPOINTNAME> <PORTADDR>",
								Action:      c.cmdPublicEndpointsPortRemove,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
							{
								Name:        "enable",
								Usage:       "Enable/Disable a port public endpoint for a service",
								Description: "serviced service public-endpoints port enable <SERVICEID> <ENDPOINTNAME> <PORTADDR> true|false",
								Action:      c.cmdPublicEndpointsPortEnable,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
						},
					},
					{
						Name:        "vhost",
						Usage:       "Manages vhost public endpoints for a service",
						Description: "serviced service public-endpoints vhost",
						Subcommands: []cli.Command{
							{
								Name:        "list",
								Usage:       "List vhost public endpoints for a service",
								Description: "serviced service public-endpoints vhost list [SERVICEID] [ENDPOINTNAME]",
								Action:      c.cmdPublicEndpointsVHostList,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "ascii, a",
										Usage: "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)",
									},
									cli.StringFlag{
										Name:  "show-fields",
										Value: "Service,ServiceID,Endpoint,Type,Protocol,Name,Enabled",
										Usage: "Comma-delimited list describing which fields to display",
									},
									cli.BoolFlag{
										Name:  "verbose, v",
										Usage: "Show JSON format",
									},
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
							{
								Name:        "add",
								Usage:       "Add a vhost public endpoint to a service",
								Description: "serviced service public-endpoints vhost add <SERVICEID> <ENDPOINTNAME> <VHOST> <ENABLED>",
								Action:      c.cmdPublicEndpointsVHostAdd,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
							{
								Name:        "remove",
								ShortName:   "rm",
								Usage:       "Remove a vhost public endpoint from a service",
								Description: "serviced service public-endpoints vhost remove <SERVICEID> <ENDPOINTNAME> <VHOST>",
								Action:      c.cmdPublicEndpointsVHostRemove,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
							{
								Name:        "enable",
								Usage:       "Enable/Disable a vhost public endpoint for a service",
								Description: "serviced service public-endpoints vhost enable <SERVICEID> <ENDPOINTNAME> <VHOST> true|false",
								Action:      c.cmdPublicEndpointsVHostEnable,
								Flags: []cli.Flag{
									cli.BoolFlag{
										Name:  "no-prefix-match, np",
										Usage: "Make SERVICEID matches on name strict 'ends with' matches",
									},
								},
							},
						},
					},
				},
			},
			{
				Name:         "clear-emergency",
				Usage:        "Clears the 'emergency shutdown' state for a service and all child services",
				Description:  "serviced service clear-emergency { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME }",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceClearEmergency,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			},
			{
				Name:         "remove-ip",
				Usage:        "Remove the IP assignment of a service's endpoints",
				Description:  "serviced service remove-ip <SERVICEID> <ENDPOINTNAME>",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceRemoveIP,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			},
			{
				Name:         "set-ip",
				Usage:        "Setting an IP address to a service's endpoints requiring an explicit IP address. If ip is not provided it does an automatic assignment",
				Description:  "serviced service set-ip <SERVICEID> <ENDPOINTNAME> [ADDRESS] [--port=PORT] [--proto=PROTOCOL]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceSetIP,
				Flags: []cli.Flag{
					cli.IntFlag{
						Name:  "port",
						Usage: "determine the port your service will use",
					},
					cli.StringFlag{
						Name:  "proto",
						Usage: "determine the port protocol your service will use",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			},
			{
				Name:        "tune",
				Usage:       "Adjust launch mode, instance count, RAM commitment, or RAM threshold for a service",
				Description: "serviced service tune SERVICEID",
				Action:      c.cmdServiceTune,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "launchMode",
						Usage: "Launch mode for this service (auto, manual)",
					},
					cli.IntFlag{
						Name:  "instances",
						Usage: "Instance count for this service",
					},
					cli.StringFlag{
						Name:  "ramCommitment",
						Usage: "RAM Commitment for this service",
					},
					cli.StringFlag{
						Name:  "ramThreshold",
						Usage: "RAM Threshold for this service",
					},
					cli.BoolFlag{
						Name:  "no-prefix-match, np",
						Usage: "Make SERVICEID matches on name strict 'ends with' matches",
					},
				},
			},
			{
				Name:        "config",
				Usage:       "Manage config files for services",
				Description: "serviced service config",
				Subcommands: []cli.Command{
					{
						Name:        "list",
						Usage:       "List all config files for a given service, or the contents of one named file",
						Description: "serviced service config list SERVICEID [FILENAME]",
						Action:      c.cmdServiceConfigList,
						Flags: []cli.Flag{
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
					{
						Name:        "edit",
						Usage:       "Edit one config file for a given service",
						Description: "serviced service config edit SERVICEID FILENAME",
						Action:      c.cmdServiceConfigEdit,
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:  "editor, e",
								Value: os.Getenv("EDITOR"),
								Usage: "Editor used to update the config file",
							},
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
				},
			},
			{
				Name:        "variable",
				Usage:       "Manage service config variables",
				Description: "serviced service variable",
				Subcommands: []cli.Command{
					{
						Name:        "list",
						Usage:       "List one or all config variables and their values for a given service",
						Description: "serviced service variable list SERVICEID",
						Action:      c.cmdServiceVariableList,
						Flags: []cli.Flag{
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
					{
						Name:        "get",
						Usage:       "Find the value of a config variable for a service",
						Description: "serviced service variable get SERVICEID VARIABLE",
						Action:      c.cmdServiceVariableGet,
						Flags: []cli.Flag{
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
					{
						Name:        "set",
						Usage:       "Add or update one variable's value for a given service",
						Description: "serviced service variable set SERVICEID VARIABLE VALUE",
						Action:      c.cmdServiceVariableSet,
						Flags: []cli.Flag{
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
					{
						Name:        "unset",
						Usage:       "Remove a variable from a given service",
						Description: "serviced service variable unset SERVICEID VARIABLE",
						Action:      c.cmdServiceVariableUnset,
						Flags: []cli.Flag{
							cli.BoolFlag{
								Name:  "no-prefix-match, np",
								Usage: "Make SERVICEID matches on name strict 'ends with' matches",
							},
						},
					},
				},
			},
		},
	})
}

// Returns a list of all the available service IDs
func (c *ServicedCli) services() (data []string) {
	svcs, err := c.driver.GetAllServiceDetails()
	if err != nil || svcs == nil || len(svcs) == 0 {
		c.exit(1)
		return
	}

	data = make([]string, len(svcs))
	for i, s := range svcs {
		data[i] = s.ID
	}

	return
}

// Returns a list of runnable commands for a particular service
func (c *ServicedCli) serviceCommands(id string) (data []string) {
	svc, err := c.driver.GetService(id)
	if err != nil || svc == nil {
		c.exit(1)
		return
	}

	data = make([]string, len(svc.Commands))
	i := 0
	for r := range svc.Commands {
		data[i] = r
		i++
	}

	return
}

// Returns a list of actionable commands for a particular service
func (c *ServicedCli) serviceActions(id string) (data []string) {
	svc, err := c.driver.GetService(id)
	if err != nil || svc == nil {
		c.exit(1)
		return
	}

	data = make([]string, len(svc.Actions))
	i := 0
	for a := range svc.Actions {
		data[i] = a
		i++
	}

	return
}

// Bash-completion command that prints a list of available services as the
// first argument
func (c *ServicedCli) printServicesFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}
	fmt.Println(strings.Join(c.services(), "\n"))
}

// Bash-completion command that prints a list of available services as all
// arguments
func (c *ServicedCli) printServicesAll(ctx *cli.Context) {
	args := ctx.Args()
	svcs := c.services()

	// If arg is a service don't add to the list
	for _, s := range svcs {
		for _, a := range args {
			if s == a {
				goto next
			}
		}
		fmt.Println(s)
	next:
	}
	return
}

func (c *ServicedCli) printHelpForRun(svc *service.Service, command string) (returncode int) {
	var (
		found             bool
		availablecommands []string
	)
	for commandname := range svc.Commands {
		availablecommands = append(availablecommands, commandname)
		if commandname == command {
			found = true
		}
	}

	sort.Strings(availablecommands)
	if command == "help" {
		fmt.Printf("Available commands for %v:\n", svc.Name)
		for _, commandname := range availablecommands {
			fmt.Printf("    %-20v  %v\n", commandname, svc.Commands[commandname].Description)
		}
		if len(availablecommands) == 0 {
			fmt.Println("    No commands available.")
		}
		return 0

	} else if !found {
		fmt.Printf("Command %#v not available.\n", command)
		fmt.Printf("Available commands for %v:\n", svc.Name)
		for _, commandname := range availablecommands {
			fmt.Printf("    %-20v  %v\n", commandname, svc.Commands[commandname].Description)
		}
		if len(availablecommands) == 0 {
			fmt.Println("    No commands available.")
		}
		c.exit(1)
		return 1

	}
	return -1
}

// Bash-completion command that completes the service ID as the first argument
// and runnable commands as the second argument
func (c *ServicedCli) printServiceRun(ctx *cli.Context) {
	var output []string

	args := ctx.Args()
	switch len(args) {
	case 0:
		output = c.services()
	case 1:
		output = c.serviceCommands(args[0])
	}
	fmt.Println(strings.Join(output, "\n"))
}

// buildServicePaths returns a map where map[service.ID] = fullpath
func (c *ServicedCli) buildServicePaths(svcs []service.ServiceDetails) (map[string]string, error) {
	svcMap := make(map[string]service.ServiceDetails)
	for _, svc := range svcs {
		svcMap[svc.ID] = svc
	}

	// likely that svcs contains all services since it was likely populated with getServices()
	// however, ensure that parent services are in svcMap
	for _, svc := range svcs {
		parentID := svc.ParentServiceID
		for parentID != "" {
			if _, ok := svcMap[parentID]; ok {
				break // break from inner for loop
			}
			svc, err := c.driver.GetServiceDetails(parentID)
			if err != nil || svc == nil {
				return nil, fmt.Errorf("unable to retrieve service for id:%s %s", parentID, err)
			}
			svcMap[parentID] = *svc

			parentID = svc.ParentServiceID
		}
	}

	// recursively build full path for all services
	pathmap := make(map[string]string)
	for _, svc := range svcs {
		fullpath := svc.Name
		parentServiceID := svc.ParentServiceID

		for parentServiceID != "" {
			fullpath = path.Join(svcMap[parentServiceID].Name, fullpath)
			parentServiceID = svcMap[parentServiceID].ParentServiceID
		}

		pathmap[svc.ID] = strings.ToLower(fullpath)
	}

	return pathmap, nil
}

// cmdSetTreeCharset sets the default behavior for --ASCII, SERVICED_TREE_ASCII, and stdout pipe
func cmdSetTreeCharset(ctx *cli.Context, config utils.ConfigReader) {
	if ctx.Bool("ascii") {
		treeCharset = treeASCII
	} else if !utils.Isatty(os.Stdout) {
		treeCharset = treeSPACE
	} else if config.BoolVal("TREE_ASCII", false) {
		treeCharset = treeASCII
	}
}

// searchForService gets the service and instance id from a provided service
// string, being either a deploymentPath/servicepath/instanceid or
// serviceid/instanceid
func (c *ServicedCli) searchForService(keyword string, prefix bool) (*service.ServiceDetails, int, error) {

	// If the last segment is an integer, it is an instance ID
	servicepath, instanceIDString := path.Split(keyword)
	instanceID := -1
	if num, err := strconv.Atoi(instanceIDString); err != nil {
		// It's not an integer, so just use the original path
		servicepath = keyword
	} else {
		instanceID = num
	}

	matches, err := c.driver.ResolveServicePath(servicepath, prefix)
	if err != nil {
		return nil, 0, err
	}

	// check the number of matches
	if count := len(matches); count == 0 {
		return nil, 0, errors.New("service not found")
	} else if count == 1 {
		return &matches[0], instanceID, nil
	}

	// more than one match, display a dialog
	var svcpath func(*service.ServiceDetails) string
	svcpath = func(svc *service.ServiceDetails) string {
		parent := svc.Parent
		if parent != nil {
			return path.Join(svcpath(parent), svc.Name)
		}
		return path.Join(svc.DeploymentID, svc.Name)
	}

	t := NewTable("Name,ServiceID,DepID/Path")
	t.Padding = 6
	for _, row := range matches {
		t.AddRow(map[string]interface{}{
			"Name":       row.Name,
			"ServiceID":  row.ID,
			"PoolID":     row.PoolID,
			"DepID/Path": svcpath(&row),
		})
	}
	t.Print()
	return nil, 0, fmt.Errorf("multiple results found; select one from list")
}

// serviced service status
func (c *ServicedCli) cmdServiceStatus(ctx *cli.Context) {
	var states map[string]map[string]interface{}
	var err error

	//Determine whether to show healthcheck fields and rows based on user input:
	//   By default, we only show individual healthcheck rows if a specific service is requested
	//   However, we will show them if the user explicitly requests the "Healthcheck" or "Healthcheck Status" fields
	showIndividualHealthChecks := false       //whether or not to add rows to the table for individual health checks.
	fieldsToShow := ctx.String("show-fields") //we will modify this if not user-set

	if !ctx.IsSet("show-fields") {
		//only show the appropriate health fields based on arguments
		if len(ctx.Args()) > 0 { //don't show "HC Fail"
			fieldsToShow = strings.Replace(fieldsToShow, "HC Fail,", "", -1)
			fieldsToShow = strings.Replace(fieldsToShow, ",HC Fail", "", -1) //in case it was last in the list
		} else { //don't show "Healthcheck" or "Healthcheck Status"

			fieldsToShow = strings.Replace(fieldsToShow, "Healthcheck Status,", "", -1)
			fieldsToShow = strings.Replace(fieldsToShow, ",Healthcheck Status", "", -1) //in case it was last in the list

			fieldsToShow = strings.Replace(fieldsToShow, "Healthcheck,", "", -1)
			fieldsToShow = strings.Replace(fieldsToShow, ",Healthcheck", "", -1) //in case it was last in the list
		}
	}

	//set showIndividualHealthChecks based on the fields
	showIndividualHealthChecks = strings.Contains(fieldsToShow, "Healthcheck") || strings.Contains(fieldsToShow, "Healthcheck Status")

	if len(ctx.Args()) > 0 {
		svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}
		if states, err = c.driver.GetServiceStatus(svc.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}
	} else {
		if states, err = c.driver.GetServiceStatus(""); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}
	}

	cmdSetTreeCharset(ctx, c.config)

	t := NewTable(fieldsToShow)
	childmap := make(map[string][]string)
	for id, state := range states {
		parent := fmt.Sprintf("%v", state["ParentID"])
		childmap[parent] = append(childmap[parent], id)
	}

	var addRows func(string)
	addRows = func(root string) {
		rows := childmap[root]
		if len(rows) > 0 {
			sort.Strings(rows)
			t.IndentRow()
			defer t.DedentRow()
			for _, rowid := range childmap[root] {
				row := states[rowid]
				if _, ok := row["Healthcheck"]; !ok || showIndividualHealthChecks { //if this is a healthcheck row, only include it if showIndividualHealthChecks is true
					t.AddRow(row)
				}

				nextRoot := rowid
				addRows(nextRoot)
			}
		}
	}
	addRows("")
	t.Padding = 3
	t.Print()
	return
}

// serviced service list [--verbose, -v] [SERVICEID]
func (c *ServicedCli) cmdServiceList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}

		if service, err := c.driver.GetService(svc.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
		} else if service == nil {
			fmt.Fprintln(os.Stderr, "service not found")
			c.exit(1)
			return
		} else {
			if ctx.String("format") == "" {
				if jsonService, err := json.MarshalIndent(service, " ", "  "); err != nil {
					fmt.Fprintf(os.Stderr, "failed to marshal service definition: %s\n", err)
					c.exit(1)
				} else {
					fmt.Println(string(jsonService))
				}
			} else {
				tpl := ctx.String("format")
				log := log.WithFields(logrus.Fields{
					"format": tpl,
				})
				if tmpl, err := template.New("template").Parse(tpl); err != nil {
					log.WithError(err).Error("Unable to parse format template")
					c.exit(1)
				} else if err := tmpl.Execute(os.Stdout, service); err != nil {
					log.WithError(err).Error("Unable to execute template")
					c.exit(1)
				}
			}
		}
		return
	}

	services, err := c.driver.GetAllServiceDetails()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	} else if services == nil || len(services) == 0 {
		fmt.Fprintln(os.Stderr, "no services found")
		c.exit(1)
		return
	}

	if ctx.Bool("verbose") {
		if jsonService, err := json.MarshalIndent(services, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal service definitions: %s\n", err)
			c.exit(1)
		} else {
			fmt.Println(string(jsonService))
		}
	} else if ctx.String("format") == "" {

		cmdSetTreeCharset(ctx, c.config)

		servicemap := api.NewServiceMap(services)
		t := NewTable(ctx.String("show-fields"))

		var addRows func(string)
		addRows = func(root string) {
			rowids := servicemap.Tree()[root]
			if len(rowids) > 0 {
				sort.Strings(rowids)
				t.IndentRow()
				defer t.DedentRow()
				for _, rowid := range rowids {
					row := servicemap.Get(rowid)
					// truncate the image id
					var imageID string
					if strings.TrimSpace(row.ImageID) != "" {
						id := strings.SplitN(row.ImageID, "/", 3)
						id[0] = "..."
						id[1] = id[1][:7] + "..."
						imageID = strings.Join(id, "/")
					}
					t.AddRow(map[string]interface{}{
						"Name":      row.Name,
						"ServiceID": row.ID,
						"Inst":      row.Instances,
						"ImageID":   imageID,
						"Pool":      row.PoolID,
						"DState":    row.DesiredState,
						"Launch":    row.Launch,
						"DepID":     row.DeploymentID,
					})
					addRows(row.ID)
				}
			}
		}
		addRows("")
		t.Padding = 6
		t.Print()
	} else {
		tpl := ctx.String("format")
		log := log.WithFields(logrus.Fields{
			"format": tpl,
		})
		tmpl, err := template.New("template").Parse(tpl)
		if err != nil {
			log.WithError(err).Error("Unable to parse template")
			c.exit(1)
		}
		for _, service := range services {
			if err := tmpl.Execute(os.Stdout, service); err != nil {
				log.WithError(err).Error("Unable to execute template")
				c.exit(1)
			}
		}
	}
	return
}

// serviced service add [[-p PORT]...] [[-q REMOTE]...] [--parent-id SERVICEID] NAME IMAGEID COMMAND
func (c *ServicedCli) cmdServiceAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		c.exit(1)
		return
	}

	var (
		parentService *service.ServiceDetails
		err           error
	)
	if parentServiceID := ctx.String("parent-id"); parentServiceID == "" {
		fmt.Fprintln(os.Stderr, "Must specify a parent service ID")
		c.exit(1)
		return
	} else if parentService, _, err = c.searchForService(parentServiceID, ctx.Bool("no-prefix-match")); err != nil {
		fmt.Fprintf(os.Stderr, "Error searching for parent service: %s", err)
		c.exit(1)
		return
	}

	cfg := api.ServiceConfig{
		Name:            args[0],
		ImageID:         args[1],
		Command:         args[2],
		ParentServiceID: parentService.ID,
		LocalPorts:      ctx.Generic("p").(*api.PortMap),
		RemotePorts:     ctx.Generic("q").(*api.PortMap),
	}

	if service, err := c.driver.AddService(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
		c.exit(1)
	} else {
		fmt.Println(service.ID)
	}
	return
}

// serviced service clone --config config { SERVICEID | SERVICENAME | [POOL/]...PARENTNAME.../SERVICENAME }
func (c *ServicedCli) cmdServiceClone(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "clone")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching for service: %s", err)
		c.exit(1)
		return
	}
	serviceID := svc.ID

	if copiedSvc, err := c.driver.CloneService(serviceID, ctx.String("suffix")); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", serviceID, err)
		c.exit(1)
	} else if copiedSvc == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
		c.exit(1)
	} else {
		fmt.Println(copiedSvc.ID)
	}
	return
}

// serviced service remove SERVICEID ...
func (c *ServicedCli) cmdServiceRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		// Don't return an error if the service doesn't exist. Philosophically not a error
		c.exit(0)
		return
	}
	serviceID := svc.ID

	if err := c.driver.RemoveService(serviceID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", serviceID, err)
		c.exit(1)
	} else {
		fmt.Println(serviceID)
	}
	return
}

// serviced service edit SERVICEID
func (c *ServicedCli) cmdServiceEdit(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "edit")
		c.exit(1)
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	jsonService, err := json.MarshalIndent(service, " ", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling service: %s\n", err)
		c.exit(1)
		return
	}

	name := fmt.Sprintf("serviced_service_edit_%s", service.ID)
	reader, err := openEditor(jsonService, name, ctx.String("editor"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	if service, err := c.driver.UpdateService(reader); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service")
		c.exit(1)
	} else {
		fmt.Println(service.ID)
	}
	return
}

// serviced service config list SERVICEID [CONFIGFILE]
func (c *ServicedCli) cmdServiceConfigList(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list")
		c.exit(1)
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	configs := service.ConfigFiles

	if len(args) < 2 {
		configList := make([]string, 0)

		for filename := range configs {
			configList = append(configList, filename)
		}
		configJson := map[string]([]string){
			"ConfigFiles": configList,
		}
		if configJsonOut, err := json.MarshalIndent(configJson, " ", "  "); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		} else {
			fmt.Printf("%s\n\n", configJsonOut)
			return
		}
	} else {
		filename := args[1]
		if _, found := configs[filename]; found {
			fmt.Printf("%s", configs[filename].Content)
		} else {
			fmt.Printf("Config file %s not found.\n", filename)
			c.exit(1)
			return
		}
	}
	return
}

// serviced service config edit SERVICEID CONFIGFILE [--editor]
func (c *ServicedCli) cmdServiceConfigEdit(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "edit")
		c.exit(1)
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	configs := service.ConfigFiles
	filename := args[1]
	if _, found := configs[filename]; !found {
		fmt.Printf("Config file %s not found.\n", filename)
		c.exit(1)
		return
	}
	configfile := configs[filename]
	contents := []byte(configfile.Content)
	splitfilename := strings.Split(filename, "/")
	shortname := splitfilename[len(splitfilename)-1]
	name := fmt.Sprintf("serviced_service_edit_%s_%s", service.ID, shortname)
	reader, err := openEditor(contents, name, ctx.String("editor"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	newcontents := new(bytes.Buffer)
	newcontents.ReadFrom(reader)
	newfile := servicedefinition.ConfigFile{
		Filename:    configfile.Filename,
		Owner:       configfile.Owner,
		Permissions: configfile.Permissions,
		Content:     string(newcontents.Bytes()),
	}
	service.ConfigFiles[filename] = newfile
	if service, err := c.driver.UpdateServiceObj(*service); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service")
		c.exit(1)
	} else {
		fmt.Println(service.ID)
	}
	return
}

// serviced service variables list SERVICEID
func (c *ServicedCli) cmdServiceVariableList(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list")
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	variables := service.Context
	keys := make([]string, 0)
	for k, _ := range variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("%s %v\n", key, variables[key])
	}
}

// serviced service variables get SERVICEID VARIABLE
func (c *ServicedCli) cmdServiceVariableGet(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "get")
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	key := args[1]
	if service.Context == nil {
		message := fmt.Sprintf("Variable %v not found.", key)
		fmt.Fprintln(os.Stderr, message)
		return
	}

	if value, found := service.Context[key]; found {
		switch value.(type) {
		case string:
			if vstr, ok := value.(string); ok {
				fmt.Printf("%v\n", vstr)
			}
		case int64:
			if vstr, ok := value.(string); ok {
				fmt.Printf("%v\n", vstr)
			}
		case uint64:
			if vstr, ok := value.(string); ok {
				fmt.Printf("%v\n", vstr)
			}
		}
	} else {
		message := fmt.Sprintf("Variable %v not found.", key)
		fmt.Fprintln(os.Stderr, message)
		return
	}
}

// serviced service variables set SERVICEID VARIABLE VALUE
func (c *ServicedCli) cmdServiceVariableSet(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "set")
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	key := args[1]
	value := args[2]
	if service.Context == nil {
		service.Context = make(map[string]interface{})
	}
	service.Context[key] = value
	if service, err := c.driver.UpdateServiceObj(*service); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service")
	} else {
		fmt.Println(service.ID)
	}
}

// serviced service variables unset SERVICEID VARIABLE
func (c *ServicedCli) cmdServiceVariableUnset(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "unset")
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	key := args[1]
	if service.Context == nil {
		message := fmt.Sprintf("Variable %v not found.", key)
		fmt.Fprintln(os.Stderr, message)
		return
	}

	if _, ok := service.Context[key]; ok {
		delete(service.Context, key)
	} else {
		message := fmt.Sprintf("Variable %s not found.", key)
		fmt.Fprintln(os.Stderr, message)
		return
	}

	if service, err := c.driver.UpdateServiceObj(*service); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service")
	} else {
		fmt.Println(service.ID)
	}
}

// serviced service assign-ip SERVICEID [IPADDRESS]
func (c *ServicedCli) cmdServiceAssignIP(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "assign-ip")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	var ipAddress string
	if len(args) > 1 {
		ipAddress = args[1]
	}

	cfg := api.IPConfig{
		ServiceID: svc.ID,
		IPAddress: ipAddress,
	}

	if err := c.driver.AssignIP(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	}
	return
}

// serviced service remove-ip <SERVICEID> <ENDPOINTNAME>
func (c *ServicedCli) cmdServiceRemoveIP(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove-ip")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}
	serviceID := svc.ID
	endpointName := ctx.Args()[1]

	arguments := []string{serviceID, endpointName}

	if err := c.driver.RemoveIP(arguments); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	}
	return
}

// serviced service set-ip <SERVICEID> <ENDPOINTNAME> [IPADDRESS] [--port=PORT] [--proto=PROTOCOL]
func (c *ServicedCli) cmdServiceSetIP(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "set-ip")
		c.exit(1)
		return
	}

	if args[len(args)-1] == "--generate-bash-completion" {
		// CC-892: Disable bash completion after SERVICE_ID because possible matches
		// are unavailable outside the container.
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	var endpointName string
	if len(args) > 1 {
		endpointName = args[1]
	}

	var ipAddress string
	if len(args) > 2 {
		ipAddress = args[2]
	}

	if ctx.Int("port") < 1 {
		fmt.Printf("Please specify the valid port number.\n\n")
		cli.ShowCommandHelp(ctx, "set-ip")
		c.exit(1)
		return
	}

	if ctx.String("proto") == "" {
		fmt.Printf("Please specify port protocol.\n\n")
		cli.ShowCommandHelp(ctx, "set-ip")
		c.exit(1)
		return
	}

	cfg := api.IPConfig{
		ServiceID:    svc.ID,
		IPAddress:    ipAddress,
		Port:         uint16(ctx.Int("port")),
		Proto:        ctx.String("proto"),
		EndpointName: endpointName,
	}

	if err := c.driver.SetIP(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	}
	return
}

// serviced service start SERVICEID...
func (c *ServicedCli) cmdServiceStart(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "start")
		c.exit(1)
		return
	}

	serviceIDs := make([]string, len(args))
	for i, svcID := range args {
		svc, _, err := c.searchForService(svcID, ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}

		serviceIDs[i] = svc.ID
	}

	if affected, err := c.driver.StartService(api.SchedulerConfig{serviceIDs, ctx.Bool("auto-launch"), ctx.Bool("sync")}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if affected == 0 {
		fmt.Println("Service(s) already started")
	} else {
		fmt.Printf("Scheduled %d service(s) to start\n", affected)
	}
	return
}

// serviced service restart SERVICEID
func (c *ServicedCli) cmdServiceRestart(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "restart")
		c.exit(1)
		return
	}

	var sIds []string
	var instances []struct {
		Service  string
		Instance int
	}

	for _, arg := range args {
		svc, instanceID, err := c.searchForService(arg, ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}

		if instanceID < 0 {
			sIds = append(sIds, svc.ID)
		} else {
			instances = append(instances, struct {
				Service  string
				Instance int
			}{
				svc.ID,
				instanceID,
			})
		}
	}

	// Batch start services
	if len(sIds) > 0 {
		if ctx.Bool("rebalance") {
			if affected, err := c.driver.RebalanceService(api.SchedulerConfig{sIds, ctx.Bool("auto-launch"), ctx.Bool("sync")}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				c.exit(1)
			} else {
				fmt.Printf("Restarting %d service(s)\n", affected)
			}
		} else {
			if affected, err := c.driver.RestartService(api.SchedulerConfig{sIds, ctx.Bool("auto-launch"), ctx.Bool("sync")}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				c.exit(1)
			} else {
				fmt.Printf("Restarting %d service(s)\n", affected)
			}
		}
	}

	// Iterate and reschedule any instances specified
	for _, instance := range instances {
		if err := c.driver.StopServiceInstance(instance.Service, instance.Instance); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
		} else {
			fmt.Printf("Restarting instance %s/%d\n", instance.Service, instance.Instance)
		}
	}
	return
}

// serviced service stop SERVICEID
func (c *ServicedCli) cmdServiceStop(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "stop")
		c.exit(1)
		return
	}

	serviceIDs := make([]string, len(args))
	for i, svcID := range args {
		svc, _, err := c.searchForService(svcID, ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}

		serviceIDs[i] = svc.ID
	}

	if affected, err := c.driver.StopService(api.SchedulerConfig{serviceIDs, ctx.Bool("auto-launch"), ctx.Bool("sync")}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if affected == 0 {
		fmt.Println("Service(s) already stopped")
	} else {
		fmt.Printf("Scheduled %d service(s) to stop\n", affected)
	}
	return
}

// serviced service pause SERVICEID
func (c *ServicedCli) cmdServicePause(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "pause")
		c.exit(1)
		return
	}

	serviceIDs := make([]string, len(args))
	for i, svcID := range args {
		svc, _, err := c.searchForService(svcID, ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}

		serviceIDs[i] = svc.ID
	}

	if affected, err := c.driver.PauseService(api.SchedulerConfig{serviceIDs, ctx.Bool("auto-launch"), ctx.Bool("sync")}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if affected == 0 {
		fmt.Println("Service(s) already paused")
	} else {
		fmt.Printf("Scheduled %d service(s) to pause\n", affected)
	}
	return
}

// serviced service shell [--saveas SAVEAS]  [--interactive, -i] SERVICEID [COMMAND]
func (c *ServicedCli) cmdServiceShell(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		return c.exit(1)
	}

	// Bash completion
	if args[len(args)-1] == "--generate-bash-completion" {
		// CC-892: Disable bash completion after SERVICE_ID because possible matches
		// are unavailable outside the container.
		return c.exit(1)
	}

	var (
		command string
		argv    []string
		isTTY   bool
	)

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return c.exit(1)
	}

	if len(args) < 2 {
		command = "/bin/bash"
		isTTY = true
	} else {
		command = args[1]
		isTTY = ctx.GlobalBool("interactive")
	}

	if len(args) > 2 {
		argv = args[2:]
	}

	config := api.ShellConfig{
		ServiceID:        svc.ID,
		Command:          command,
		Args:             argv,
		SaveAs:           ctx.GlobalString("saveas"),
		IsTTY:            isTTY,
		Mounts:           ctx.GlobalStringSlice("mount"),
		ServicedEndpoint: fmt.Sprintf("localhost:%s", api.GetOptionsRPCPort()),
	}

	if err := c.driver.StartShell(config); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr != nil && exitErr.ProcessState != nil && exitErr.ProcessState.Sys() != nil {
				if status, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok {
					return c.exit(status.ExitStatus())
				}
			}
		}
		return c.exit(1)
	} else {
		return c.exit(0)
	}
}

// serviced service run SERVICEID [COMMAND [ARGS ...]]
func (c *ServicedCli) cmdServiceRun(ctx *cli.Context) error {
	// set up signal handler to stop the run
	stopChan := make(chan struct{})
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Debug("Received stop signal")
		close(stopChan)
		log.Info("Stopped service run")
	}()

	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		return c.exit(1)
	}

	if len(args) < 2 {
		for _, s := range c.serviceCommands(args[0]) {
			fmt.Println(s)
		}
		fmt.Fprintf(os.Stderr, "serviced service run")
		return c.exit(1)
	}

	// Bash completion, CC-892
	if args[len(args)-1] == "--generate-bash-completion" {
		// if "serviced service run SERVICE_ID --generate-bash-completion" is executed
		if len(args) == 2 {
			svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
			if err == nil {
				for _, cmd := range c.serviceCommands(svcDetails.ID) {
					fmt.Println(cmd)
				}
			} else {
				return c.exit(1)
			}
		}
		return c.exit(0)
	}

	var (
		command string
		argv    []string
	)

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return c.exit(1)
	}

	svc, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return c.exit(1)
	}

	if returncode := c.printHelpForRun(svc, args[1]); returncode >= 0 {
		return c.exit(returncode)
	}

	command = args[1]
	if len(args) > 2 {
		argv = args[2:]
	}

	uuid, _ := utils.NewUUID62()

	config := api.ShellConfig{
		ServiceID:        svc.ID,
		Command:          command,
		Username:         ctx.GlobalString("user"),
		Args:             argv,
		SaveAs:           uuid,
		IsTTY:            ctx.GlobalBool("interactive"),
		Mounts:           ctx.GlobalStringSlice("mount"),
		ServicedEndpoint: fmt.Sprintf("localhost:%s", api.GetOptionsRPCPort()),
		LogToStderr:      ctx.GlobalBool("logtostderr"),
	}

	config.LogStash.Enable = ctx.GlobalBool("logstash")
	config.LogStash.SettleTime = ctx.GlobalString("logstash-settle-time")

	exitcode := 1
	if exitcode, err = c.driver.RunShell(config, stopChan); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return c.exit(exitcode)
}

// serviced service attach { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME/INSTANCE } [COMMAND ...]
func (c *ServicedCli) cmdServiceAttach(ctx *cli.Context) error {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		c.exit(1)
		return nil
	}

	// Bash completion
	if args[len(args)-1] == "--generate-bash-completion" {
		// CC-892: The attach command does not require an additional argument after
		// SERVICE_ID. Disable bash completion after SERVICE_ID.
		return c.exit(1)
	}

	svc, instanceID, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return err
	}

	if instanceID < 0 {
		instanceID = 0
	}
	command := ""
	argv := []string{}
	if len(args) > 1 {
		command = args[1]
		argv = args[2:]
	}

	if err := c.driver.AttachServiceInstance(svc.ID, instanceID, command, argv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return err
	}
	return nil
}

// serviced service action { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME/INSTANCE } ACTION
func (c *ServicedCli) cmdServiceAction(ctx *cli.Context) error {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		c.exit(1)
		return nil
	}

	// Bash completion
	if args[len(args)-1] == "--generate-bash-completion" {
		// CC-892
		// if a tab is pressed after serviced service SERVICE_ID and the
		// service is found
		if len(args) == 2 {
			svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
			if err == nil {
				actions := c.serviceActions(svc.ID)
				fmt.Println(strings.Join(actions, "\n"))
			} else {
				c.exit(1)
			}
		}
		return c.exit(0)
	}

	svc, instanceID, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return err
	}

	switch len(args) {
	case 1:
		actions := c.serviceActions(svc.ID)
		if len(actions) > 0 {
			fmt.Println(strings.Join(actions, "\n"))
		} else {
			fmt.Fprintln(os.Stderr, "no actions found")
			c.exit(1)
		}
	default:
		if instanceID < 0 {
			instanceID = 0
		}
		action := ""
		argv := []string{}
		if len(args) > 1 {
			action = args[1]
			argv = args[2:]
		}

		if err := c.driver.SendDockerAction(svc.ID, instanceID, action, argv); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
		}
	}

	return fmt.Errorf("serviced service action")
}

// serviced service logs { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME/INSTANCE }
func (c *ServicedCli) cmdServiceLogs(ctx *cli.Context) error {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		c.exit(1)
		return nil
	}

	// Bash completion
	if args[len(args)-1] == "--generate-bash-completion" {
		// CC-892: The logs command does not require an additional argument after
		// SERVICE_ID. Disable bash completion after SERVICE_ID.
		return c.exit(1)
	}

	svc, instanceID, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return err
	}

	if instanceID < 0 {
		instanceID = 0
	}
	command := ""
	argv := []string{}
	if len(args) > 1 {
		command = args[1]
		argv = args[2:]
	}

	if err := c.driver.LogsForServiceInstance(svc.ID, instanceID, command, argv); err != nil {
		c.exit(1)
		fmt.Fprintln(os.Stderr, err)
	}

	return fmt.Errorf("serviced service logs")
}

// serviced service list-snapshot SERVICEID [--show-tags]
func (c *ServicedCli) cmdServiceListSnapshots(ctx *cli.Context) {
	showTags := ctx.Bool("show-tags")
	if len(ctx.Args()) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list-snapshots")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	if snapshots, err := c.driver.GetSnapshotsByServiceID(svc.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if snapshots == nil || len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "no snapshots found")
	} else {
		if showTags { //print a table of snapshot, description, tag list
			t := NewTable("Snapshot,Description,Tags")
			for _, s := range snapshots {
				//build a comma-delimited list of the tags
				tags := strings.Join(s.Tags, ",")
				snapshotID := s.SnapshotID
				if s.Invalid {
					snapshotID += " [DEPRECATED]"
				}

				//make the row and add it to the table
				row := make(map[string]interface{})
				row["Snapshot"] = snapshotID
				row["Description"] = s.Description
				row["Tags"] = tags
				t.Padding = 6
				t.AddRow(row)
			}
			//print the table
			t.Print()
		} else { //just print a list of snapshots
			for _, s := range snapshots {
				fmt.Println(s)
			}
		}
	}
	return
}

// serviced service snapshot SERVICEID [--tags=<tag1>,<tag2>...]
func (c *ServicedCli) cmdServiceSnapshot(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "snapshot")
		c.exit(1)
		return
	}

	description := ""
	if nArgs <= 3 {
		description = ctx.String("description")
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	//get the tags (if any)
	tag := ctx.String("tag")

	cfg := api.SnapshotConfig{
		ServiceID: svc.ID,
		Message:   description,
		Tag:       tag,
	}
	if snapshot, err := c.driver.AddSnapshot(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if snapshot == "" {
		fmt.Fprintln(os.Stderr, "received nil snapshot")
		c.exit(1)
	} else {
		fmt.Println(snapshot)
	}
	return
}

// serviced service endpoints SERVICEID
func (c *ServicedCli) cmdServiceEndpoints(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "endpoints")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	var reportExports, reportImports bool
	if ctx.Bool("all") {
		reportImports = true
		reportExports = true
	} else if ctx.Bool("imports") {
		reportImports = true
		reportExports = false
	} else {
		reportImports = false
		reportExports = true
	}

	if endpoints, err := c.driver.GetEndpoints(svc.ID, reportImports, reportExports, ctx.Bool("verify")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	} else if len(endpoints) == 0 {
		fmt.Fprintf(os.Stderr, "%s - no endpoints defined\n", svc.Name)
		return
	} else {
		hostmap, err := c.driver.GetHostMap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get host info, printing host IDs instead of names: %s", err)
			c.exit(1)
		}

		t := NewTable("Name,ServiceID,Endpoint,Purpose,Host,HostIP,HostPort,ContainerID,ContainerIP,ContainerPort")
		t.Padding = 4
		for _, endpoint := range endpoints {
			serviceName := svc.Name
			if svc.Instances > 1 && endpoint.Endpoint.ContainerID != "" {
				serviceName = fmt.Sprintf("%s/%d", serviceName, endpoint.Endpoint.InstanceID)
			}

			host := endpoint.Endpoint.HostID
			hostinfo, ok := hostmap[endpoint.Endpoint.HostID]
			if ok {
				host = hostinfo.Name
			}

			var hostPort string
			if endpoint.Endpoint.HostPort != 0 {
				hostPort = strconv.Itoa(int(endpoint.Endpoint.HostPort))
			}

			t.AddRow(map[string]interface{}{
				"Name":          serviceName,
				"ServiceID":     endpoint.Endpoint.ServiceID,
				"Endpoint":      endpoint.Endpoint.Application,
				"Purpose":       endpoint.Endpoint.Purpose,
				"Host":          host,
				"HostIP":        endpoint.Endpoint.HostIP,
				"HostPort":      hostPort,
				"ContainerID":   fmt.Sprintf("%-12.12s", endpoint.Endpoint.ContainerID),
				"ContainerIP":   endpoint.Endpoint.ContainerIP,
				"ContainerPort": endpoint.Endpoint.ContainerPort,
			})
		}
		t.Print()
	}
	return
}

// serviced service clear-emergency { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME }
func (c *ServicedCli) cmdServiceClearEmergency(ctx *cli.Context) {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "clear-emergency")
		c.exit(1)
		return
	}

	svc, _, err := c.searchForService(ctx.Args().First(), ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	count, err := c.driver.ClearEmergency(svc.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	fmt.Printf("Cleared emergency status for %d services\n", count)
	return
}

// serviced service tune SERVICEID
func (c *ServicedCli) cmdServiceTune(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "tune")
		c.exit(1)
		return
	}

	svcDetails, _, err := c.searchForService(args[0], ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	service, err := c.driver.GetService(svcDetails.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}

	// Check the arguments
	if !(ctx.IsSet("instances") || ctx.IsSet("ramCommitment") || ctx.IsSet("ramThreshold") || ctx.IsSet("launchMode")) {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "tune")
		return
	}

	modified := false
	if ctx.IsSet("launchMode") {
		oldLaunchMode := service.Launch
		newLaunchMode := ctx.String("launchMode")
		if newLaunchMode != "auto" && newLaunchMode != "manual" {
			fmt.Printf("Incorrect Usage.\n\n")
			cli.ShowCommandHelp(ctx, "tune")
			return
		}
		if oldLaunchMode != newLaunchMode {
			service.Launch = newLaunchMode
			modified = true
		}
	}

	if ctx.IsSet("instances") {
		oldInstanceCount := service.Instances
		newInstanceCount := ctx.Int("instances")
		if oldInstanceCount != newInstanceCount {
			service.Instances = newInstanceCount
			modified = true
		}
	}

	if ctx.IsSet("ramCommitment") {
		oldCommitment := service.RAMCommitment
		newCommitment, err := utils.NewEngNotationFromString(ctx.String("ramCommitment"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
			return
		}

		if oldCommitment.Value != newCommitment.Value {
			service.RAMCommitment = newCommitment
			modified = true
		}
	}

	if ctx.IsSet("ramThreshold") {
		oldThreshold := uint64(service.RAMThreshold)
		newThreshold := ctx.String("ramThreshold")

		suffix := newThreshold[len(newThreshold)-1:]
		if suffix != "%" {
			fmt.Fprintln(os.Stderr, fmt.Errorf("ramThreshold '%s' does not end with %%", newThreshold))
			c.exit(1)
			return
		}

		percent := newThreshold[:len(newThreshold)-1]
		val, err := strconv.ParseUint(percent, 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("ramThreshold '%s' must be an integer", newThreshold))
			c.exit(1)
			return
		}

		if oldThreshold != val {
			service.RAMThreshold = uint(val)
			modified = true
		}
	}

	if modified {
		if service, err := c.driver.UpdateServiceObj(*service); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if service == nil {
			fmt.Fprintln(os.Stderr, "received nil service")
			c.exit(1)
			return
		} else {
			fmt.Println(service.ID)
		}
	} else {
		fmt.Printf("Service already reflects desired configured - no changes made\n\n")
		return
	}

}
