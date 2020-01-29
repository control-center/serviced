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
	"io/ioutil"
	"os"
	"strings"

	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/service"

	. "gopkg.in/check.v1"
)

var _ = Suite(&LogStashTest{})

type LogStashTest struct {
	cache        *serviceCache
	//serviceStore *servicemocks.Store
	//unusedCTX    datastore.Context
}

func (t *LogStashTest) SetUpTest(c *C) {
	t.cache = NewServiceCache()
	//t.serviceStore = &servicemocks.Store{}
}

func (t *LogStashTest) TearDownTest(c *C) {
	t.cache = nil
	//t.serviceStore = nil
}


func getTestServiceDefinition() servicedefinition.ServiceDefinition {
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


func (t *LogStashTest) TestGettingFilterDefinitionsFromServiceDefinitions(c *C) {
	services := make([]servicedefinition.ServiceDefinition, 1)
	services[0] = getTestServiceDefinition()
	filterDefs := getFilterDefinitions(services)

	// make sure we find the specific filter definition we are looking for
	c.Assert(filterDefs["Pepe"], Equals, "My Test Filter")

	// make sure the number matches the number we define
	c.Assert(len(filterDefs), Equals, 4)
}

func (t *LogStashTest) Test_getFilterSection_WithNoServiceLogs(c *C) {
	logInfo := serviceLogInfo{}
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: "filter1",
			Version: "0",
			Filter: "some filter string",
		},
	}

	logFiles := []string{}
	result := getFilterSection(logInfo, logFilters, &logFiles)

	c.Assert(len(result), Equals, 0)
	c.Assert(len(logFiles), Equals, 0)
}

func (t *LogStashTest) Test_getFilterSection_WithNoLogFilters(c *C) {
	logInfo := serviceLogInfo{
		Name: "service1",
		Version: "1.0",
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/var/log/something",
				Filters: []string{"filter1"},
			},
		},
	}
	logFilters := []*logfilter.LogFilter{}

	logFiles := []string{}
	result := getFilterSection(logInfo, logFilters, &logFiles)

	c.Assert(len(result), Equals, 0)
	c.Assert(len(logFiles), Equals, 0)
}

func (t *LogStashTest) Test_getFilterSection_WithUndefinedFilter(c *C) {
	version := "1.0"
	logInfo := serviceLogInfo{
		Name: "service1",
		Version: version,
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/var/log/something",
				Filters: []string{"undefined"},
			},
		},
	}
	filterName := "filter1"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: version,
			Filter: "fitler me something",
		},
	}

	logFiles := []string{}
	result := getFilterSection(logInfo, logFilters, &logFiles)

	c.Assert(len(result), Equals, 0)
	c.Assert(len(logFiles), Equals, 0)
}

func (t *LogStashTest) Test_getFilterSection_Simple(c *C) {
	version := "1.0"
	filterName1 := "filter1"
	filterName2 := "filter2"
	logInfo := serviceLogInfo{
		Name: "service1",
		Version: version,
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/var/log/something",
				Filters: []string{filterName1},
			},
			servicedefinition.LogConfig{
				Path: "/var/log/something/else",
				Filters: []string{filterName2},
			},
		},
	}
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName1,
			Version: version,
			Filter: "filter def 1",
		},
		&logfilter.LogFilter{
			Name: filterName2,
			Version: version,
			Filter: "filter def 2",
		},
	}

	logFiles := []string{}
	result := getFilterSection(logInfo, logFilters, &logFiles)

	c.Assert(len(result), Not(Equals), 0)
	c.Assert(strings.Contains(result, logFilters[0].Filter), Equals, true)
	c.Assert(strings.Contains(result, logFilters[1].Filter), Equals, true)

	c.Assert(len(logFiles), Equals, 2)
	c.Assert(logFiles[0], Equals, logInfo.LogConfigs[0].Path)
	c.Assert(logFiles[1], Equals, logInfo.LogConfigs[1].Path)
}

