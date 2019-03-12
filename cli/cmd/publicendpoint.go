// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/domain/service"
)

// The vhost and port public endpoint structures are different, so we'll
// make a unified structure for output (both text and json) that matches
// the UI table.  This is only needed for output, not for api commands.
type PublicEndpoint struct {
	Service     string
	ServiceID   string
	Application string
	EpType      string
	Protocol    string
	Name        string
	Enabled     bool
}

func NewPublicEndpoint(service string, serviceID string, endpoint string, epType string,
	protocol string, name string, enabled bool) PublicEndpoint {
	return PublicEndpoint{
		Service:     service,
		ServiceID:   serviceID,
		Application: endpoint,
		EpType:      epType,
		Protocol:    protocol,
		Name:        name,
		Enabled:     enabled,
	}
}

// serviced service public-endpoints
func (c *ServicedCli) cmdPublicEndpointsListAll(ctx *cli.Context) {
	// If they specify only vhosts/ports, return those.  If they didn't specify
	// either then both are returned.
	cmdPublicEndpointsList(
		c,
		ctx,
		ctx.Bool("vhosts") || (!ctx.Bool("vhosts") && !ctx.Bool("ports")),
		ctx.Bool("ports") || (!ctx.Bool("vhosts") && !ctx.Bool("ports")),
	)
}

// Method that executes the serviced service public-endpoints list.  Also called from the
// port list *, and vhost list * subcommands.
func cmdPublicEndpointsList(c *ServicedCli, ctx *cli.Context, showVHosts bool, showPorts bool) {

	var publicEndpoints []PublicEndpoint

	if len(ctx.Args()) > 0 {
		// Provided the service id/name.
		svcDetails, _, err := c.searchForService(ctx.Args()[0],ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		svc, err := c.driver.GetService(svcDetails.ID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		services := []service.Service{*svc}

		publicEndpoints, err = c.createPublicEndpoints(ctx, services, showVHosts, showPorts)
		// If there was an error getting the endpoints, show the error now.
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return
		}
	} else {
		// Showing all service ports/vhosts.
		peps, err := c.driver.GetAllPublicEndpoints()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get public endpoints: %s\n", err)
			return
		} else if peps == nil || len(peps) == 0 {
			fmt.Fprintln(os.Stderr, "no services found")
			return
		}
		publicEndpoints, err = c.convertPublicEndpoints(peps, showVHosts, showPorts)
	}

	// If we're generating JSON..
	if ctx.Bool("verbose") {
		if jsonOutput, err := json.MarshalIndent(publicEndpoints, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal public endoints: %s\n", err)
		} else {
			fmt.Println(string(jsonOutput))
		}
		return
	}

	// Give a message if there are no endpoints, or no port/vhosts endpoints.
	if len(publicEndpoints) == 0 {
		fmt.Fprintln(os.Stderr, "No public endpoints found")
		return
	}

	// Generate the output table.
	/*
	   Service            ServiceID                      Endpoint                Type       Protocol      Name                  Enabled
	   ├─opentsdb         bl8kffvyafgmpcnhir6pbif9o      opentsdb-reader         vhost      https         opentsdb              false
	   ├─HMaster          2fnhe4qmhypdirx28uhaf45tl      hbase-masterinfo-1      vhost      https         hbase                 false
	   ├─RabbitMQ         1ayd73wlfpi09y452kbu1wphx      rabbitmq admin          vhost      https         rabbitmq              false
	   ├─RabbitMQ         1ayd73wlfpi09y452kbu1wphx      rabbitmq admin          port       https         zenoss-1442:9090      true
	   ├─RabbitMQ         1ayd73wlfpi09y452kbu1wphx      rabbitmq                port       https         zenoss-1442:7777      true
	   ├─Zenoss.core      1h0hv34l78r0b11vxfsyzldgc      zproxy                  vhost      https         zenoss5               true
	   └─Zenoss.core      1h0hv34l78r0b11vxfsyzldgc      zproxy                  port       https         zenoss-1442:2222      true
	*/

	cmdSetTreeCharset(ctx, c.config)
	t := NewTable(ctx.String("show-fields"))
	t.IndentRow()

	for _, pep := range publicEndpoints {
		t.AddRow(map[string]interface{}{
			"Service":   pep.Service,
			"ServiceID": pep.ServiceID,
			"Endpoint":  pep.Application,
			"Type":      pep.EpType,
			"Protocol":  pep.Protocol,
			"Name":      pep.Name,
			"Enabled":   pep.Enabled,
		})
	}

	t.Padding = 6
	t.Print()
	return
}

