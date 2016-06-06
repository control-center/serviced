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

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/domain/service"
)

// The vhost and port public endpoint structures are different, so we'll
// make a unified structure for output that matches the UI table.
type PublicEndpoint struct {
    Service string
    ServiceID string
    Endpoint string
    EpType string
    Protocol string
    Name string
    Enabled bool
}

func NewPublicEndpoint(service string, serviceID string, endpoint string, epType string,
                           protocol string, name string, enabled bool) PublicEndpoint {
	return PublicEndpoint{
        Service   : service,
        ServiceID : serviceID,
        Endpoint  : endpoint,
        EpType    : epType,
        Protocol  : protocol,
        Name      : name,
        Enabled   : enabled,
	}
}

// serviced service public-endpoints
func (c *ServicedCli) cmdPublicEndpointList(ctx *cli.Context) {
    var services []service.Service

	if len(ctx.Args()) > 0 {
        // Provided the service id/name.
		svc, err := c.searchForService(ctx.Args()[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
        services = []service.Service{*svc}
    } else {
        // Showing all service ports/vhosts.
        var err error
        services, err = c.driver.GetServices()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Unable to get services: %s\n", err)
            return
        } else if services == nil || len(services) == 0 {
            fmt.Fprintln(os.Stderr, "no services found")
            return
        }
    }
    
    // Get the list of public endpoints requested.
    publicEndpoints, err := c.getPublicEndpoints(ctx, services)
    // If there was an error getting the endpoints, show the error now.
    if err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        return
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
            "Endpoint":  pep.Endpoint,
            "Type":      pep.EpType,
            "Protocol":  pep.Protocol,
            "Name":      pep.Name,
            "Enabled":   pep.Enabled,
        })
    }

    t.Padding = 6
    t.Print()
}

// Create a unified list of vhosts and port based public endpoints.
func (c *ServicedCli) getPublicEndpoints(ctx *cli.Context, services []service.Service) ([]PublicEndpoint, error) {
    // If they specify only vhosts/ports, show those.  If they didn't specify
    // either then both are shown.
    showVHosts := ctx.Bool("vhosts") || (!ctx.Bool("vhosts") && !ctx.Bool("ports"))
    showPorts := ctx.Bool("ports") || (!ctx.Bool("vhosts") && !ctx.Bool("ports"))

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
                        ep.Name,
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
                    publicEndpoint := NewPublicEndpoint(
                        svc.Name,
                        svc.ID,
                        ep.Name,
                        "port",
                        port.Protocol,
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