func (t *LogStashTest) Test_getFilterSection_NoDups(c *C) {
	version := "1.0"
	filterName1 := "filter1"
	filterName2 := "filter2"
	logInfo := serviceLogInfo{
		Name: "service1",
		Version: version,
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/var/log/something",
				Filters: []string{filterName1},
			},
			servicedefinition.LogConfig{
				Path: "/var/log/something/else",
				Filters: []string{filterName2},
			},
		},
	}
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName1,
			Version: version,
			Filter: "filter def 1",
		},
		&logfilter.LogFilter{
			Name: filterName2,
			Version: version,
			Filter: "filter def 2",
		},
	}

	// Add the first path and verify it's not added to the result
	logFiles := []string{logInfo.LogConfigs[0].Path}
	result := getFilterSection(logInfo, logFilters, &logFiles)

	c.Assert(len(result), Not(Equals), 0)
	c.Assert(strings.Contains(result, logFilters[1].Filter), Equals, true)
	c.Assert(strings.Contains(result, logFilters[0].Filter), Equals, false)

	c.Assert(len(logFiles), Equals, 2)
	c.Assert(logFiles[0], Equals, logInfo.LogConfigs[0].Path)
	c.Assert(logFiles[1], Equals, logInfo.LogConfigs[1].Path)
}

func (t *LogStashTest) Test_addServiceLogs_WithNoServices(c *C) {
	serviceLogs := map[string]serviceLogInfo{}

	addServiceLogs("1.0", []service.Service{}, serviceLogs)

	c.Assert(len(serviceLogs), Equals, 0)
}

func (t *LogStashTest) Test_addServiceLogs_WithNoLogFiles(c *C) {
	version := "1.0"
	serviceLogs := map[string]serviceLogInfo{}
	svcs := []service.Service{
		service.Service{
			ID: "id1",
			Name: "service1",
			Version: version,
			LogConfigs: []servicedefinition.LogConfig{},
		},
	}
	addServiceLogs(version, svcs, serviceLogs)

	c.Assert(len(serviceLogs), Equals, 0)
}

func (t *LogStashTest) Test_addServiceLogs_WithNewServices(c *C) {
	version := "1.0"
	svcs := getTestServices(version)
	serviceLogs := map[string]serviceLogInfo{}

	addServiceLogs(version, svcs, serviceLogs)

	c.Assert(len(serviceLogs), Equals, len(svcs))
}

// Verify that when serviceLogs already contains older services, then those services are
// replaced by newer ones
func (t *LogStashTest) Test_addServiceLogs_ReplacesOlderServices(c *C) {
	newVersion := "2.0.1"
	svcs := getTestServices(newVersion)

	oldVersion := "2.0.0"
	serviceLogs := getTestServiceLogs(oldVersion)

	addServiceLogs(svcs[0].Version, svcs, serviceLogs)

	c.Assert(len(serviceLogs), Equals, len(svcs))

	logInfo, ok := serviceLogs["service1"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, newVersion)
	c.Assert(logInfo.LogConfigs[0].Filters, DeepEquals, svcs[0].LogConfigs[0].Filters)

	logInfo, ok = serviceLogs["service2"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, newVersion)
	c.Assert(logInfo.LogConfigs[0].Filters, DeepEquals, svcs[1].LogConfigs[0].Filters)
}

// Verify that when serviceLogs already has newer services, then those services are not
// replaced by older ones
func (t *LogStashTest) Test_addServiceLogs_KeepsNewerServices(c *C) {
	oldVersion := "1.0.1"
	svcs := getTestServices(oldVersion)

	newVersion := "2.0.0"
	serviceLogs := getTestServiceLogs(newVersion)

	addServiceLogs(svcs[0].Version, svcs, serviceLogs)

	c.Assert(len(serviceLogs), Equals, len(svcs))

	logInfo, ok := serviceLogs["service1"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, newVersion)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "log-filter1")

	logInfo, ok = serviceLogs["service2"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, newVersion)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "log-filter2")
}

