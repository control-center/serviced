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
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/elastic/go-elasticsearch/v7"
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
	driver := &elasticDriver{}
	driver.host = host
	driver.port = port
	driver.index = index
	driver.settings = map[string]interface{}{
		"number_of_shards":           1,
		"number_of_replicas":         0,
		"mapping.total_fields.limit": 2000,
	}
	driver.mappings = make([]Mapping, 0)
	return driver
}

//Make sure elasticDriver implements datastore.Driver
var _ datastore.Driver = &elasticDriver{}

type elasticDriver struct {
	host           string
	port           uint16
	settings       map[string]interface{}
	mappings       []Mapping
	index          string
}

func (ed *elasticDriver) GetConnection() (datastore.Connection, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			ed.elasticURL(),
		},
	})
	if err != nil {
		plog.Errorf("Error creating the client: %s", err)
		return nil, err
	}
	return &elasticConnection{ed.index, es}, nil
}

func (ed *elasticDriver) Initialize(timeout time.Duration) error {

	quit := make(chan int)
	healthy := make(chan int)

	go ed.checkHealth(quit, healthy)

	select {
	case <-healthy:
		plog.Debug("Got response from Elastic")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from Elasticsearch")
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
		plog.Debug("Got response from Elasticsearch")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from Elasticsearch")
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
	logger.Info("Adding mapping to Elasticsearch")

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		logger.WithError(err).Error("Unable to read mapping file")
		return err
	}

	type mapFile struct {
		Mappings map[string]interface{}
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

	if value, err := newMapping(allMappings.Mappings); err != nil {
		logger.WithError(err).WithField("rawmapping", allMappings.Mappings).Error("Unable to create mapping")
		return err
	} else {
		ed.AddMapping(value)
	}

	logger.Info("Successfully added mapping to Elasticsearch")
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
		plog.WithError(err).Error("Unable to read healthcheck response from Elasticsearch")
		return health, err
	}
	if err := json.Unmarshal(body, &health); err != nil {
		plog.WithError(err).WithField("response", string(body)).Error("Unable to decode JSON healthcheck response from Elasticsearch")
		return health, err
	}
	plog.WithError(err).WithField("response", string(body)).Debug("Received good healthcheck response from Elasticsearch")
	return health, nil
}

func SetDiscSpaceThresholds(elasticURL string) error {
	client := &http.Client{}
	clusterSettingsURL := fmt.Sprintf("%v/_cluster/settings", elasticURL)
	clusterSettings := map[string]interface{}{
		"transient": map[string]string{
			"cluster.routing.allocation.disk.watermark.low":         "3gb",
			"cluster.routing.allocation.disk.watermark.high":        "2gb",
			"cluster.routing.allocation.disk.watermark.flood_stage": "1gb",
		},
	}

	clusterSettingsJson, _ := json.Marshal(clusterSettings)

	req, _ := http.NewRequest(http.MethodPut, clusterSettingsURL, bytes.NewBuffer(clusterSettingsJson))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		return fmt.Errorf("http status: %v %s", resp.StatusCode, err)
	}

	return nil
}

func TurnOffIndexReadOnlyMode(index string, elasticURL string) error {
	client := &http.Client{}
	indexSettingsURL := fmt.Sprintf("%v/%s/_settings", elasticURL, index)
	indexSettings := map[string]interface{}{
		"index": map[string]interface{}{
			"blocks": map[string]string{
				"read_only_allow_delete": "false",
			},
		},
	}

	indexSettingsJson, _ := json.Marshal(indexSettings)

	req, _ := http.NewRequest(http.MethodPut, indexSettingsURL, bytes.NewBuffer(indexSettingsJson))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		return fmt.Errorf("http status: %v %s", resp.StatusCode, err)
	}

	return nil
}

func (ed *elasticDriver) checkHealth(quit chan int, healthy chan int) {
	for {
		select {
		default:
			if ed.isUp() {
				healthy <- 1
				return
			}
			plog.Info("Waiting for Elasticsearch")
			time.Sleep(1000 * time.Millisecond)

		case <-quit:
			return
		}
	}

}

func (ed *elasticDriver) postMappings() error {
	//mapping types are deprecated since 6.x release and it's completely removed from 8.x version
	mappingUrl := fmt.Sprintf("%s%s", ed.indexURL(), "_mapping")

	post := func(mappingBytes []byte) error {
		client := &http.Client{}
		logger := plog.WithField("mapurl", mappingUrl)
		logger.WithField("mapping", string(mappingBytes)).Debug("Posting mapping")

		req, _ := http.NewRequest(http.MethodPut, mappingUrl, bytes.NewBuffer(mappingBytes))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := client.Do(req)

		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error mapping %s", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		logger.WithField("body", body).Debug("Received response")
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("response %d mapping %s", resp.StatusCode, string(body))
		}
		return nil
	}

	ed.AddMapping(Mapping{
		Entries: map[string]interface{}{
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type": "keyword",
				},
			},
		},
	})
	plog.Debugf("Mapping dict %s", ed.mappings)
	for i := range ed.mappings {
		mappingBytes, err := json.Marshal(ed.mappings[i].Entries)

		if err != nil {
			return err
		}

		err = post(mappingBytes)
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
	client := &http.Client{}
	config := map[string]interface{}{
		"settings": map[string]interface{}{
			"index": ed.settings,
		},
	}
	configBytes, err := json.Marshal(config)
	logger.WithField("config", string(configBytes)).Debug("Posting Index")

	if err != nil {
		return err
	}

	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(configBytes))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	logger.WithField("response", resp).Debug("Received response from Elasticsearch")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	errResponse := true
	if resp.StatusCode == 400 {
		//ignore if 400 and resource_already_exists_exception
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err == nil {
			if errType, found := result["error"].(map[string]interface{})["type"]; found {
				switch errType.(type) {
				case string:
					if errType.(string) == "resource_already_exists_exception" {
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
