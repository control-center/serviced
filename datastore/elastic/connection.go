// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/mattbaird/elastigo/core"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"

	"encoding/json"
	"fmt"
)

type elasticConnection struct {
	index string
}

func (ec *elasticConnection) Put(key datastore.Key, msg datastore.JsonMessage) error {
	//func Index(pretty bool, index string, _type string, id string, data interface{}) (api.BaseResponse, error) {

	//stupid API won't let me pass raw json. We should just post/parse using gorilla http
	var data interface{}
	err := json.Unmarshal(msg.Bytes(), &data)
	if err != nil {
		return err
	}
	resp, err := core.Index(false, ec.index, key.Kind(), key.ID(), &data)
	glog.Infof("Put response: %s", resp)
	if err != nil {
		glog.Infof("Put err: %v", err)
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("Non OK response: %v", resp)
	}
	return nil
}

func (ec *elasticConnection) Get(key datastore.Key) (datastore.JsonMessage, error) {
	//	func Get(pretty bool, index string, _type string, id string) (api.BaseResponse, error) {
	glog.Infof("Get for {kind:%s, id:%s}", key.Kind(), key.ID())
	//	err := core.GetSource(ec.index, key.Kind(), key.ID(), &bytes)
	response, err := core.Get(false, ec.index, key.Kind(), key.ID())
	if err != nil {
		glog.Infof("Error is %v", err)
		return nil, err
	}
	if !response.Exists {
		glog.Infof("Entity not found for {kind:%s, id:%s}", key.Kind(), key.ID())
		return nil, datastore.ErrNoSuchEntity{key}
	}
	//again this stupid API doesn't let me deal with raw bytes. GetSource does but doesn't differentiate
	//between bad requests and exists
	bytes, err := json.Marshal(response.Source)
	if err != nil {
		return nil, err
	}
	return datastore.NewJsonMessage(bytes), nil
}

func (ec *elasticConnection) Delete(key datastore.Key) error {
	//func Delete(pretty bool, index string, _type string, id string, version int, routing string) (api.BaseResponse, error) {
	resp, err := core.Delete(false, ec.index, key.Kind(), key.ID(), 0, "")
	glog.Infof("Delete response: %v", resp)
	if err != nil {
		return err
	}
	return nil
}

func (ec *elasticConnection) Query(query datastore.Query) ([]datastore.JsonMessage, error) {
	return make([]datastore.JsonMessage, 0), nil
}
