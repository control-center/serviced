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
	"encoding/json"
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

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	dockerclient "github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
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
				Description: "serviced service clone { SERVICEID | SERVICENAME | [POOL/]...PARENTNAME.../SERVICENAME }",
				Action:      c.cmdServiceClone,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "suffix",
						Value: "",
						Usage: "name to append to service name, volumes, endpoints",
					},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing service",
				Description:  "serviced service remove SERVICEID",
				BashComplete: c.printServicesAll,
				Action:       c.cmdServiceRemove,
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
				},
			}, {
				Name:         "assign-ip",
				Usage:        "Assigns an IP address to a service's endpoints requiring an explicit IP address",
				Description:  "serviced service assign-ip SERVICEID [IPADDRESS]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceAssignIP,
			}, {
				Name:         "start",
				Usage:        "Starts a service",
				Description:  "serviced service start SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStart,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
					},
				},
			}, {
				Name:         "restart",
				Usage:        "Restarts a service",
				Description:  "serviced service restart { SERVICEID | INSTANCEID }",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceRestart,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
					},
				},
			}, {
				Name:         "stop",
				Usage:        "Stops a service",
				Description:  "serviced service stop SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStop,
				Flags: []cli.Flag{
					cli.BoolTFlag{
						Name:  "auto-launch",
						Usage: "Recursively schedules child services",
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
						Value:  &cli.StringSlice{},
						Usage: "bind mount: HOST_PATH[,CONTAINER_PATH]",
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
						Name:  "logstash-idle-flush-time",
						Value: "100ms",
						Usage: "time duration for logstash to flush log messages",
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
				},
			}, {
				Name:         "attach",
				Usage:        "Run an arbitrary command in a running service container",
				Description:  "serviced service attach { SERVICEID | SERVICENAME | DOCKERID | POOL/...PARENTNAME.../SERVICENAME/INSTANCE } [COMMAND]",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceAttach,
			}, {
				Name:         "action",
				Usage:        "Run a predefined action in a running service container",
				Description:  "serviced service action { SERVICEID | SERVICENAME | DOCKERID | POOL/...PARENTNAME.../SERVICENAME/INSTANCE } ACTION",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceAction,
			}, {
				Name:         "logs",
				Usage:        "Output the logs of a running service container - calls docker logs",
				Description:  "serviced service logs { SERVICEID | SERVICENAME | DOCKERID | POOL/...PARENTNAME.../SERVICENAME/INSTANCE }",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceLogs,
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
				},
			},
		},
	})
}

// Returns a list of all the available service IDs
func (c *ServicedCli) services() (data []string) {
	svcs, err := c.driver.GetServices()
	if err != nil || svcs == nil || len(svcs) == 0 {
		return
	}

	data = make([]string, len(svcs))
	for i, s := range svcs {
		data[i] = s.ID
	}

	return
}

// Returns a list of runnable commands for a particular service
func (c *ServicedCli) serviceRuns(id string) (data []string) {
	svc, err := c.driver.GetService(id)
	if err != nil || svc == nil {
		return
	}

	data = make([]string, len(svc.Runs))
	i := 0
	for r := range svc.Runs {
		data[i] = r
		i++
	}

	return
}

// Returns a list of actionable commands for a particular service
func (c *ServicedCli) serviceActions(id string) (data []string) {
	svc, err := c.driver.GetService(id)
	if err != nil || svc == nil {
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
		output = c.serviceRuns(args[0])
	}
	fmt.Println(strings.Join(output, "\n"))
}

// buildServicePaths returns a map where map[service.ID] = fullpath
func (c *ServicedCli) buildServicePaths(svcs []service.Service) (map[string]string, error) {
	svcMap := make(map[string]service.Service)
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
			glog.Warningf("should not have to retrieve parent service %s", parentID)

			svc, err := c.driver.GetService(parentID)
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
		glog.V(2).Infof("service: %-16s  %s  path: %s", svc.Name, svc.ID, pathmap[svc.ID])
	}

	return pathmap, nil
}

