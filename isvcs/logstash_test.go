package isvcs

import (
	"fmt"
	"github.com/zenoss/serviced/dao"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func getTestingService() dao.ServiceDefinition {
	service := dao.ServiceDefinition{
		Name:        "testsvc",
		Description: "Top level service. This directory is part of a unit test.",
		LogFilters: map[string]string{
			"Pepe": "My Test Filter",
		},
		Services: []dao.ServiceDefinition{
			dao.ServiceDefinition{
				Name:    "s1",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageId: "ubuntu",
				LogFilters: map[string]string{
					"Pepe2": "My Second Filter",
				},
				ConfigFiles: map[string]dao.ConfigFile{
					"/etc/my.cnf": dao.ConfigFile{Filename: "/etc/my.cnf", Content: "\n# SAMPLE config file for mysql\n\n[mysqld]\n\ninnodb_buffer_pool_size = 16G\n\n"},
				},
				Endpoints: []dao.ServiceEndpoint{
					dao.ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Purpose:     "export",
					},
					dao.ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Purpose:     "import",
					},
				},
				LogConfigs: []dao.LogConfig{
					dao.LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
						Filters: []string{
							"Pepe",
						},
					},
				},
			},
			dao.ServiceDefinition{
				Name:    "s2",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageId: "ubuntu",
				Endpoints: []dao.ServiceEndpoint{
					dao.ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Purpose:     "export",
					},
				},
				LogConfigs: []dao.LogConfig{
					dao.LogConfig{
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
	services := make([]dao.ServiceDefinition, 1)
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
	services := make([]dao.ServiceDefinition, 1)
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
	tmpName := os.TempDir() + "/logstash_test.conf"
	writeLogStashConfigFile(filters, tmpName)

	// make sure the file exists
	_, err := os.Stat(tmpName)
	if err != nil {
		t.Error(fmt.Sprintf("Was unable to stat %s", tmpName))
	}

	// attempt to clean up after ourselves
	defer os.Remove(tmpName)

	// read the contents
	contents, err := ioutil.ReadFile(tmpName)
	if err != nil {
		t.Error(fmt.Sprintf("Unable to read output file %v", err))
	}

	// make sure our filter string is in it
	if !strings.Contains(string(contents), filters) {
		t.Error("Was unable to write the logstash conf file")
	}
}