func (c *ServicedCli) convertPublicEndpoints(peps []service.PublicEndpoint, showVHosts bool, showPorts bool) ([]PublicEndpoint, error) {

	result := []PublicEndpoint{}
	var pepType, proto, pepName string
	for _, pep := range peps {
		add := false
		if showVHosts && pep.VHostName != "" {
			pepType = "vhost"
			proto = "https"
			pepName = pep.VHostName
			add = true
		} else if showPorts && pep.PortAddress != "" {
			pepType = "port"
			proto = pep.Protocol
			pepName = pep.PortAddress
			add = true
		}
		if add {
			npep := NewPublicEndpoint(pep.ServiceName, pep.ServiceID, pep.Application, pepType, proto, pepName, pep.Enabled)
			result = append(result, npep)
		}
	}
	return result, nil
}

// Create a unified list of vhosts and port based public endpoints.
func (c *ServicedCli) createPublicEndpoints(ctx *cli.Context, services []service.Service,
	showVHosts bool, showPorts bool) ([]PublicEndpoint, error) {
	publicEndpoints := []PublicEndpoint{}

	// See if they provided the endpoint name
	epName := ""
	epFound := false
	if len(ctx.Args()) > 1 {
		epName = ctx.Args()[1]
	}

	// Iterate the list of services -> endpoints -> vhosts/ports
	for _, svc := range services {
		for _, ep := range svc.Endpoints {
			if epName != "" && epName != ep.Name {
				continue
			}
			epFound = true
			if showVHosts {
				for _, vhost := range ep.VHostList {
					publicEndpoint := NewPublicEndpoint(
						svc.Name,
						svc.ID,
						ep.Application,
						"vhost",
						"https",
						vhost.Name,
						vhost.Enabled,
					)
					publicEndpoints = append(publicEndpoints, publicEndpoint)
				}
			}
			if showPorts {
				for _, port := range ep.PortList {
					protocol := port.Protocol
					if protocol == "" {
						if port.UseTLS {
							protocol = "other-tls"
						} else {
							protocol = "other"
						}
					}
					publicEndpoint := NewPublicEndpoint(
						svc.Name,
						svc.ID,
						ep.Application,
						"port",
						protocol,
						port.PortAddr,
						port.Enabled,
					)
					publicEndpoints = append(publicEndpoints, publicEndpoint)
				}
			}
		}
	}

	if !epFound {
		return nil, fmt.Errorf("Endpoint '%s' not found", epName)
	}

	return publicEndpoints, nil
}

// List port public endpoints
// serviced service public-endpoints port list [SERVICEID] [ENDPOINTNAME]
func (c *ServicedCli) cmdPublicEndpointsPortList(ctx *cli.Context) {
	cmdPublicEndpointsList(c, ctx, false, true)
}