// searches for service from definitions given keyword
func (c *ServicedCli) searchForService(keyword string) (*service.Service, error) {
	svcs, err := c.driver.GetServices()
	if err != nil {
		return nil, err
	}

	pathmap, err := c.buildServicePaths(svcs)
	if err != nil {
		return nil, err
	}

	var services []service.Service
	for _, svc := range svcs {
		poolPath := path.Join(strings.ToLower(svc.PoolID), pathmap[svc.ID])
		switch strings.ToLower(keyword) {
		case svc.ID, strings.ToLower(svc.Name), pathmap[svc.ID], poolPath:
			services = append(services, svc)
		default:
			if keyword == "" {
				services = append(services, svc)
			} else if strings.HasSuffix(pathmap[svc.ID], strings.ToLower(keyword)) {
				services = append(services, svc)
			}
		}
	}

	switch len(services) {
	case 0:
		return nil, fmt.Errorf("service not found")
	case 1:
		return &services[0], nil
	}

	t := NewTable("Name,ServiceID,DepID,Pool/Path")
	t.Padding = 6
	for _, row := range services {
		t.AddRow(map[string]interface{}{
			"Name":      row.Name,
			"ServiceID": row.ID,
			"DepID":     row.DeploymentID,
			"Pool/Path": path.Join(row.PoolID, pathmap[row.ID]),
		})
	}
	t.Print()
	return nil, fmt.Errorf("multiple results found; select one from list")
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
		svc, err := c.searchForService(ctx.Args()[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		} else if svc == nil {
			fmt.Fprintln(os.Stderr, "service not found")
			return
		}

		if states, err = c.driver.GetServiceStatus(svc.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	} else {
		if states, err = c.driver.GetServiceStatus(""); err != nil {
			fmt.Fprintln(os.Stderr, err)
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
		svc, err := c.searchForService(ctx.Args()[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		serviceID := svc.ID
		if service, err := c.driver.GetService(serviceID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if service == nil {
			fmt.Fprintln(os.Stderr, "service not found")
			return
		} else {
			if ctx.String("format") == "" {
				if jsonService, err := json.MarshalIndent(service, " ", "  "); err != nil {
					fmt.Fprintf(os.Stderr, "failed to marshal service definition: %s\n", err)
				} else {
					fmt.Println(string(jsonService))
				}
			} else {
				if tmpl, err := template.New("template").Parse(ctx.String("format")); err != nil {
					glog.Errorf("Template parsing error: %s", err)
				} else if err := tmpl.Execute(os.Stdout, service); err != nil {
					glog.Errorf("Template execution error: %s", err)
				}
			}
		}
		return
	}

	services, err := c.driver.GetServices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	} else if services == nil || len(services) == 0 {
		fmt.Fprintln(os.Stderr, "no services found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonService, err := json.MarshalIndent(services, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal service definitions: %s\n", err)
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
		tmpl, err := template.New("template").Parse(ctx.String("format"))
		if err != nil {
			glog.Errorf("Template parsing error: %s", err)
		}
		for _, service := range services {
			if err := tmpl.Execute(os.Stdout, service); err != nil {
				glog.Errorf("Template execution error: %s", err)
			}
		}
	}
}

// serviced service add [[-p PORT]...] [[-q REMOTE]...] [--parent-id SERVICEID] NAME IMAGEID COMMAND
func (c *ServicedCli) cmdServiceAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	var (
		parentService *service.Service
		err           error
	)
	if parentServiceID := ctx.String("parent-id"); parentServiceID == "" {
		fmt.Fprintln(os.Stderr, "Must specify a parent service ID")
		return
	} else if parentService, err = c.searchForService(parentServiceID); err != nil {
		fmt.Fprintf(os.Stderr, "Error searching for parent service: %s", err)
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
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
	} else {
		fmt.Println(service.ID)
	}
}

// serviced service clone --config config { SERVICEID | SERVICENAME | [POOL/]...PARENTNAME.../SERVICENAME }
func (c *ServicedCli) cmdServiceClone(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "clone")
		return
	}

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching for service: %s", err)
		return
	}

	if copiedSvc, err := c.driver.CloneService(svc.ID, ctx.String("suffix")); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", svc.ID, err)
	} else if copiedSvc == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
	} else {
		fmt.Println(copiedSvc.ID)
	}
}

