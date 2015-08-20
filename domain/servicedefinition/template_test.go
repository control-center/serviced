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

// +build integration

package servicedefinition

import (
	"github.com/control-center/serviced/commons"

	"fmt"
	"log"
	"sort"
	"testing"
)

var testSvc *ServiceDefinition

func init() {

	// Test definition should match on disk (filesystem based) definition
	testSvc = &ServiceDefinition{
		Name:        "testsvc",
		Description: "Top level service. This directory is part of a unit test.",
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name:    "postmetrics",
				Command: "/loadavg.sh",
				ImageID: "ubuntu",
				ConfigFiles: map[string]ConfigFile{
					"/loadavg.sh": ConfigFile{Owner: "root:root", Filename: "/loadavg.sh", Permissions: "0775", Content: `#!/bin/bash

interval=1


if [ ! -x /usr/bin/curl ]; then
	if ! apt-get install -y curl ;then
		echo "Could not install curl"
		exit 1
	fi

fi
echo "posting loadavg at $interval second(s) interval"
while :
	do
	now=` + "`date +%s`" + `
	value=` + "`cat /proc/loadavg | cut -d ' ' -f 1`" + `
	data="{\"control\":{\"type\":null,\"value\":null},\"metrics\":[{\"metric\":\"loadavg\",\"timestamp\":$now,\"value\":$value,\"tags\":{\"name\":\"value\"}}]}"

	output=` + "`curl -s -XPOST -H \"Content-Type: application/json\" -d \"$data\" \"$CONTROLPLANE_CONSUMER_URL\"`" + `

	if ! [[ "$output" == *OK* ]]
	then
		echo "failure";
	fi

	sleep $interval
done
`},
				},
			},
			ServiceDefinition{
				Name:    "s1",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				ConfigFiles: map[string]ConfigFile{
					"/etc/my.cnf": ConfigFile{Owner: "root:root", Filename: "/etc/my.cnf", Permissions: "0660", Content: "\n# SAMPLE config file for mysql\n\n[mysqld]\n\ninnodb_buffer_pool_size = 16G\n\n"},
				},
				Endpoints: []EndpointDefinition{
					EndpointDefinition{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Name:        "www",
						Purpose:     "export",
					},
					EndpointDefinition{
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Name:        "websvc",
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
				Volumes: []Volume{
					{
						Owner:             "zenoss:zenoss",
						Permission:        "0777",
						ResourcePath:      "test1",
						ContainerPath:     "/test1",
						InitContainerPath: "/initFromHere",
						Type:              "",
					}, {
						Owner:             "zenoss:zenoss",
						Permission:        "0777",
						ResourcePath:      "test2",
						ContainerPath:     "/test2",
						InitContainerPath: "",
						Type:              "",
					},
				},
			},
			ServiceDefinition{
				Name:    "s2",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				ConfigFiles: map[string]ConfigFile{
					"/foo/bar.txt": ConfigFile{
						Filename:    "/foo/bar.txt",
						Owner:       "zenoss:zenoss",
						Permissions: "660",
						Content:     "baz\n",
					},
				},
				Endpoints: []EndpointDefinition{
					EndpointDefinition{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Name:        "websvc",
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

// This function checks if the given ServiceDefinition is equivalent. True is
// returned if true, false is returned otherwise. A non-empty message is returned
// that identifies the first inequality that was discovered.
func (a *ServiceDefinition) equals(b *ServiceDefinition) (identical bool, msg string) {

	if a.Name != b.Name {
		return false, fmt.Sprintf("Names are not equal %s != %s", a.Name, b.Name)
	}
	if a.Description != b.Description {
		return false, fmt.Sprintf("Descriptions are not equal %s != %s", a.Description, b.Description)
	}
	if a.ImageID != b.ImageID {
		return false, fmt.Sprintf("ImageIDs are not equal %s != %s", a.ImageID, b.ImageID)
	}
	if a.Command != b.Command {
		return false, fmt.Sprintf("Commands are not equal %s != %s", a.Command, b.Command)
	}
	if len(a.Endpoints) != len(b.Endpoints) {
		return false, fmt.Sprintf("Number of endpoints differ between %s [%d] and %s [%d]",
			a.Name, len(a.Endpoints), b.Name, len(b.Endpoints))
	}
	if len(a.Services) != len(b.Services) {
		return false, fmt.Sprintf("Number of sub services differ between %s [%d] and %s [%d]",
			a.Name, len(a.Services), b.Name, len(b.Services))
	}
	if len(a.Volumes) != len(b.Volumes) {
		return false, fmt.Sprintf("Number of volumes differ between %s [%d] and %s [%d]",
			a.Name, len(a.Volumes), b.Name, len(b.Volumes))
	}

	sort.Sort(ServiceDefinitionByName(a.Services))
	sort.Sort(ServiceDefinitionByName(b.Services))

	for i := range a.Services {
		identical, msg := a.Services[i].equals(&b.Services[i])
		if identical != true {
			return identical, msg
		}
	}

	// check config files
	if len(a.ConfigFiles) != len(b.ConfigFiles) {
		log.Printf("s1 :%v  \n\ns2 %s", a.ConfigFiles, b.ConfigFiles)
		return false, fmt.Sprintf("%s has %d configs, %s has %d configs",
			a, len(a.ConfigFiles), b, len(b.ConfigFiles))
	}
	for filename, confFile := range a.ConfigFiles {
		if _, ok := b.ConfigFiles[filename]; !ok {
			return false, fmt.Sprintf("%s has configFile %s, but %s does not", a, filename, b)
		}
		if confFile != b.ConfigFiles[filename] {
			return false, fmt.Sprintf("ConfigFile mismatch %s, a: \n\n%s \n\nb: \n\n%s", filename, confFile, b.ConfigFiles[filename])
		}
	}

	// check snapshot
	if a.Snapshot.Pause != b.Snapshot.Pause {
		return false, fmt.Sprintf("Snapshot pause commands are not equal %s != %s", a.Snapshot.Pause, b.Snapshot.Pause)
	}
	if a.Snapshot.Resume != b.Snapshot.Resume {
		return false, fmt.Sprintf("Snapshot resume commands are not equal %s != %s", a.Snapshot.Resume, b.Snapshot.Resume)
	}

	return true, ""
}

func TestServiceDefinitionFromPath(t *testing.T) {

	sd, err := BuildFromPath("./testsvc")

	if err != nil {
		t.Fatalf("Problem parsing template: %s", err)
	}
	identical, msg := testSvc.equals(sd)
	if !identical {
		t.Fatalf(msg)
	}

}
