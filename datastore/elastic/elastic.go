// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/zenoss/serviced/datastore"

	"encoding/json"
	"fmt"
)

type Json []byte

type PayloadFactory func() interface{}

type ElasticDriver interface {
	SetProperty(name string, prop interface{}) error
	// AddMapping add a document mapping to be registered with ElasticSearch
	AddMapping(name string, mapping interface{}) error
	GetMappings() map[string]interface{}
	RegisterFactory(entityType string, factory PayloadFactory) error
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
	settings  map[string]interface{}
	mappings  map[string]interface{}
	factories map[string]PayloadFactory
}

func (ec *elasticDriver) Put(entity *datastore.Entity) error {
	data, error := ec.serialize(entity)
	fmt.Printf("got json ", string(*data))
	return error
}

func (ec *elasticDriver) Get(id string) (*datastore.Entity, error) {
	return nil, nil
}

func (ec *elasticDriver) Query(query datastore.Query) ([]*datastore.Entity, error) {
	return make([]*datastore.Entity, 0), nil
}

func (ec *elasticDriver) Delete(key datastore.Key) error {
	return nil
}

func (ec *elasticDriver) SetProperty(name string, prop interface{}) error {
	ec.settings[name] = prop
	return nil
}

func (ec *elasticDriver) AddMapping(name string, mapping interface{}) error {
	//TODO: this should add the fields from Entity, key and type)
	ec.mappings[name] = mapping
	return nil
}
func (ec *elasticDriver) GetMappings() map[string]interface{} {
	return ec.mappings
}
func (ec *elasticDriver) RegisterFactory(entityType string, factory PayloadFactory) error {
	ec.factories[entityType] = factory
	return nil
}

func (ec *elasticDriver) serialize(entity *datastore.Entity) (*Json, error) {
	var result Json
	result, err := json.Marshal(entity)
	return &result, err
}

func (ec *elasticDriver) convertPayload(entity *datastore.Entity) error {
	factory, found := ec.factories[entity.Key.Kind()]
	if !found {
		return nil
	}
	value := factory()
	data, error := json.Marshal(entity.Payload)
	if error != nil {
		return error
	}
	json.Unmarshal(data, value)
	if error != nil {
		return error
	}
	entity.Payload = value
	return nil
}