// Verify that when serviceLogs already has services with the same version, then those services are
// replaced by ones in the new call to addServiceLogs
func (t *LogStashTest) Test_addServiceLogs_UpdatesServices(c *C) {
	version := "1.0.1"
	svcs := getTestServices(version)
	serviceLogs := getTestServiceLogs(version)

	addServiceLogs(svcs[0].Version, svcs, serviceLogs)

	c.Assert(len(serviceLogs), Equals, len(svcs))

	logInfo, ok := serviceLogs["service1"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, version)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "log-filter1")

	logInfo, ok = serviceLogs["service2"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, version)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "log-filter2")
}

func (t *LogStashTest) Test_addServiceLogs_AccumulatesServices(c *C) {
	version1 := "1.0.1"
	svcs := getTestServices(version1)
	svcs[0].Name = "otherService1"
	svcs[1].Name = "otherService2"

	version2 := "1.0.2"
	serviceLogs := getTestServiceLogs(version2)

	addServiceLogs(svcs[0].Version, svcs, serviceLogs)

	c.Assert(len(serviceLogs), Equals, 4)

	logInfo, ok := serviceLogs["otherService1"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, version1)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "svc-filter1")

	logInfo, ok = serviceLogs["otherService2"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, version1)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "svc-filter2")

	logInfo, ok = serviceLogs["service1"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, version2)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "log-filter1")

	logInfo, ok = serviceLogs["service2"]
	c.Assert(ok, Equals, true)
	c.Assert(logInfo.Version, Equals, version2)
	c.Assert(logInfo.LogConfigs[0].Filters[0], Equals, "log-filter2")
}

func (t *LogStashTest) Test_findNewestFilter_WithNoFilters(c *C) {
	logFilters := []*logfilter.LogFilter{}
	logInfo := serviceLogInfo{
		Version: "1.0",
	}
	result, found := findNewestFilter("filterName", logInfo, logFilters)

	c.Assert(found, Equals, false)
	c.Assert(len(result), Equals, 0)
}

func (t *LogStashTest) Test_findNewestFilter_SimpleMatch(c *C) {
	filterName := "filter1"
	version := "1.0"
	expectedFilter := "filterValue"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: version,
			Filter: expectedFilter,
		},
	}
	logInfo := serviceLogInfo{
		Version: version,
	}

	result, found := findNewestFilter(filterName, logInfo, logFilters)

	c.Assert(found, Equals, true)
	c.Assert(result, Equals, expectedFilter)
}

func (t *LogStashTest) Test_findNewestFilter_IgnoreOlder(c *C) {
	filterName := "filter1"
	version := "1.0"
	expectedFilter := "filterValue"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: "0.9",
			Filter: "wrong filterValue",
		},
		&logfilter.LogFilter{
			Name: filterName,
			Version: version,
			Filter: expectedFilter,
		},
	}
	logInfo := serviceLogInfo{
		Version: version,
	}

	result, found := findNewestFilter(filterName, logInfo, logFilters)

	c.Assert(found, Equals, true)
	c.Assert(result, Equals, expectedFilter)
}

func (t *LogStashTest) Test_findNewestFilter_IgnoreNewer(c *C) {
	filterName := "filter1"
	version := "1.0"
	expectedFilter := "filterValue"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: "2.0",
			Filter: "wrong filterValue",
		},
		&logfilter.LogFilter{
			Name: filterName,
			Version: version,
			Filter: expectedFilter,
		},
	}
	logInfo := serviceLogInfo{
		Version: version,
	}

	result, found := findNewestFilter(filterName, logInfo, logFilters)

	c.Assert(found, Equals, true)
	c.Assert(result, Equals, expectedFilter)
}

func (t *LogStashTest) Test_findNewestFilter_PickFromMixedVersions(c *C) {
	filterName := "filter1"
	expectedFilter := "filterValue"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: "1.0",
			Filter: "wrong filterValue",
		},
		&logfilter.LogFilter{
			Name: filterName,
			Version: "2.0",
			Filter: expectedFilter,
		},
	}
	logInfo := serviceLogInfo{
		Version: "1.5",
	}

	result, found := findNewestFilter(filterName, logInfo, logFilters)

	c.Assert(found, Equals, true)
	c.Assert(result, Equals, expectedFilter)
}

