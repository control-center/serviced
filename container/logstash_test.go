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
package container

import (
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"

	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func getTestService() service.Service {
	return service.Service{
		ID:              "0",
		Name:            "Zenoss",
		Context:         nil,
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0, 0},
		ImageID:         "",
		PoolID:          "",
		DesiredState:    int(service.SVCStop),
		Launch:          "auto",
		Endpoints:       []service.ServiceEndpoint{},
		ParentServiceID: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/path/to/log/file",
				Type: "test",
				LogTags: []servicedefinition.LogTag{
					servicedefinition.LogTag{
						Name:  "pepe",
						Value: "foobar",
					},
				},
			},
			servicedefinition.LogConfig{
				Path: "/path/to/second/log/file",
				Type: "test2",
				LogTags: []servicedefinition.LogTag{
					servicedefinition.LogTag{
						Name:  "pepe",
						Value: "foobar",
					},
				},
			},
		},
	}
}

const logstashContainerDirectory = "/usr/local/serviced/resources/logstash"

func TestMakeSureTagsMakeItIntoTheJson(t *testing.T) {
	service := getTestService()

	tmp, err := ioutil.TempFile("/tmp", "test-logstash-")
	if err != nil {
		t.Errorf("Error creating temporary file error: %s", err)
		return
	}
	confFileLocation := tmp.Name()
	// go ahead and clean up
	defer func() {
		os.Remove(confFileLocation)
	}()

	if err := writeLogstashAgentConfig(confFileLocation, &service, "0", logstashContainerDirectory); err != nil {
		t.Errorf("Error writing config file %s", err)
		return
	}

	contents, err := ioutil.ReadFile(confFileLocation)
	if err != nil {
		t.Errorf("Error reading config file %s", err)
		return
	}

	if !strings.Contains(string(contents), "\"pepe\":\"foobar\"") {
		t.Errorf("Tags did not make it into the config file %s", string(contents))
		return
	}
}

func TestMakeSureConfigIsValidJSON(t *testing.T) {
	service := getTestService()

	tmp, err := ioutil.TempFile("/tmp", "test-logstash-")
	if err != nil {
		t.Errorf("Error creating temporary file error: %s", err)
		return
	}
	confFileLocation := tmp.Name()
	// go ahead and clean up
	defer func() {
		os.Remove(confFileLocation)
	}()

	if err := writeLogstashAgentConfig(confFileLocation, &service, "0", logstashContainerDirectory); err != nil {
		t.Errorf("Error writing config file %s", err)
		return
	}

	contents, err := ioutil.ReadFile(confFileLocation)
	if err != nil {
		t.Errorf("Error reading config file %s", err)
		return
	}

	var dat map[string]interface{}
	if err := json.Unmarshal(contents, &dat); err != nil {
		t.Errorf("Error decoding config file %s with err %s", string(contents), err)
		return
	}

	glog.V(1).Infof("encoded file %v", dat)

	// make sure path to logfile is present
	if !strings.Contains(string(contents), "path/to/log/file") {
		t.Errorf("The logfile path was not in the configuration: err %s, %s", err, string(contents))
	}
}

func TestDontWriteToNilMap(t *testing.T) {
	service := service.Service{
		ID:              "0",
		Name:            "Zenoss",
		Context:         nil,
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       0,
		InstanceLimits:  domain.MinMax{0, 0, 0},
		ImageID:         "",
		PoolID:          "",
		DesiredState:    int(service.SVCStop),
		Launch:          "auto",
		Endpoints:       []service.ServiceEndpoint{},
		ParentServiceID: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/path/to/log/file",
				Type: "test",
			},
		},
	}

	tmp, err := ioutil.TempFile("/tmp", "test-logstash-")
	if err != nil {
		t.Errorf("Error creating temporary file error: %s", err)
		return
	}
	confFileLocation := tmp.Name()
	// go ahead and clean up
	defer func() {
		os.Remove(confFileLocation)
	}()

	if err := writeLogstashAgentConfig(confFileLocation, &service, "0", logstashContainerDirectory); err != nil {
		t.Errorf("Writing with empty tags produced an error %s", err)
		return
	}
}
