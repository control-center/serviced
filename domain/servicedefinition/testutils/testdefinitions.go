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

// +build unit integration

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
						VHostList:   []VHost{VHost{Name: "testhost"}},
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
