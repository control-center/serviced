// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/zenoss/elastigo/api"
	"github.com/zenoss/elastigo/core"
	"github.com/zenoss/elastigo/indices"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"

	"encoding/json"
	"fmt"
	"reflect"
)

type elasticConnection struct {
	index string
}

func (ec *elasticConnection) Put(key datastore.Key, msg datastore.JSONMessage) error {
	//func Index(pretty bool, index string, _type string, id string, data interface{}) (api.BaseResponse, error) {

	glog.V(4).Infof("Put for {kind:%s, id:%s} %v", key.Kind(), key.ID(), string(msg.Bytes()))
	var raw json.RawMessage
	raw = msg.Bytes()
	resp, err := core.Index(false, ec.index, key.Kind(), key.ID(), &raw)
	indices.Refresh(ec.index)
	glog.V(4).Infof("Put response: %v", resp)
	if err != nil {
		glog.Errorf("Put err: %v", err)
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("non OK response: %v", resp)
	}
	return nil
}

func (ec *elasticConnection) Get(key datastore.Key) (datastore.JSONMessage, error) {
	//	func Get(pretty bool, index string, _type string, id string) (api.BaseResponse, error) {
	glog.V(4).Infof("Get for {kind:%v, id:%v}", key.Kind(), key.ID())
	//	err := core.GetSource(ec.index, key.Kind(), key.ID(), &bytes)
	response, err := elasticGet(false, ec.index, key.Kind(), key.ID())
	if err != nil {
		glog.Errorf("Error is %v", err)
		return nil, err
	}
	if !response.Exists {
		glog.V(4).Infof("Entity not found for {kind:%s, id:%s}", key.Kind(), key.ID())
		return nil, datastore.ErrNoSuchEntity{Key: key}
	}
	bytes := response.Source
	return datastore.NewJSONMessage(bytes), nil
}

func (ec *elasticConnection) Delete(key datastore.Key) error {
	//func Delete(pretty bool, index string, _type string, id string, version int, routing string) (api.BaseResponse, error) {
	resp, err := core.Delete(false, ec.index, key.Kind(), key.ID(), 0, "")
	indices.Refresh(ec.index)
	glog.V(4).Infof("Delete response: %v", resp)
	if err != nil {
		return err
	}
	return nil
}

func (ec *elasticConnection) Query(query interface{}) ([]datastore.JSONMessage, error) {

	search, ok := query.(*search.SearchDsl)
	if !ok {
		return nil, fmt.Errorf("invalid search type %v", reflect.ValueOf(query))
	}
	resp, err := search.Result()
	if err != nil {
		err = fmt.Errorf("error executing query %v", err)
		glog.Errorf("%v", err)
		return nil, err
	}
	return toJSONMessages(resp), nil
}

// convert search result of json host to dao.Host array
func toJSONMessages(result *core.SearchResult) []datastore.JSONMessage {
	glog.V(4).Infof("Converting results %v", result)
	var total = len(result.Hits.Hits)
	var msgs = make([]datastore.JSONMessage, total)
	for i := 0; i < total; i++ {
		glog.V(4).Infof("Adding result %s", string(result.Hits.Hits[i].Source))
		msg := datastore.NewJSONMessage(result.Hits.Hits[i].Source)
		msgs[i] = msg
	}
	return msgs
}

//Modified from elastigo to use custom response type
func elasticGet(pretty bool, index string, _type string, id string) (elasticResponse, error) {
	var url string
	var retval elasticResponse
	if len(_type) > 0 {
		url = fmt.Sprintf("/%s/%s/%s?%s", index, _type, id, api.Pretty(pretty))
	} else {
		url = fmt.Sprintf("/%s/%s?%s", index, id, api.Pretty(pretty))
	}
	body, err := api.DoCommand("GET", url, nil)
	if err != nil {
		return retval, err
	}
	if err == nil {
		// marshall into json
		jsonErr := json.Unmarshal(body, &retval)
		if jsonErr != nil {
			return retval, jsonErr
		}
	}
	return retval, err
}

//Modified from elastigo BaseResponse to accept document source into a json.RawMessage
type elasticResponse struct {
	Ok      bool            `json:"ok"`
	Index   string          `json:"_index,omitempty"`
	Type    string          `json:"_type,omitempty"`
	ID      string          `json:"_id,omitempty"`
	Source  json.RawMessage `json:"_source,omitempty"` // depends on the schema you've defined
	Version int             `json:"_version,omitempty"`
	Found   bool            `json:"found,omitempty"`
	Exists  bool            `json:"exists,omitempty"`
}
