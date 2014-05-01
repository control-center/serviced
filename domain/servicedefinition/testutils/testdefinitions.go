// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package testutils

import (
	"github.com/zenoss/serviced/commons"
	. "github.com/zenoss/serviced/domain/servicedefinition"
)

var ValidSvcDef *ServiceDefinition

func init() {
	// should we make the service definition from the dao test package public and use here?
	ValidSvcDef = CreateValidServiceDefinition()
}

func CreateValidServiceDefinition() *ServiceDefinition {
	// should we make the service definition from the dao test package public and use here?
	return &ServiceDefinition{
		Name: "testsvc",
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name: "s1",
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Name:        "www",
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					ServiceEndpoint{
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
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
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