// serviced service remove SERVICEID ...
func (c *ServicedCli) cmdServiceRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if err := c.driver.RemoveService(svc.ID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", svc.ID, err)
	} else {
		fmt.Println(svc.ID)
	}
}

// serviced service edit SERVICEID
func (c *ServicedCli) cmdServiceEdit(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "edit")
		return
	}

	service, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	jsonService, err := json.MarshalIndent(service, " ", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling service: %s\n", err)
		return
	}

	name := fmt.Sprintf("serviced_service_edit_%s", service.ID)
	reader, err := openEditor(jsonService, name, ctx.String("editor"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if service, err := c.driver.UpdateService(reader); err != nil {
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
		return
	}

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
	}
}

// serviced service start SERVICEID
func (c *ServicedCli) cmdServiceStart(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "start")
		return
	}

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if affected, err := c.driver.StartService(api.SchedulerConfig{svc.ID, ctx.Bool("auto-launch")}); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if affected == 0 {
		fmt.Println("Service already started")
	} else {
		fmt.Printf("Scheduled %d service(s) to start\n", affected)
	}
}

// serviced service restart SERVICEID
func (c *ServicedCli) cmdServiceRestart(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "restart")
		return
	}

	if !isInstanceID(args[0]) {
		svc, err := c.searchForService(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		if affected, err := c.driver.RestartService(api.SchedulerConfig{svc.ID, ctx.Bool("auto-launch")}); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Printf("Restarting %d service(s)\n", affected)
		}
	} else {
		runningSvc, err := c.searchForRunningService(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		if err := c.driver.StopRunningService(runningSvc.HostID, runningSvc.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Printf("Restarting 1 service(s)\n")
		}
	}
}

// serviced service stop SERVICEID
func (c *ServicedCli) cmdServiceStop(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "stop")
		return
	}

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if affected, err := c.driver.StopService(api.SchedulerConfig{svc.ID, ctx.Bool("auto-launch")}); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if affected == 0 {
		fmt.Println("Service already stopped")
	} else {
		fmt.Printf("Scheduled %d service(s) to stop\n", affected)
	}
}

// sendLogMessage sends a log message to the host agent
func sendLogMessage(lbClientPort string, serviceLogInfo node.ServiceLogInfo) error {
	client, err := node.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return err
	}
	defer client.Close()
	return client.SendLogMessage(serviceLogInfo, nil)
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

	var (
		command string
		argv    []string
		isTTY   bool
	)

	svc, err := c.searchForService(args[0])
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
		glog.Infof("Received stop signal, stopping")
		close(stopChan)
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
		for _, s := range c.serviceRuns(args[0]) {
			fmt.Println(s)
		}
		fmt.Fprintf(os.Stderr, "serviced service run")
		return c.exit(1)
	}

	var (
		command string
		argv    []string
	)

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return c.exit(1)
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
	config.LogStash.IdleFlushTime = ctx.GlobalString("logstash-idle-flush-time")

	exitcode := 1
	if exitcode, err = c.driver.RunShell(config, stopChan); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return c.exit(exitcode)
}

