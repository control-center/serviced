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
	"encoding/json"
	"fmt"
	"reflect"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/logging"
	"github.com/zenoss/elastigo/api"
	"github.com/zenoss/elastigo/core"
	"github.com/zenoss/elastigo/indices"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/logri"
)

type elasticConnection struct {
	index string
}

var (
	plog        = logging.PackageLogger() // the standard package logger
	traceLogger *logri.Logger             // a 'trace' logger for an additional level of debug messages
)

func init() {
	traceLoggerName := plog.Name + ".trace"
	traceLogger = logri.GetLogger(traceLoggerName)
}

func (ec *elasticConnection) Put(key datastore.Key, msg datastore.JSONMessage) error {
	logger := plog.WithFields(log.Fields{
		"kind": key.Kind(),
		"id":   key.ID(),
	})
	logger.Debug("Put")
	traceLogger.WithField("payload", string(msg.Bytes())).Debug("Put")

	var raw json.RawMessage
	raw = msg.Bytes()
	resp, err := core.IndexWithParameters(
		false, ec.index, key.Kind(), key.ID(), "", msg.Version(), "", "", "", 0, "", "", false, &raw,
	)
	if err != nil {
		logger.WithError(err).Error("Put failed")
		if eserr, iseserror := err.(api.ESError); iseserror && eserr.Code == 409 {
			// Conflict
			return fmt.Errorf("your changes conflict with those made by another user. Please reload and try your changes again")
		}
		return err
	}
	indices.Refresh(ec.index)
	traceLogger.WithField("response", resp).Debug("Put")
	if !resp.Ok {
		return fmt.Errorf("non OK response: %v", resp)
	}
	return nil
}

func (ec *elasticConnection) Get(key datastore.Key) (datastore.JSONMessage, error) {
	logger := plog.WithFields(log.Fields{
		"kind": key.Kind(),
		"id":   key.ID(),
	})
	logger.Debug("Get")
	traceLogger.Debug("Get")

	//	func Get(pretty bool, index string, _type string, id string) (api.BaseResponse, error) {
	//	err := core.GetSource(ec.index, key.Kind(), key.ID(), &bytes)
	response, err := elasticGet(false, ec.index, key.Kind(), key.ID())
	if err != nil {
		logger.WithError(err).Error("Get failed")
		return nil, err
	}
	if !response.Exists {
		logger.Debug("Entity not found")
		return nil, datastore.ErrNoSuchEntity{Key: key}
	}
	bytes := response.Source
	msg := datastore.NewJSONMessage(bytes, response.Version)
	return msg, nil
}

func (ec *elasticConnection) Delete(key datastore.Key) error {
	logger := plog.WithFields(log.Fields{
		"kind": key.Kind(),
		"id":   key.ID(),
	})
	logger.Debug("Delete")

	//func Delete(pretty bool, index string, _type string, id string, version int, routing string) (api.BaseResponse, error) {
	resp, err := core.Delete(false, ec.index, key.Kind(), key.ID(), 0, "")
	indices.Refresh(ec.index)
	traceLogger.WithField("response", resp).Debug("Delete")
	if err != nil {
		return err
	}
	return nil
}

func (ec *elasticConnection) Query(query interface{}) ([]datastore.JSONMessage, error) {
	switch s := query.(type) {
	case *search.SearchDsl:
		resp, err := s.Result()
		if err != nil {
			err = fmt.Errorf("error executing query %v", err)
			plog.WithError(err).Error("error executing query")
			return nil, err
		}
		return toJSONMessages(resp), nil
	case SearchRequest:
		resp, err := core.SearchRequest(
			s.Pretty,
			s.Index,
			s.Type,
			s.Query,
			s.Scroll,
			s.Scan,
		)
		if err != nil {
			err = fmt.Errorf("error executing query %v", err)
			return nil, err
		}
		return toJSONMessages(&resp), nil
	default:
		return nil, fmt.Errorf("invalid search type %v", reflect.ValueOf(query))
	}
}

// convert search result of json host to dao.Host array
func toJSONMessages(result *core.SearchResult) []datastore.JSONMessage {
	var total = len(result.Hits.Hits)
	var msgs = make([]datastore.JSONMessage, total)

	logger := plog.WithField("total", total)
	if total > 0 {
		// Note that while it's possible for queries to span types, the vast majority of CC use cases (all?)
		//   only query a single type at a time, so it's sufficient to get the type of the first hit
		logger.WithField("type", result.Hits.Hits[0].Type)
	}
	logger.Debug("Query finished")

	for i := 0; i < total; i++ {
		src := result.Hits.Hits[i].Source
		fields := result.Hits.Hits[i].Fields
		var data []byte
		if len(fields) == 0 {
			data = src
		} else {
			data = fields
		}
		version := result.Hits.Hits[i].Version
		msg := datastore.NewJSONMessage(data, version)
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

// SearchRequest used to directly search elastic
type SearchRequest struct {
	Pretty bool
	Index  string
	Type   string
	Query  interface{}
	Scroll string
	Scan   int
}