func (t *LogStashTest) Test_findNewestFilter_PickFromOlderVersions(c *C) {
	filterName := "filter1"
	expectedFilter := "filterValue"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: "0.9",
			Filter: "wrong filterValue",
		},
		&logfilter.LogFilter{
			Name: filterName,
			Version: "1.0",
			Filter: expectedFilter,
		},
	}
	logInfo := serviceLogInfo{
		Version: "1.5",
	}

	result, found := findNewestFilter(filterName, logInfo, logFilters)

	c.Assert(found, Equals, true)
	c.Assert(result, Equals, expectedFilter)
}

func (t *LogStashTest) Test_findNewestFilter_PickFromNewerVersions(c *C) {
	filterName := "filter1"
	expectedFilter := "filterValue"
	logFilters := []*logfilter.LogFilter{
		&logfilter.LogFilter{
			Name: filterName,
			Version: "2.0",
			Filter: "wrong filterValue",
		},
		&logfilter.LogFilter{
			Name: filterName,
			Version: "3.0",
			Filter: expectedFilter,
		},
	}
	logInfo := serviceLogInfo{
		Version: "1.5",
	}

	result, found := findNewestFilter(filterName, logInfo, logFilters)

	c.Assert(found, Equals, true)
	c.Assert(result, Equals, expectedFilter)
}

func  (t *LogStashTest) TestWritingConfigFile(c *C) {
	filters := "This is my test filter"
	auditLogSection := "Audit Log Section string"
	stdoutSection := "Stdout Section string"
	tmpfile, err := ioutil.TempFile("", "logstash_test.conf")
	c.Logf("Created tempfile: %s", tmpfile.Name())
	c.Assert(err, IsNil)

	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write([]byte("${FILTER_SECTION}"))
	c.Assert(err, IsNil)
	_, err = tmpfile.Write([]byte("${AUDIT_SECTION}"))
	c.Assert(err, IsNil)
	_, err = tmpfile.Write([]byte("${STDOUT_SECTION}"))
	c.Assert(err, IsNil)
	err = tmpfile.Sync()
	c.Assert(err, IsNil)

	err = writeLogStashConfigFile(filters, auditLogSection, tmpfile.Name())
	c.Assert(err, IsNil)

	// read the contents
	contents, err := ioutil.ReadFile(tmpfile.Name())
	c.Assert(err, IsNil)

	// make sure our filter and auditLogSection string is in it
	// and stdoutSection is not in it
	if !strings.Contains(string(contents), filters) && !strings.Contains(string(contents), auditLogSection) && strings.Contains(string(contents), stdoutSection) {
		c.Logf("Read in contents: %s", string(contents))
		c.Log(filters)
		c.Error("Was unable to write the logstash conf file")

	}
}

func getTestServices(version string) []service.Service {
	return []service.Service{
		service.Service{
			ID: "id1",
			Name: "service1",
			Version: version,
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					Path: "/var/log/log1",
					Filters: []string{"svc-filter1"},
				},

			},
		},
		service.Service{
			ID: "id2",
			Name: "service2",
			Version: version,
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					Path: "/var/log/log2",
					Filters: []string{"svc-filter2"},
				},
			},
		},
	}
}

func getTestServiceLogs(version string) map[string]serviceLogInfo {
	return map[string]serviceLogInfo{
		"service1": serviceLogInfo{
			Name: "service1",
			ID: "id1",
			Version: version,
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					Path: "/var/log/log1",
					Filters: []string{"log-filter1"},
				},
			},
		},
		"service2": serviceLogInfo{
			Name: "service2",
			ID: "id2",
			Version: version,
			LogConfigs: []servicedefinition.LogConfig{
				servicedefinition.LogConfig{
					Path: "/var/log/log2",
					Filters: []string{"log-filter2"},
				},
			},
		},
	}
}
