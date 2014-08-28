// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package testutils

import (
	"github.com/control-center/serviced/commons"
	. "github.com/control-center/serviced/domain/servicedefinition"
)

//ValidSvcDef used for testing
var ValidSvcDef *ServiceDefinition

func init() {
	// should we make the service definition from the dao test package public and use here?
	ValidSvcDef = CreateValidServiceDefinition()
}

//CreateValidServiceDefinition create a populated ServiceDefinition for testing
func CreateValidServiceDefinition() *ServiceDefinition {
	// should we make the service definition from the dao test package public and use here?
	return &ServiceDefinition{
		Name:   "testsvc",
		Launch: "auto",
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name:   "s1",
				Launch: "auto",
				Endpoints: []EndpointDefinition{
					EndpointDefinition{
						Name:        "www",
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					EndpointDefinition{
						Name:        "websvc",
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Purpose:     "import",
						AddressConfig: AddressResourceConfig{
							Port:     8081,
							Protocol: commons.TCP,
						},
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
				Snapshot: SnapshotCommands{
					Pause:  "echo pause",
					Resume: "echo resume",
				},
			},
			ServiceDefinition{
				Name:    "s2",
				Launch:  "auto",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				Endpoints: []EndpointDefinition{
					EndpointDefinition{
						Name:        "websvc",
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Purpose:     "export",
						VHosts:      []string{"testhost"},
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
				Snapshot: SnapshotCommands{
					Pause:  "echo pause",
					Resume: "echo resume",
				},
			},
		},
	}

}
