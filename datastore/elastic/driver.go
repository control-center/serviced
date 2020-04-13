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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/elastigo/api"
)

// Driver describes an the Elastic Search driver
type Driver interface {
	datastore.Driver

	// SetProperty sets a value to a named property.
	SetProperty(name string, prop interface{}) error

	// AddMapping add a document mapping to be registered with ElasticSearch
	AddMapping(mapping Mapping) error

	// Initialize the driver, register mappings with elasticsearch.
	// Timeout in ms to wait for elastic to be available.
	Initialize(timeout time.Duration) error
}

type elasticDriver struct {
	host     string
	port     uint16
	settings map[string]interface{}
	mappings []Mapping
	index    string
}

// Ensure elasticDriver implements datastore.Driver
var _ datastore.Driver = &elasticDriver{}

// Singleton instance because elastigo doesn't support multiple endpoints.
var driver *elasticDriver = nil

// New creates a new elastic.Driver
func New(host string, port uint16, index string, requestTimeout time.Duration) ElasticDriver {
	if driver == nil {
		driver = newDriver(host, port, index, requestTimeout)
	}
	return driver
}

func newDriver(host string, port uint16, index string, requestTimeout time.Duration) *elasticDriver {
	api.Domain = host
	api.Port = fmt.Sprintf("%v", port)
	api.HttpClient = &http.Client{
		Timeout: time.Second * requestTimeout,
	}
	return &elasticDriver{
		host:     host,
		port:     port,
		index:    index,
		settings: map[string]interface{}{"number_of_shards": 1},
		mappings: make([]Mapping, 0),
	}
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
		plog.Debug("Got response from ElasticSearch")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from ElasticSearch")
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
		plog.Debug("Got response from ElasticSearch")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from ElasticSearch")
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
	logger := plog.WithField("mappingfile", path)
	logger.Info("Adding mapping to ElasticSearch")

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		logger.WithError(err).Error("Unable to read mapping file")
		return err
	}

	type mapFile struct {
		Mappings map[string]map[string]interface{}
		Settings map[string]interface{}
	}
	var allMappings mapFile
	err = json.Unmarshal(bytes, &allMappings)
	if err != nil {
		logger.WithError(err).Error("Unable to decode JSON from mapping file")
		return err
	}
	for key, val := range allMappings.Settings {
		ed.settings[key] = val
	}
	for key, mapping := range allMappings.Mappings {

		var rawMapping = make(map[string]map[string]interface{})
		rawMapping[key] = mapping
		value, err := newMapping(rawMapping)
		if err != nil {
			logger.WithError(err).WithField("rawmapping", rawMapping).Error("Unable to create mapping")
			return err
		}
		ed.AddMapping(value)
	}

	logger.Info("Successfully added mapping to ElasticSearch")
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
		plog.WithError(err).Error("Unable to read healthcheck response from ElasticSearch")
		return health, err
	}
	if err := json.Unmarshal(body, &health); err != nil {
		plog.WithError(err).WithField("response", string(body)).Error("Unable to decode JSON healthcheck response from ElasticSearch")
		return health, err
	}
	plog.WithError(err).WithField("response", string(body)).Debug("Received good healthcheck response from ElasticSearch")
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
			plog.Info("Waiting for ElasticSearch")
			time.Sleep(1000 * time.Millisecond)

		case <-quit:
			return
		}
	}

}

func (ed *elasticDriver) postMappings() error {

	post := func(typeName string, mappingBytes []byte) error {
		mapURL := fmt.Sprintf("%s/%s/_mapping", ed.indexURL(), typeName)
		logger := plog.WithField("mapurl", mapURL)
		logger.WithField("mapping", string(mappingBytes)).Debug("Posting mapping")
		resp, err := http.Post(mapURL, "application/json", bytes.NewReader(mappingBytes))
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error mapping %s: %s", typeName, err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		logger.WithField("body", body).Debug("Received response")
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

		err = post(mapping.Name, mappingBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ed *elasticDriver) deleteIndex() error {
	url := ed.indexURL()
	logger := plog.WithField("index", url)
	logger.Info("Deleting Index")

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
	logger.WithField("response", body).Debug("Delete response")
	if err != nil {
		return err
	}
	logger.Info("Index deleted")
	return nil
}

func (ed *elasticDriver) postIndex() error {
	url := ed.indexURL()
	logger := plog.WithField("index", url)

	config := make(map[string]interface{})
	config["settings"] = ed.settings
	configBytes, err := json.Marshal(config)
	logger.WithField("config", string(configBytes)).Debug("Posting Index")

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
	logger.WithField("response", resp).Debug("Received response from ElasticSearch")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	errResponse := true
	if resp.StatusCode == 400 {
		//ignore if 400 and IndexAlreadyExistsException
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err == nil {
			if errString, found := result["error"]; found {
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
		logger.WithField("response", string(body)).Error("Unable to create index")
		return fmt.Errorf("error posting index: %v", resp.Status)
	}
	return nil
}
