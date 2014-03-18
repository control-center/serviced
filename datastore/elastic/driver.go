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
	GetMappings() map[string]interface{}
	Initialize() error
	GetConnection() datastore.Connection
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

func (ed *elasticDriver) GetConnection() datastore.Connection {
	return &elasticConnection{ed.index}
}

func (ed *elasticDriver) Initialize() error {
	url := fmt.Sprintf("http://%s:%d/%s", ed.host, ed.port, ed.index)
	glog.Infof("Posting to %s", url)
	config := make(map[string]interface{})
	config["settings"] = ed.settings
	config["mappings"] = ed.mappings
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

func (ed *elasticDriver) SetProperty(name string, prop interface{}) error {
	ed.settings[name] = prop
	return nil
}

func (ed *elasticDriver) AddMapping(name string, mapping interface{}) error {
	//TODO: this should add the fields from Entity, key and type)
	ed.mappings[name] = mapping
	return nil
}
func (ed *elasticDriver) GetMappings() map[string]interface{} {
	return ed.mappings
}

//func (ec *elasticDriver) serialize(entity *datastore.Entity) (*Json, error) {
//	var result Json
//	result, err := json.Marshal(entity)
//	return &result, err
//}
//
//func (ed *elasticDriver) convertPayload(entity *datastore.Entity) error {
//	factory, found := ed.factories[entity.Key.Kind()]
//	if !found {
//		return nil
//	}
//	value := factory()
//	data, error := json.Marshal(entity.Payload)
//	if error != nil {
//		return error
//	}
//	json.Unmarshal(data, value)
//	if error != nil {
//		return error
//	}
//	entity.Payload = value
//	return nil
//}
//