// buildRunningServicePaths returns a map where map[rs.ID] = fullpath
func (c *ServicedCli) buildRunningServicePaths(rss []dao.RunningService) (map[string]string, error) {
	pathmap := make(map[string]string)

	// generate parentmap and namemap for all running services with key of serviceid/inst
	parentmap := make(map[string]string)
	namemap := make(map[string]string)
	for _, rs := range rss {
		rskey := path.Join(rs.ServiceID, fmt.Sprintf("%d", rs.InstanceID))
		namemap[rskey] = rs.Name
		parentmap[rskey] = rs.ParentServiceID

		// populate namemap and parentmap for parent services
		parentID := rs.ParentServiceID
		for parentID != "" {
			pakey := path.Join(parentID, fmt.Sprintf("%d", 0))
			_, ok := namemap[parentID]
			if ok {
				break // break from inner for loop
			}

			svc, err := c.driver.GetService(parentID)
			if err != nil || svc == nil {
				return pathmap, fmt.Errorf("unable to retrieve service for id:%s %s", parentID, err)
			}
			namemap[pakey] = svc.Name
			parentmap[pakey] = svc.ParentServiceID

			parentID = svc.ParentServiceID
		}
	}

	// recursively build full path for all running services
	for _, rs := range rss {
		fullpath := path.Join(rs.Name, fmt.Sprintf("%d", rs.InstanceID))
		parentServiceID := rs.ParentServiceID

		for parentServiceID != "" {
			pakey := path.Join(parentServiceID, fmt.Sprintf("%d", 0))
			fullpath = path.Join(namemap[pakey], fullpath)
			parentServiceID = parentmap[pakey]
		}

		pathmap[rs.ID] = strings.ToLower(fullpath)
		glog.V(2).Infof("========= rs:%s %s  path[%s]:%s", rs.ServiceID, rs.Name, rs.ID, pathmap[rs.ID])
	}

	return pathmap, nil
}

// searches for a running service from running services given keyword
func (c *ServicedCli) searchForRunningService(keyword string) (*dao.RunningService, error) {
	rss, err := c.driver.GetRunningServices()
	if err != nil {
		return nil, err
	}

	hostmap, err := c.driver.GetHostMap()
	if err != nil {
		return nil, err
	}

	pathmap, err := c.buildRunningServicePaths(rss)
	if err != nil {
		return nil, err
	}

	var states []dao.RunningService
	for _, rs := range rss {
		if rs.DockerID == "" {
			continue
		}

		poolPathInstance := path.Join(strings.ToLower(rs.PoolID), pathmap[rs.ID])
		serviceIDInstance := path.Join(rs.ServiceID, fmt.Sprintf("%d", rs.InstanceID))
		switch strings.ToLower(keyword) {
		case rs.ServiceID, serviceIDInstance, strings.ToLower(rs.Name), rs.ID, rs.DockerID, rs.DockerID[0:12], pathmap[rs.ID], poolPathInstance:
			states = append(states, rs)
		default:
			if keyword == "" {
				states = append(states, rs)
			} else if strings.HasSuffix(pathmap[rs.ID], strings.ToLower(keyword)) {
				states = append(states, rs)
			}
		}
	}

	switch len(states) {
	case 0:
		return nil, fmt.Errorf("no matches found")
	case 1:
		return &states[0], nil
	}

	t := NewTable("Name,ID,Host,HostIP,DockerID,Pool/Path")
	t.Padding = 6
	for _, row := range states {
		svcid := row.ServiceID
		name := row.Name
		if row.Instances > 1 {
			svcid = fmt.Sprintf("%s/%d", row.ServiceID, row.InstanceID)
			name = fmt.Sprintf("%s/%d", row.Name, row.InstanceID)
		}

		t.AddRow(map[string]interface{}{
			"Name":      name,
			"ID":        svcid,
			"Host":      hostmap[row.HostID].Name,
			"HostIP":    hostmap[row.HostID].IPAddr,
			"DockerID":  row.DockerID[0:12],
			"Pool/Path": path.Join(row.PoolID, pathmap[row.ID]),
		})
	}
	t.Print()
	return nil, fmt.Errorf("multiple results found; specify unique item from list")
}

