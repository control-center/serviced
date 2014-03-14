// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/zenoss/serviced/datastore"
)

type ElasticDriver interface {
	SetProperty(name string, prop interface{}) error
	// AddMapping add a document mapping to be registered with ElasticSearch
	AddMapping(name string, mapping interface{}) error
	GetMappings() map[string]interface{}
}

func New(host string, port uint16) ElasticDriver {
	//TODO: set elastic host and port
	//TODO: singleton since elastigo doesn't support multiple endpoints
	driver := &elasticDriver{}
	driver.settings = map[string]interface{}{"number_of_shards": 1}
	driver.mappings = make(map[string]interface{})
	return driver
}

//Make sure elasticDriver implements datastore.Driver
var _ datastore.Driver = &elasticDriver{}

type elasticDriver struct {
	settings map[string]interface{}
	mappings map[string]interface{}
}

func (ed *elasticDriver) GetConnection() datastore.Connection {
	return &elasticConnection{}
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
