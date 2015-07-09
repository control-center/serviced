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
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sort"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	dockerclient "github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

var unstartedTime = time.Date(1999, 12, 31, 23, 59, 0, 0, time.UTC)

// Initializer for serviced service subcommands
func (c *ServicedCli) initService() {

	defaultMetricsForwarderPort := ":22350"
	if cpConsumerUrl, err := url.Parse(os.Getenv("CONTROLPLANE_CONSUMER_URL")); err == nil {
		hostParts := strings.Split(cpConsumerUrl.Host, ":")
		if len(hostParts) == 2 {
			defaultMetricsForwarderPort = ":" + hostParts[1]
		}
	}

	rpcPort := c.config.IntVal("RPC_PORT", defaultRPCPort)

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
					cli.BoolFlag{"verbose, v", "Show JSON format"},
					cli.BoolFlag{"ascii, a", "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)"},
					cli.StringFlag{"format", "", "format the output using the given go template"},
					cli.StringFlag{"show-fields", "Name,ServiceID,Inst,ImageID,Pool,DState,Launch,DepID", "Comma-delimited list describing which fields to display"},
				},
			}, {
				Name:        "status",
				Usage:       "Displays the status of deployed services",
				Description: "serviced service status { SERVICEID | SERVICENAME | [POOL/]...PARENTNAME.../SERVICENAME }",
				Action:      c.cmdServiceStatus,
				Flags: []cli.Flag{
					cli.BoolFlag{"ascii, a", "use ascii characters for service tree (env SERVICED_TREE_ASCII=1 will default to ascii)"},
					cli.StringFlag{"show-fields", "Name,ServiceID,Status,Uptime,RAM,Cur/Max/Avg,Hostname,InSync,DockerID", "Comma-delimited list describing which fields to display"},
				},
			}, {
				Name:        "add",
				Usage:       "Adds a new service",
				Description: "serviced service add NAME IMAGEID COMMAND",
				Action:      c.cmdServiceAdd,
				Flags: []cli.Flag{
					cli.GenericFlag{"p", &api.PortMap{}, "Expose a port for this service (e.g. -p tcp:3306:mysql)"},
					cli.GenericFlag{"q", &api.PortMap{}, "Map a remote service port (e.g. -q tcp:3306:mysql)"},
					cli.StringFlag{"parent-id", "", "Parent service ID for which this service relates"},
				},
			}, {
				Name:        "clone",
				Usage:       "Clones a new service",
				Description: "serviced service clone { SERVICEID | SERVICENAME | [POOL/]...PARENTNAME.../SERVICENAME }",
				Action:      c.cmdServiceClone,
				Flags: []cli.Flag{
					cli.StringFlag{"suffix", "", "name to append to service name, volumes, endpoints"},
				},
			}, {
				Name:         "migrate",
				ShortName:    "mig",
				Usage:        "Migrate an existing service",
				Description:  "serviced service migrate SERVICEID PATH_TO_SCRIPT",
				BashComplete: c.printServicesAll,
				Action:       c.cmdServiceMigrate,
				Flags: []cli.Flag{
					cli.BoolFlag{"dry-run", "Executes the migration and validation without updating anything"},
					cli.StringFlag{"sdk-version", "", "Overrides the default service-migration SDK version"},
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
					cli.StringFlag{"editor, e", os.Getenv("EDITOR"), "Editor used to update the service definition"},
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
					cli.BoolTFlag{"auto-launch", "Recursively schedules child services"},
				},
			}, {
				Name:         "restart",
				Usage:        "Restarts a service",
				Description:  "serviced service restart SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceRestart,
				Flags: []cli.Flag{
					cli.BoolTFlag{"auto-launch", "Recursively schedules child services"},
				},
			}, {
				Name:         "stop",
				Usage:        "Stops a service",
				Description:  "serviced service stop SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStop,
				Flags: []cli.Flag{
					cli.BoolTFlag{"auto-launch", "Recursively schedules child services"},
				},
			}, {
				Name:         "proxy",
				Usage:        "Starts a server proxy for a container",
				Description:  "serviced service proxy SERVICEID INSTANCEID COMMAND",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceProxy,
				Flags: []cli.Flag{
					cli.StringFlag{"forwarder-binary", "/usr/local/serviced/resources/logstash/logstash-forwarder", "path to the logstash-forwarder binary"},
					cli.StringFlag{"forwarder-config", "/etc/logstash-forwarder.conf", "path to the logstash-forwarder config file"},
					cli.IntFlag{"muxport", 22250, "multiplexing port to use"},
					cli.BoolTFlag{"mux", "enable port multiplexing"},
					cli.StringFlag{"keyfile", "", "path to private key file (defaults to compiled in private keys"},
					cli.StringFlag{"certfile", "", "path to public certificate file (defaults to compiled in public cert)"},
					cli.StringFlag{"endpoint", api.GetGateway(rpcPort), "serviced endpoint address"},
					cli.BoolTFlag{"autorestart", "restart process automatically when it finishes"},
					cli.BoolFlag{"disable-metric-forwarding", "disable forwarding of metrics for this container"},
					cli.StringFlag{"metric-forwarder-port", defaultMetricsForwarderPort, "the port the container processes send performance data to"},
					cli.BoolTFlag{"logstash", "forward service logs via logstash-forwarder"},
					cli.StringFlag{"logstash-idle-flush-time", "5s", "time duration for logstash to flush log messages"},
					cli.StringFlag{"logstash-settle-time", "0s", "time duration to wait for logstash to flush log messages before closing"},
					cli.StringFlag{"virtual-address-subnet", c.config.StringVal("VIRTUAL_ADDRESS_SUBNET", "10.3"), "/16 subnet for virtual addresses"},
				},
			}, {
				Name:         "shell",
				Usage:        "Starts a service instance",
				Description:  "serviced service shell SERVICEID [COMMAND]",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceShell,
				Flags: []cli.Flag{
					cli.StringFlag{"saveas, s", "", "saves the service instance with the given name"},
					cli.BoolFlag{"interactive, i", "runs the service instance as a tty"},
					cli.StringSliceFlag{"mount", &cli.StringSlice{}, "bind mount: HOST_PATH[,CONTAINER_PATH]"},
				},
			}, {
				Name:         "run",
				Usage:        "Runs a service command in a service instance",
				Description:  "serviced service run SERVICEID COMMAND [ARGS]",
				BashComplete: c.printServiceRun,
				Before:       c.cmdServiceRun,
				Flags: []cli.Flag{
					cli.BoolFlag{"interactive, i", "runs the service instance as a tty"},
					cli.BoolFlag{"logtostderr", "enable/disable detailed serviced run logging (false by default)"},
					cli.BoolTFlag{"logstash", "enable/disable log stash (true by default)"},
					cli.StringFlag{"logstash-idle-flush-time", "100ms", "time duration for logstash to flush log messages"},
					cli.StringFlag{"logstash-settle-time", "5s", "time duration to wait for logstash to flush log messages before closing"},
					cli.StringSliceFlag{"mount", &cli.StringSlice{}, "bind mount: HOST_PATH[,CONTAINER_PATH]"},
					cli.StringFlag{"user", "", "container username used to run command"},
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
			}, {
				Name:         "snapshot",
				Usage:        "Takes a snapshot of the service",
				Description:  "serviced service snapshot SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceSnapshot,
				Flags: []cli.Flag{
					cli.StringFlag{"description, d", "", "a description of the snapshot"},
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
func cmdSetTreeCharset(ctx *cli.Context, config ConfigReader) {
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

	t := NewTable(ctx.String("show-fields"))
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
				t.AddRow(row)
				nextRoot := fmt.Sprintf("%v", row["ServiceID"])
				addRows(nextRoot)
			}
		}
	}
	addRows("")
	t.Padding = 6
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

// serviced service migrate SERVICEID ...
func (c *ServicedCli) cmdServiceMigrate(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 || len(args) > 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "migrate")
		return
	}

	svc, err := c.searchForService(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var input *os.File
	if len(args) == 2 {
		filepath := args[1]
		var err error
		if input, err = os.Open(filepath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer input.Close()
	} else {
		input = os.Stdin
	}

	if migratedSvc, err := c.driver.RunMigrationScript(svc.ID, input, ctx.Bool("dry-run"), ctx.String("sdk-version")); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", svc.ID, err)
	} else {
		fmt.Println(migratedSvc.ID)
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

		glog.Infof("KWW: name=%s id=%s hostId=%s serviceId=%s instanceId=%d\n", runningSvc.Name, runningSvc.ID, // DEBUG: KWW:
			runningSvc.HostID, runningSvc.ServiceID, runningSvc.InstanceID) // DEBUG: KWW:

		if err := c.driver.StopRunningService(runningSvc.HostID, runningSvc.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Printf("Restarting service\n")
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

// serviced service proxy SERVICE_ID INSTANCEID COMMAND
func (c *ServicedCli) cmdServiceProxy(ctx *cli.Context) error {
	if len(ctx.Args()) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		return nil
	}

	args := ctx.Args()
	options := api.ControllerOptions{
		MuxPort:                 ctx.GlobalInt("muxport"),
		Mux:                     ctx.GlobalBool("mux"),
		TLS:                     true,
		KeyPEMFile:              ctx.GlobalString("keyfile"),
		CertPEMFile:             ctx.GlobalString("certfile"),
		ServicedEndpoint:        ctx.GlobalString("endpoint"),
		Autorestart:             ctx.GlobalBool("autorestart"),
		MetricForwarderPort:     ctx.GlobalString("metric-forwarder-port"),
		Logstash:                ctx.GlobalBool("logstash"),
		LogstashIdleFlushTime:   ctx.GlobalString("logstash-idle-flush-time"),
		LogstashSettleTime:      ctx.GlobalString("logstash-settle-time"),
		LogstashBinary:          ctx.GlobalString("forwarder-binary"),
		LogstashConfig:          ctx.GlobalString("forwarder-config"),
		VirtualAddressSubnet:    ctx.GlobalString("virtual-address-subnet"),
		ServiceID:               args[0],
		InstanceID:              args[1],
		Command:                 args[2:],
		MetricForwardingEnabled: !ctx.GlobalBool("disable-metric-forwarding"),
	}

	if err := c.driver.StartProxy(options); err != nil {
		sendLogMessage(options.ServicedEndpoint,
			node.ServiceLogInfo{
				ServiceID: options.ServiceID,
				Message:   "container controller terminated with: " + err.Error(),
			})
		fmt.Fprintln(os.Stderr, err)
	}

	return fmt.Errorf("serviced service proxy")
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

	config := api.ShellConfig{
		ServiceID:        svc.ID,
		Command:          command,
		Username:         ctx.GlobalString("user"),
		Args:             argv,
		SaveAs:           dfs.NewLabel(svc.ID),
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

	hosts, err := c.driver.GetHosts()
	if err != nil {
		return nil, err
	}
	hostmap := make(map[string]host.Host)
	for _, host := range hosts {
		hostmap[host.ID] = host
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
		hosts, err := c.driver.GetHosts()
		if err != nil {
			return err
		}
		hostmap := make(map[string]host.Host)
		for _, host := range hosts {
			hostmap[host.ID] = host
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
		hosts, err := c.driver.GetHosts()
		if err != nil {
			return err
		}
		hostmap := make(map[string]host.Host)
		for _, host := range hosts {
			hostmap[host.ID] = host
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

// serviced service list-snapshot SERVICEID
func (c *ServicedCli) cmdServiceListSnapshots(ctx *cli.Context) {
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
		for _, s := range snapshots {
			fmt.Println(s)
		}
	}
}

// serviced service snapshot SERVICEID
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

	if snapshot, err := c.driver.AddSnapshot(svc.ID, description); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	} else if snapshot == "" {
		fmt.Fprintln(os.Stderr, "received nil snapshot")
		c.exit(1)
	} else {
		fmt.Println(snapshot)
	}
}
