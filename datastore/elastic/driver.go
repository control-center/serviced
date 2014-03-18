// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type ElasticDriver interface {
	SetProperty(name string, prop interface{}) error
	// AddMapping add a document mapping to be registered with ElasticSearch
	AddMapping(name string, mapping interface{}) error
	Initialize() error
	GetConnection() (datastore.Connection, error)
}

func New(host string, port uint16, index string) ElasticDriver {
	//TODO: set elastic host and port
	//TODO: singleton since elastigo doesn't support multiple endpoints

	driver := &elasticDriver{}
	driver.host = host
	driver.port = port
	driver.index = index
	driver.settings = map[string]interface{}{"number_of_shards": 1}
	driver.mappings = make(map[string]interface{})
	return driver
}

//Make sure elasticDriver implements datastore.Driver
var _ datastore.Driver = &elasticDriver{}

type elasticDriver struct {
	host     string
	port     uint16
	settings map[string]interface{}
	mappings map[string]interface{}
	index    string
}

func (ed *elasticDriver) GetConnection() (datastore.Connection, error) {
	return &elasticConnection{ed.index}, nil
}

func (ed *elasticDriver) Initialize() error {
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
	//TODO: this should add the fields from Entity, key and type)
	ed.mappings[name] = mapping
	return nil
}

func (ed *elasticDriver) postMappings() error {
	baseUrl := fmt.Sprintf("http://%s:%d/%s/", ed.host, ed.port, ed.index)

	for typeName, mapping := range ed.mappings {

		mappingBytes, err := json.Marshal(mapping)

		glog.Infof("mappping %v to  %v", typeName, string(mappingBytes))

		if err != nil {
			return err
		}
		mapURL := fmt.Sprintf("%s/%s/_mapping", baseUrl, typeName)
		glog.Infof("Posting mapping to %s", mapURL)
		resp, err := http.Post(mapURL, "application/json", bytes.NewReader(mappingBytes))
		if err != nil {
			return fmt.Errorf("Error mapping %s: %s", typeName, err)
		}
		glog.Infof("Response %v", resp)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		glog.Infof("Post result %s", body)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("Response &d mapping %s: $s", resp.StatusCode, typeName, string(body))
		}
	}
	return nil
}

func (ed *elasticDriver) postIndex() error {
	url := fmt.Sprintf("http://%s:%d/%s/", ed.host, ed.port, ed.index)
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
		return fmt.Errorf("Error posting index: %v", resp.Status)
	}
	return nil
}
