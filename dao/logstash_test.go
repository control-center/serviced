// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package dao

import (
	"fmt"
	"io/ioutil"
//	"os"
	"strings"
	"testing"
)

func getTestingService() ServiceDefinition {
	service := ServiceDefinition{
		Name:        "testsvc",
		Description: "Top level service. This directory is part of a unit test.",
		LogFilters: map[string]string{
			"Pepe": "My Test Filter",
		},
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name:    "s1",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageId: "ubuntu",
				LogFilters: map[string]string{
					"Pepe2": "My Second Filter",
				},
				ConfigFiles: map[string]ConfigFile{
					"/etc/my.cnf": ConfigFile{Filename: "/etc/my.cnf", Content: "\n# SAMPLE config file for mysql\n\n[mysqld]\n\ninnodb_buffer_pool_size = 16G\n\n"},
				},
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Purpose:     "import",
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
						Filters: []string{
							"Pepe",
						},
					},
				},
			},
			ServiceDefinition{
				Name:    "s2",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageId: "ubuntu",
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Purpose:     "export",
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
			},
		},
	}

	return service
}

func TestGettingFilterDefinitionsFromServiceDefinitions(t *testing.T) {
	services := make([]ServiceDefinition, 1)
	services[0] = getTestingService()
	filterDefs := getFilterDefinitions(services)

	// make sure we find the specific filter definition we are looking for
	if filterDefs["Pepe"] != "My Test Filter" {
		t.Error("Was unable to extract the filter definition")
	}

	// make sure the number matches the number we define
	if len(filterDefs) != 2 {
		t.Error("Found " + string(len(filterDefs)) + " instead of 2 filter definitions")
	}
}

func TestConstructingFilterString(t *testing.T) {
	services := make([]ServiceDefinition, 1)
	services[0] = getTestingService()
	filterDefs := getFilterDefinitions(services)
	filters := getFilters(services, filterDefs)
	testString := "My Test Filter"

	// make sure our test filter definition is in the constructed filters
	if !strings.Contains(filters, testString) {
		t.Error(fmt.Sprintf("Was unable to find %s in the filters", testString))
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
		t.Error("error calling writeLogStashConfigFile: %s", err)
		t.Fail()
	}

	// read the contents
	contents, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Error(fmt.Sprintf("Unable to read output file %v", err))
	}

	// make sure our filter string is in it
	if !strings.Contains(string(contents), filters) {
		t.Log("Read in contents: %s", string(contents))
		t.Log(filters)
		t.Error("Was unable to write the logstash conf file")
	}
}
