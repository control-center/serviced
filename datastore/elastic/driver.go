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

package elastic

import (
	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/api"
	"github.com/zenoss/glog"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//ElasticDriver describes an the Elastic Search driver
type ElasticDriver interface {
	SetProperty(name string, prop interface{}) error
	// AddMapping add a document mapping to be registered with ElasticSearch
	AddMapping(mapping Mapping) error
	//Initialize the driver, register mappings with elasticserach. Timeout in ms to wait for elastic to be available.
	Initialize(timeout time.Duration) error
	GetConnection() (datastore.Connection, error)
}

// New creates a new ElasticDriver
func New(host string, port uint16, index string) ElasticDriver {
	return newDriver(host, port, index)
}

func newDriver(host string, port uint16, index string) *elasticDriver {
	api.Domain = host
	api.Port = fmt.Sprintf("%v", port)
	//TODO: singleton since elastigo doesn't support multiple endpoints

	driver := &elasticDriver{}
	driver.host = host
	driver.port = port
	driver.index = index
	driver.settings = map[string]interface{}{"number_of_shards": 1}
	driver.mappings = make([]Mapping, 0)
	return driver
}

//Make sure elasticDriver implements datastore.Driver
var _ datastore.Driver = &elasticDriver{}

type elasticDriver struct {
	host     string
	port     uint16
	settings map[string]interface{}
	mappings []Mapping
	index    string
}

func (ed *elasticDriver) GetConnection() (datastore.Connection, error) {
	return &elasticConnection{ed.index}, nil
}

func (ed *elasticDriver) Initialize(timeout time.Duration) error {

	quit := make(chan int)
	healthy := make(chan int)

	go ed.checkHealth(quit, healthy)

	select {
	case <-healthy:
		glog.V(4).Infof("Got response from Elastic")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from Elastic")
	}

	if err := ed.postIndex(); err != nil {
		return err
	}

	if err := ed.postMappings(); err != nil {
		return err
	}

	// postMapping and postIndex affect es health
	go ed.checkHealth(quit, healthy)

	select {
	case <-healthy:
		glog.V(4).Infof("Got response from Elastic")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from Elastic")
	}

	return nil
}

func (ed *elasticDriver) SetProperty(name string, prop interface{}) error {
	ed.settings[name] = prop
	return nil
}

func (ed *elasticDriver) AddMapping(mapping Mapping) error {
	ed.mappings = append(ed.mappings, mapping)
	return nil
}

func (ed *elasticDriver) AddMappingsFile(path string) error {
	glog.Infof("AddMappingsFiles %v", path)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	glog.V(4).Infof("AddMappingsFiles: content %v", string(bytes))

	type mapFile struct {
		Mappings map[string]map[string]interface{}
		Settings map[string]interface{}
	}
	var allMappings mapFile
	err = json.Unmarshal(bytes, &allMappings)
	if err != nil {
		return err
	}
	for key, val := range allMappings.Settings {
		ed.settings[key] = val
	}
	for key, mapping := range allMappings.Mappings {

		var rawMapping = make(map[string]map[string]interface{})
		rawMapping[key] = mapping
		if value, err := newMapping(rawMapping); err != nil {
			glog.Errorf("%v; could not create mapping from: %v", err, rawMapping)
			return err
		} else {
			ed.AddMapping(value)
		}
	}

	return nil
}

func (ed *elasticDriver) elasticURL() string {
	return fmt.Sprintf("http://%s:%d", ed.host, ed.port)
}

func (ed *elasticDriver) indexURL() string {
	return fmt.Sprintf("%s/%s/", ed.elasticURL(), ed.index)
}

func (ed *elasticDriver) isUp() bool {
	health, err := ed.getHealth()
	if err != nil {
		glog.Errorf("isUp() err=%v", err)
		return false
	}
	status := health["status"]
	return status == "green" || status == "yellow"
}

func (ed *elasticDriver) getHealth() (map[string]interface{}, error) {
	health := make(map[string]interface{})
	healthURL := fmt.Sprintf("%v/_cluster/health", ed.elasticURL())
	resp, err := http.Get(healthURL)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return health, err
	}
	if resp.StatusCode != 200 {
		return health, fmt.Errorf("http status: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("error reading elastic health: %v", err)
		return health, err
	}
	if err := json.Unmarshal(body, &health); err != nil {
		glog.Errorf("error unmarshalling elastic health: %v; err: %v", string(body), err)
		return health, err
	}
	glog.V(4).Infof("Elastic Health: %v; err: %v", string(body), err)
	return health, nil

}

func (ed *elasticDriver) checkHealth(quit chan int, healthy chan int) {
	for {
		select {
		default:
			if ed.isUp() {
				healthy <- 1
				return
			}
			glog.Infof("Waiting for Elastic Search")
			time.Sleep(1000 * time.Millisecond)

		case <-quit:
			return
		}
	}

}

func (ed *elasticDriver) postMappings() error {

	post := func(typeName string, mappingBytes []byte) error {
		mapURL := fmt.Sprintf("%s/%s/_mapping", ed.indexURL(), typeName)
		glog.V(4).Infof("Posting mapping to %s", mapURL)
		resp, err := http.Post(mapURL, "application/json", bytes.NewReader(mappingBytes))
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error mapping %s: %s", typeName, err)
		}
		glog.V(4).Infof("Response %v", resp)
		body, err := ioutil.ReadAll(resp.Body)
		glog.V(4).Infof("Post result %s", body)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("response %d mapping %s: %s", resp.StatusCode, typeName, string(body))
		}
		return nil
	}

	for _, mapping := range ed.mappings {
		mappingBytes, err := json.Marshal(mapping)
		if err != nil {
			return err
		}

		glog.V(4).Infof("mappping %v to  %v", mapping.Name, string(mappingBytes))
		err = post(mapping.Name, mappingBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ed *elasticDriver) deleteIndex() error {
	url := ed.indexURL()
	glog.Infof("Deleting Index %v", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	glog.V(4).Infof("Delete response %s", body)
	if err != nil {
		return err
	}
	return nil
}

func (ed *elasticDriver) postIndex() error {
	url := ed.indexURL()
	glog.V(4).Infof("Posting Index to %v", url)

	config := make(map[string]interface{})
	config["settings"] = ed.settings
	configBytes, err := json.Marshal(config)
	glog.V(4).Infof("Config is %v", string(configBytes))

	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(configBytes))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	glog.V(4).Infof("Response %v", resp)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	errResponse := true
	if resp.StatusCode == 400 {
		glog.V(4).Info("400 response code")
		//ignore if 400 and IndexAlreadyExistsException
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err == nil {
			if errString, found := result["error"]; found {
				glog.V(4).Infof("Found error in response: '%v'", errString)
				switch errString.(type) {
				case string:
					if strings.Contains(errString.(string), "IndexAlreadyExistsException") {
						errResponse = false
					}
				}
			}
		}
	} else if resp.StatusCode >= 200 || resp.StatusCode < 300 {
		errResponse = false
	}
	if errResponse {
		glog.Errorf("Error creating index: %s", string(body))
		return fmt.Errorf("error posting index: %v", resp.Status)
	}
	return nil
}