// Add a port public endpoint
// serviced service public-endpoints port add <SERVICEID> <ENDPOINTNAME> <PORTADDR> <PROTOCOL> <ENABLED>
func (c *ServicedCli) cmdPublicEndpointsPortAdd(ctx *cli.Context) {
	// Make sure we have each argument.
	if len(ctx.Args()) != 5 {
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	restart := ctx.Bool("restart")
	serviceid := ctx.Args()[0]
	endpointName := ctx.Args()[1]
	portAddr := ctx.Args()[2]
	protocol := ctx.Args()[3]
	isEnabled, err := strconv.ParseBool(ctx.Args()[4])
	if err != nil {
		fmt.Fprintln(os.Stderr, "The enabled flag must be true or false")
		return
	}

	// Determine if tls should be on.
	usetls := false
	switch protocol {
	case "http":
		break
	case "https":
		usetls = true
		break
	case "other":
		protocol = "" // Stored as an empty string.
	case "other-tls":
		protocol = "" // Stored as an empty string.
		usetls = true
		break
	default:
		fmt.Fprintln(os.Stderr, "The protocol must be one of: https, http, other-tls, other")
		return
	}

	// We need the serviceid, but they may have provided the service id or name.
	svc, _, err := c.searchForService(serviceid,ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	port, err := c.driver.AddPublicEndpointPort(svc.ID, endpointName, portAddr, usetls, protocol, isEnabled, restart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		fmt.Printf("%s\n", port.PortAddr)
	}
	return
}

// Remove a port public endpoint
// serviced service public-endpoints port remove <SERVICEID> <ENDPOINTNAME> <PORTADDR>
func (c *ServicedCli) cmdPublicEndpointsPortRemove(ctx *cli.Context) {
	// Make sure we have each argument.
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	serviceid := ctx.Args()[0]
	endpointName := ctx.Args()[1]
	portAddr := ctx.Args()[2]

	// We need the serviceid, but they may have provided the service id or name.
	svc, _, err := c.searchForService(serviceid,ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = c.driver.RemovePublicEndpointPort(svc.ID, endpointName, portAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		fmt.Printf("%s\n", portAddr)
	}
	return
}

// List vhost public endpoints
// serviced service public-endpoints vhost list [SERVICEID] [ENDPOINTNAME]
func (c *ServicedCli) cmdPublicEndpointsVHostList(ctx *cli.Context) {
	cmdPublicEndpointsList(c, ctx, true, false)
}

// Enable/Disable a port public endpoint
// serviced service public-endpoints port enable <SERVICEID> <ENDPOINTNAME> <PORTADDR> <true|false>
func (c *ServicedCli) cmdPublicEndpointsPortEnable(ctx *cli.Context) {
	// Make sure we have each argument.
	if len(ctx.Args()) != 4 {
		cli.ShowCommandHelp(ctx, "enable")
		return
	}

	serviceid := ctx.Args()[0]
	endpointName := ctx.Args()[1]
	portAddr := ctx.Args()[2]
	isEnabled, err := strconv.ParseBool(ctx.Args()[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, "The enabled flag must be true or false")
		return
	}

	// We need the serviceid, but they may have provided the service id or name.
	svc, _, err := c.searchForService(serviceid,ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = c.driver.EnablePublicEndpointPort(svc.ID, endpointName, portAddr, isEnabled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		fmt.Printf("%s\n", portAddr)
	}
	return
}

// Add a vhost public endpoint
// serviced service public-endpoints vhost add <SERVICEID> <ENDPOINTNAME> <VHOST> <ENABLED>"
func (c *ServicedCli) cmdPublicEndpointsVHostAdd(ctx *cli.Context) {
	// Make sure we have each argument.
	if len(ctx.Args()) != 4 {
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	restart := ctx.Bool("restart")
	serviceid := ctx.Args()[0]
	endpointName := ctx.Args()[1]
	vhostName := ctx.Args()[2]
	isEnabled, err := strconv.ParseBool(ctx.Args()[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, "The enabled flag must be true or false")
		return
	}

	// We need the serviceid, but they may have provided the service id or name.
	svc, _, err := c.searchForService(serviceid,ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	vhost, err := c.driver.AddPublicEndpointVHost(svc.ID, endpointName, vhostName, isEnabled, restart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		fmt.Printf("%s\n", vhost.Name)
	}
	return
}

// Remove a vhost public endpoint
// serviced service public-endpoints vhost remove <SERVICEID> <ENDPOINTNAME> <VHOST>
func (c *ServicedCli) cmdPublicEndpointsVHostRemove(ctx *cli.Context) {
	// Make sure we have each argument.
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	serviceid := ctx.Args()[0]
	endpointName := ctx.Args()[1]
	vhostName := ctx.Args()[2]

	// We need the serviceid, but they may have provided the service id or name.
	svc, _, err := c.searchForService(serviceid,ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = c.driver.RemovePublicEndpointVHost(svc.ID, endpointName, vhostName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		fmt.Printf("%s\n", vhostName)
	}

	return
}

// Enable/Disable a vhost public endpoint
// serviced service public-endpoints vhost enable <SERVICEID> <ENDPOINTNAME> <VHOST> true|false
func (c *ServicedCli) cmdPublicEndpointsVHostEnable(ctx *cli.Context) {
	// Make sure we have each argument.
	if len(ctx.Args()) != 4 {
		cli.ShowCommandHelp(ctx, "enable")
		return
	}

	serviceid := ctx.Args()[0]
	endpointName := ctx.Args()[1]
	vhostName := ctx.Args()[2]
	isEnabled, err := strconv.ParseBool(ctx.Args()[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, "The enabled flag must be true or false")
		return
	}

	// We need the serviceid, but they may have provided the service id or name.
	svc, _, err := c.searchForService(serviceid,ctx.Bool("no-prefix-match"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = c.driver.EnablePublicEndpointVHost(svc.ID, endpointName, vhostName, isEnabled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	} else {
		fmt.Printf("%s\n", vhostName)
	}
	return
}
