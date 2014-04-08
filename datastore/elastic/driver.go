// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore/driver"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"github.com/mattbaird/elastigo/api"
)

//ElasticDriver describes an the Elastic Search driver
type ElasticDriver interface {
	SetProperty(name string, prop interface{}) error
	// AddMapping add a document mapping to be registered with ElasticSearch
	AddMapping(name string, mapping interface{}) error
	AddMappingFile(name string, path string) error
	//Initialize the driver, register mappings with elasticserach. Timeout in ms to wait for elastic to be available.
	Initialize(timeout time.Duration) error
	GetConnection() (driver.Connection, error)
}

// New creates a new ElasticDriver
func New(host string, port uint16, index string) ElasticDriver {
	return new(host, port, index)
}

func new(host string, port uint16, index string) *elasticDriver {
	api.Domain = host
	api.Port = fmt.Sprintf("%v", port)
	//TODO: singleton since elastigo doesn't support multiple endpoints

	driver := &elasticDriver{}
	driver.host = host
	driver.port = port
	driver.index = index
	driver.settings = map[string]interface{}{"number_of_shards": 1}
	driver.mappings = make(map[string]interface{})
	driver.mappingPaths = make(map[string]string)
	return driver
}

//Make sure elasticDriver implements datastore.Driver
var _ driver.Driver = &elasticDriver{}

type elasticDriver struct {
	host         string
	port         uint16
	settings     map[string]interface{}
	mappings     map[string]interface{}
	mappingPaths map[string]string
	index        string
}

func (ed *elasticDriver) GetConnection() (driver.Connection, error) {
	return &elasticConnection{ed.index}, nil
}

func (ed *elasticDriver) Initialize(timeout time.Duration) error {

	quit := make(chan int)
	healthy := make(chan int)

	go ed.checkHealth(quit, healthy)

	select {
	case <-healthy:
		glog.Infof("Got response from Elastic")
	case <-time.After(timeout):
		return errors.New("timed Out waiting for response from Elastic")
	}

	if err := ed.postIndex(); err != nil {
		return err
	}

	if err := ed.postMappings(); err != nil {
		return err
	}
	return nil
}

func (ed *elasticDriver) SetProperty(name string, prop interface{}) error {
	ed.settings[name] = prop
	return nil
}

func (ed *elasticDriver) AddMapping(name string, mapping interface{}) error {
	ed.mappings[name] = mapping
	return nil
}

func (ed *elasticDriver) AddMappingFile(name string, path string) error {
	ed.mappingPaths[name] = path
	return nil
}

func (ed *elasticDriver) elasticURL() string {
	return fmt.Sprintf("http://%s:%d", ed.host, ed.port)
}

func (ed *elasticDriver) indexURL() string {
	return fmt.Sprintf("%s/%s/", ed.elasticURL(), ed.index)
}

func (ed *elasticDriver) isUp() bool {
	healthURL := fmt.Sprintf("%v/_cluster/health", ed.elasticURL())
	resp, err := http.Get(healthURL)
	if err == nil && resp.StatusCode == 200 {
		return true
	}
	return false
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
		glog.Infof("Posting mapping to %s", mapURL)
		resp, err := http.Post(mapURL, "application/json", bytes.NewReader(mappingBytes))
		if err != nil {
			return fmt.Errorf("error mapping %s: %s", typeName, err)
		}
		glog.Infof("Response %v", resp)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		glog.Infof("Post result %s", body)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("response %d mapping %s: %s", resp.StatusCode, typeName, string(body))
		}
		return nil
	}

	for typeName, path := range ed.mappingPaths {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		glog.Infof("mappping %v to  %v", path, string(bytes))
		err = post(typeName, bytes)
		if err != nil {
			return err
		}
	}

	for typeName, mapping := range ed.mappings {
		mappingBytes, err := json.Marshal(mapping)
		if err != nil {
			return err
		}

		glog.Infof("mappping %v to  %v", typeName, string(mappingBytes))
		err = post(typeName, mappingBytes)
		if err != nil {
			return err
		}

	}

	return nil
}

func (ed *elasticDriver) postIndex() error {
	url := ed.indexURL()
	glog.Infof("Posting Index to %s", url)

	config := make(map[string]interface{})
	config["settings"] = ed.settings
	configBytes, err := json.Marshal(config)
	glog.Infof("Config is %v", string(configBytes))

	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(configBytes))
	if err != nil {
		return err
	}
	glog.Infof("Response %v", resp)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	glog.Infof("Post result %s", body)
	if err != nil {
		return err
	}

	errResponse := true
	if resp.StatusCode == 400 {
		glog.Info("400 response code")
		//ignore if 400 and IndexAlreadyExistsException
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err == nil {
			if errString, found := result["error"]; found {
				glog.Infof("Found error in response: %v", errString)
				switch errString.(type) {
				case string:
					if strings.HasPrefix(errString.(string), "IndexAlreadyExistsException") {
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