// serviced service attach { SERVICEID | SERVICENAME | DOCKERID | POOL/...PARENTNAME.../SERVICENAME/INSTANCE } [COMMAND ...]
func (c *ServicedCli) cmdServiceAttach(ctx *cli.Context) error {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	rs, err := c.searchForRunningService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	// attach on remote host if service is running on remote
	myHostID, err := utils.HostID()
	if err != nil {
		return err
	}

	if rs.HostID != myHostID {
		hostmap, err := c.driver.GetHostMap()
		if err != nil {
			return err
		}

		cmd := []string{"/usr/bin/ssh", "-t", hostmap[rs.HostID].IPAddr, "--", "serviced", "--endpoint", api.GetOptionsRPCEndpoint(), "service", "attach", args[0]}
		if len(args) > 1 {
			cmd = append(cmd, args[1:]...)
		}

		glog.V(1).Infof("remote attaching with: %s\n", cmd)
		return syscall.Exec(cmd[0], cmd[0:], os.Environ())
	}

	// attach on local host if service is running locally
	var command string
	if len(args) > 1 {
		command = args[1]
	}

	var argv []string
	if len(args) > 2 {
		argv = args[2:]
	}

	cfg := api.AttachConfig{
		Running: rs,
		Command: command,
		Args:    argv,
	}

	if err := c.driver.Attach(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return fmt.Errorf("serviced service attach")
}

// serviced service action { SERVICEID | SERVICENAME | DOCKERID | POOL/...PARENTNAME.../SERVICENAME/INSTANCE } ACTION
func (c *ServicedCli) cmdServiceAction(ctx *cli.Context) error {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	rs, err := c.searchForRunningService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	switch len(args) {
	case 1:
		actions := c.serviceActions(rs.ServiceID)
		if len(actions) > 0 {
			fmt.Println(strings.Join(actions, "\n"))
		} else {
			fmt.Fprintln(os.Stderr, "no actions found")
		}
	default:
		cfg := api.AttachConfig{
			Running: rs,
			Command: args[1],
		}
		if len(args) > 2 {
			cfg.Args = args[2:]
		}

		if err := c.driver.Action(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return fmt.Errorf("serviced service action")
}

// serviced service logs { SERVICEID | SERVICENAME | DOCKERID | POOL/...PARENTNAME.../SERVICENAME/INSTANCE }
func (c *ServicedCli) cmdServiceLogs(ctx *cli.Context) error {
	// verify args
	args := ctx.Args()
	if len(args) < 1 {
		if !ctx.Bool("help") {
			fmt.Fprintf(os.Stderr, "Incorrect Usage.\n\n")
		}
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	rs, err := c.searchForRunningService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	// docker logs on remote host if service is running on remote
	myHostID, err := utils.HostID()
	if err != nil {
		return err
	}

	if rs.HostID != myHostID {
		hostmap, err := c.driver.GetHostMap()
		if err != nil {
			return err
		}

		cmd := []string{"/usr/bin/ssh", "-t", hostmap[rs.HostID].IPAddr, "--", "serviced", "--endpoint", api.GetOptionsRPCEndpoint(), "service", "logs", args[0]}
		if len(args) > 1 {
			cmd = append(cmd, args[1:]...)
		}

		glog.V(1).Infof("outputting remote logs with: %s\n", cmd)
		return syscall.Exec(cmd[0], cmd[0:], os.Environ())
	}

	// docker logs on local host if service is running locally
	var argv []string
	if len(args) > 2 {
		argv = args[2:]
	}

	if err := dockerclient.Logs(rs.DockerID, argv); err != nil {
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
		return
	}

	svc, err := c.searchForService(ctx.Args().First())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if snapshots, err := c.driver.GetSnapshotsByServiceID(svc.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
}

// serviced service snapshot SERVICEID [--tags=<tag1>,<tag2>...]
func (c *ServicedCli) cmdServiceSnapshot(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "snapshot")
		return
	}

	description := ""
	if nArgs <= 3 {
		description = ctx.String("description")
	}

	svc, err := c.searchForService(ctx.Args().First())
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
}

// serviced service endpoints SERVICEID
func (c *ServicedCli) cmdServiceEndpoints(ctx *cli.Context) {
	nArgs := len(ctx.Args())
	if nArgs < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "endpoints")
		return
	}

	svc, err := c.searchForService(ctx.Args().First())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
		return
	} else if len(endpoints) == 0 {
		fmt.Fprintf(os.Stderr, "%s - no endpoints defined\n", svc.Name)
		return
	} else {
		hostmap, err := c.driver.GetHostMap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get host info, printing host IDs instead of names: %s", err)
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
}
