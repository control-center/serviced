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

// +build unit

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package facade

import (
	"github.com/control-center/serviced/domain/servicedefinition"

	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func getTestingService() servicedefinition.ServiceDefinition {
	service := servicedefinition.ServiceDefinition{
		Name:        "testsvc",
		Description: "Top level service. This directory is part of a unit test.",
		LogFilters: map[string]string{
			"Pepe": "My Test Filter",
		},
		Services: []servicedefinition.ServiceDefinition{
			servicedefinition.ServiceDefinition{
				Name:    "s1",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				ConfigFiles: map[string]servicedefinition.ConfigFile{
					"/etc/my.cnf": servicedefinition.ConfigFile{Filename: "/etc/my.cnf", Content: "\n# SAMPLE config file for mysql\n\n[mysqld]\n\ninnodb_buffer_pool_size = 16G\n\n"},
				},
				Endpoints: []servicedefinition.EndpointDefinition{
					servicedefinition.EndpointDefinition{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					servicedefinition.EndpointDefinition{
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Purpose:     "import",
					},
				},
				LogConfigs: []servicedefinition.LogConfig{
					servicedefinition.LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
						Filters: []string{
							"Pepe",
						},
					},
				},
				LogFilters: map[string]string{
					"Pepe2": "My Second Filter",
				},
				Services: []servicedefinition.ServiceDefinition{
					servicedefinition.ServiceDefinition{
						Name:    "s1child",
						Command: "/usr/bin/python -m SimpleHTTPServer",
						ImageID: "ubuntu",
						LogConfigs: []servicedefinition.LogConfig{
							servicedefinition.LogConfig{
								Path: "/tmp/foo2",
								Type: "foo2",
								Filters: []string{
									"Pepe4",
								},
							},
						},
						LogFilters: map[string]string{
							"Pepe4": "My Fourth Filter",
						},
					},

				},
			},
			servicedefinition.ServiceDefinition{
				Name:    "s2",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				Endpoints: []servicedefinition.EndpointDefinition{
					servicedefinition.EndpointDefinition{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Purpose:     "export",
					},
				},
				LogConfigs: []servicedefinition.LogConfig{
					servicedefinition.LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
						Filters: []string{
							"Pepe3",
						},
					},
				},
				LogFilters: map[string]string{
					"Pepe3": "My Third Filter",
				},
				Services: []servicedefinition.ServiceDefinition{
					servicedefinition.ServiceDefinition{
						Name:    "s2child",
						Command: "/usr/bin/python -m SimpleHTTPServer",
						ImageID: "ubuntu",
						LogConfigs: []servicedefinition.LogConfig{
							servicedefinition.LogConfig{
								Path: "/tmp/foo2",
								Type: "foo2",
								Filters: []string{
									"Pepe4",
								},
							},
						},
						LogFilters: map[string]string{
							"Pepe4": "My Fourth Filter",
						},
					},
				},
			},
		},
	}

	return service
}

func TestGettingFilterDefinitionsFromServiceDefinitions(t *testing.T) {
	services := make([]servicedefinition.ServiceDefinition, 1)
	services[0] = getTestingService()
	filterDefs := getFilterDefinitions(services)

	// make sure we find the specific filter definition we are looking for
	if filterDefs["Pepe"] != "My Test Filter" {
		t.Error("Was unable to extract the filter definition")
	}

	// make sure the number matches the number we define
	if len(filterDefs) != 4 {
		t.Error(fmt.Sprintf("Found %d instead of 2 filter definitions", len(filterDefs)))
	}
}

func TestConstructingFilterString(t *testing.T) {
	services := make([]servicedefinition.ServiceDefinition, 1)
	services[0] = getTestingService()
	filterDefs := getFilterDefinitions(services)
	typeFilter := []string{}
	filters := getFilters(services, filterDefs, &typeFilter)
	testString := "My Test Filter"

	// make sure our test filter definition is in the constructed filters
	if !strings.Contains(filters, testString) {
		t.Error(fmt.Sprintf("Was unable to find %s in the filters", testString))
	}
}

func TestNoDuplicateFilters(t *testing.T) {
	services := make([]servicedefinition.ServiceDefinition, 1)
	services[0] = getTestingService()
	filterDefs := getFilterDefinitions(services)
	typeFilter := []string{}
	filters := getFilters(services, filterDefs, &typeFilter)

	filterCount := strings.Count(filters, "if [type] == \"foo2\"")
	if filterCount != 1 {
		t.Error(fmt.Sprintf("expected only 1 filter for 'foo2', but found %d: filters=%s", filterCount, filters))
	}
}

func TestWritingConfigFile(t *testing.T) {
	filters := "This is my test filter"
	tmpfile, err := ioutil.TempFile("", "logstash_test.conf")
	t.Logf("Created tempfile: %s", tmpfile.Name())
	if err != nil {
		t.Logf("could not create tempfile: %s", err)
		t.FailNow()
	}
	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write([]byte("${FILTER_SECTION}"))
	if err != nil {
		t.Logf("%s", err)
		t.FailNow()
	}
	err = tmpfile.Sync()
	if err != nil {
		t.Logf("%s", err)
		t.FailNow()
	}

	if err = writeLogStashConfigFile(filters, tmpfile.Name()); err != nil {
		t.Errorf("error calling writeLogStashConfigFile: %s", err)
		t.Fail()
	}

	// read the contents
	contents, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Error(fmt.Sprintf("Unable to read output file %v", err))
	}

	// make sure our filter string is in it
	if !strings.Contains(string(contents), filters) {
		t.Logf("Read in contents: %s", string(contents))
		t.Log(filters)
		t.Error("Was unable to write the logstash conf file")
	}
}
