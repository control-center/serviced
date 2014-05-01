// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.
package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/servicedefinition"

	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGetLogstashBindMounts(t *testing.T) {
	bindMounts := getLogstashBindMounts("pepe")
	pieces := strings.Split(bindMounts, "-v")
	// the first part is the logstash stuff mount and the second
	// is the config file
	configMount := strings.TrimSpace(pieces[2])
	glog.V(1).Infof("configMount: %s", configMount)
	if configMount != "pepe:"+LOGSTASH_CONTAINER_CONFIG {
		t.Errorf("String: %s Was not equal to %s", configMount, "pepe:"+LOGSTASH_CONTAINER_CONFIG)
	}
}

func getTestService() dao.Service {
	return dao.Service{
		Id:              "0",
		Name:            "Zenoss",
		Context:         "",
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       domain.MinMax{0,0},
		ImageId:         "",
		PoolId:          "",
		DesiredState:    0,
		Launch:          "auto",
		Endpoints:       []servicedefinition.ServiceEndpoint{},
		ParentServiceId: "",
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

func TestMakeSureTagsMakeItIntoTheJson(t *testing.T) {
	service := getTestService()
	confFileLocation, err := writeLogstashAgentConfig(&service)

	if err != nil {
		t.Errorf("Error writing config file %s", err)
		return
	}

	// go ahead and clean up
	defer func() {
		os.Remove(confFileLocation)
	}()
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
	confFileLocation, err := writeLogstashAgentConfig(&service)

	if err != nil {
		t.Errorf("Error writing config file %s", err)
		return
	}

	// go ahead and clean up
	defer func() {
		os.Remove(confFileLocation)
	}()
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
	service := dao.Service{
		Id:              "0",
		Name:            "Zenoss",
		Context:         "",
		Startup:         "",
		Description:     "Zenoss 5.x",
		Instances:       domain.MinMax{0,0},
		ImageId:         "",
		PoolId:          "",
		DesiredState:    0,
		Launch:          "auto",
		Endpoints:       []servicedefinition.ServiceEndpoint{},
		ParentServiceId: "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LogConfigs: []servicedefinition.LogConfig{
			servicedefinition.LogConfig{
				Path: "/path/to/log/file",
				Type: "test",
			},
		},
	}
	_, err := writeLogstashAgentConfig(&service)
	if err != nil {
		t.Errorf("Writing with empty tags produced an error %s", err)
	}
}
