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
	"context"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/logging"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/zenoss/logri"
	"io/ioutil"
)

type elasticConnection struct {
	index  string
	client *elasticsearch.Client
}

var (
	plog        = logging.PackageLogger() // the standard package logger
	traceLogger *logri.Logger             // a 'trace' logger for an additional level of debug messages
)

func init() {
	traceLoggerName := plog.Name + ".trace"
	traceLogger = logri.GetLogger(traceLoggerName)
}

func BuildID(id string, key string) string {
	return fmt.Sprintf("%s-%s", id, key)
}

func (ec *elasticConnection) Put(key datastore.Key, msg datastore.JSONMessage) error {
	logger := plog.WithFields(log.Fields{
		"kind": key.Kind(),
		"id":   key.ID(),
	})
	logger.Debug("Put")
	traceLogger.WithField("payload", string(msg.Bytes())).Debug("Put")

	//Because of document type depreciation in ES 7.x we combine id and key
	args := []func(*esapi.IndexRequest){
		ec.client.Index.WithDocumentID(BuildID(key.ID(), key.Kind())),
		ec.client.Index.WithRefresh("true"),
	}

	if (msg.Version()["seqNo"] + msg.Version()["primaryTerm"]) > 0 {
		args = append(args, ec.client.Index.WithIfPrimaryTerm(msg.Version()["primaryTerm"]),
			ec.client.Index.WithIfSeqNo(msg.Version()["seqNo"]))
	}

	res, err := ec.client.Index(ec.index, bytes.NewReader(msg.Bytes()), args...)

	defer res.Body.Close()
	if err != nil {
		logger.WithError(err).Error("Put failed")
		//TODO handle new type of ES error
		//if eserr, iseserror := err.(api.ESError); iseserror && eserr.Code == 409 {
		// Conflict
		//return fmt.Errorf("Your changes conflict with those made by another user. Please reload and try your changes again.")
		//}
		return err
	}

	traceLogger.WithField("response", res).Debug("Put")
	if res.IsError() {
		logger.Info("error while put")
		return fmt.Errorf("non OK response: %v", res)
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

	//Because of document type depreciation in ES 7.x we combine id and key
	response, err := ec.elasticGet(BuildID(key.ID(), key.Kind()))
	if err != nil {
		logger.WithError(err).Error("Get failed")
		return nil, err
	}
	if !response.Found {
		logger.Debug("Entity not found")
		return nil, datastore.ErrNoSuchEntity{Key: key}
	}
	bytes := response.Source
	msg := datastore.NewJSONMessage(bytes, map[string]int{
		"version":     response.Version,
		"primaryTerm": response.PrimaryTerm,
		"seqNo":       response.SeqNo})
	return msg, nil
}

func (ec *elasticConnection) Delete(key datastore.Key) error {
	logger := plog.WithFields(log.Fields{
		"kind": key.Kind(),
		"id":   key.ID(),
	})
	logger.Debug("Delete")

	//Because of document type depreciation in ES 7.x we combine id and key
	res, err := ec.client.Delete(ec.index,
		BuildID(key.ID(), key.Kind()),
		ec.client.Delete.WithRefresh("true"))

	traceLogger.WithField("response", res).Debug("Delete")
	if err != nil {
		return err
	}
	return nil
}

func (ec *elasticConnection) Query(query esapi.SearchRequest) ([]datastore.JSONMessage, error) {

	resp, err := query.Do(context.Background(), ec.client)
	if err != nil {
		err = fmt.Errorf("error executing query %v", err)
		plog.WithError(err).Error("error executing query")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			plog.WithError(err).Errorf("Error parsing the response body: %s", err)
		} else {
			plog.WithError(err).Errorf("[%s] %s: %s",
				resp.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
		return nil, err
	}

	return toJSONMessages(resp)
}

// convert search result of json host to dao.Host array
func toJSONMessages(result *esapi.Response) ([]datastore.JSONMessage, error) {
	// Deserialize the response into a map.
	var r elasticSearchResults
	if err := json.NewDecoder(result.Body).Decode(&r); err != nil {
		plog.WithError(err).Errorf("Error parsing the response body: %s", err)
		return nil, err
	}

	var total = r.Hits.Total.Value
	var msgs = make([]datastore.JSONMessage, total)

	logger := plog.WithField("total", total)
	if total > 0 {
		// Note that while it's possible for queries to span types, the vast majority of CC use cases (all?)
		//   only query a single type at a time, so it's sufficient to get the type of the first hit
		logger.WithField("id-type", r.Hits.Hits[0].Id)
	}
	logger.Debug("Query finished")

	for i, hit := range r.Hits.Hits {
		var data bytes.Buffer
		if err := json.NewEncoder(&data).Encode(hit.Source); err != nil {
			plog.WithError(err).Errorf("Error encoding query: %s", err)
			return nil, err
		}

		msg := datastore.NewJSONMessage(data.Bytes(), map[string]int{
			"version":     hit.Version,
			"primaryTerm": hit.PrimaryTerm,
			"seqNo":       hit.SeqNo,
		})
		msgs[i] = msg
	}
	return msgs, nil
}

//Modified from elastigo to use custom response type
func (ec *elasticConnection) elasticGet(id string) (elasticResponse, error) {
	var (
		retval elasticResponse
		res    *esapi.Response
		err    error
	)

	res, err = ec.client.Get(ec.index, id)

	defer res.Body.Close()

	if err != nil {
		plog.Errorf("Error getting response: %s", err)
		return retval, err
	}

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			plog.Errorf("Error parsing the response body: %s", err)
			return retval, err
		} else {
			// Print the response status and error information.
			plog.Errorf("%s ID:%s", res.Status(), id)
			return retval, nil
		}
	}
	//body, err := api.DoCommand("GET", url, nil)

	// marshall into json
	var body []byte
	body, err = ioutil.ReadAll(res.Body)

	if err != nil {
		return retval, err
	}

	jsonErr := json.Unmarshal(body, &retval)
	if jsonErr != nil {
		return retval, jsonErr
	}
	return retval, err
}

//Modified from elastigo BaseResponse to accept document source into a json.RawMessage
type elasticResponse struct {
	Ok          bool            `json:"ok"`
	Index       string          `json:"_index,omitempty"`
	Type        string          `json:"_type,omitempty"`
	ID          string          `json:"_id,omitempty"`
	Source      json.RawMessage `json:"_source,omitempty"` // depends on the schema you've defined
	Version     int             `json:"_version,omitempty"`
	Found       bool            `json:"found,omitempty"`
	Exists      bool            `json:"exists,omitempty"`
	PrimaryTerm int             `json:"_primary_term,omitempty"`
	SeqNo       int             `json:"_seq_no,omitempty"`
}

type elasticSearchResults struct {
	Took     int            `json:"took,omitempty"`
	TimedOut bool           `json:"timed_out,omitempty"`
	Shards   map[string]int `json:"_shards,omitempty"`
	Hits     struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index       string                 `json:"_index"`
			Type        string                 `json:"_type"`
			Id          string                 `json:"_id"`
			Version     int                    `json:"_version,omitempty"`
			PrimaryTerm int                    `json:"_primary_term,omitempty"`
			SeqNo       int                    `json:"_seq_no,omitempty"`
			Score       float64                `json:"_score"`
			Source      map[string]interface{} `json:"_source"`
		} `json:"hits,omitempty"`
	} `json:"hits,omitempty"`
}
