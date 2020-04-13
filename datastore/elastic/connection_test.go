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

// +build integration

package elastic_test

import (
	"fmt"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"encoding/json"
	"reflect"
	"testing"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{elastic.ElasticTest{Index: "twitter"}})

type S struct {
	elastic.ElasticTest
}

//func TestPutGetDelete(t *testing.T) {
func (s *S) TestPutGetDelete(t *C) {
	esdriver := s.Driver()
	//	driver, err := getConnection()
	//	if err != nil {
	//		t.Fatalf("Error initializing driver: %v", err)
	//	}

	//Turn off read only mode
	elasticUrl := fmt.Sprintf("http://localhost:%d", s.Port)
	elastic.SetDiscSpaceThresholds(elasticUrl)
	elastic.TurnOffIndexReadOnlyMode("twitter", elasticUrl)

	conn, err := esdriver.GetConnection()
	if err != nil {
		t.Fatalf("Error getting connection: %v", err)
	}

	k := datastore.NewKey("tweet", "1")
	tweet := map[string]string{
		"user":      "kimchy",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out Elasticsearch",
		"type":      "tweet",
	}
	tweetJSON, err := json.Marshal(tweet)
	versions := map[string]int{
		"version":     0,
		"primaryTerm": 0,
		"seqNo":       0,
	}
	err = conn.Put(k, datastore.NewJSONMessage(tweetJSON, versions))
	if err != nil {
		t.Errorf("%v", err)
	}

	//Get tweet
	raw, err := conn.Get(k)
	if err != nil {
		t.Fatalf("Unexpected: %v", err)
	}
	t.Logf("raw is %v", string(raw.Bytes()))
	var tweetMap map[string]string
	json.Unmarshal(raw.Bytes(), &tweetMap)
	t.Logf("tweet is %v", tweetMap)

	if tweetMap["user"] != "kimchy" {
		t.Errorf("Expected kimchy, found %s", tweetMap["user"])
	}

	//Delete tweet
	err = conn.Delete(k)
	if err != nil {
		t.Errorf("Unexpected delete error: %v", err)
	}

	//test not found
	raw, err = conn.Get(k)
	if raw != nil {
		t.Errorf("Expected nil return;")
	}
	if err == nil {
		t.Error("Expected error, not nil")
	} else if !datastore.IsErrNoSuchEntity(err) {
		t.Logf("type is %s", reflect.ValueOf(err))
		t.Fatalf("Unexpected: %v", err)
	}

}

func (s *S) TestQuery(t *C) {
	esdriver := s.Driver()
	conn, err := esdriver.GetConnection()
	if err != nil {
		t.Fatalf("Error getting connection: %v", err)
	}

	k := datastore.NewKey("tweet", "1")
	tweet := map[string]string{
		"user":      "kimchy",
		"state":     "NY",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out Elasticsearch",
		"type":      "tweet",
	}
	tweetJSON, err := json.Marshal(tweet)
	versions := map[string]int{
		"version":     0,
		"primaryTerm": 0,
		"seqNo":       0,
	}
	err = conn.Put(k, datastore.NewJSONMessage(tweetJSON, versions))
	if err != nil {
		t.Errorf("%v", err)
	}

	k = datastore.NewKey("tweet", "2")
	tweet = map[string]string{
		"user":      "kimchy2",
		"state":     "NY",
		"post_date": "2010-11-15T14:12:12",
		"message":   "trying out Elasticsearch again",
		"type":      "tweet",
	}
	tweetJSON, err = json.Marshal(tweet)
	err = conn.Put(k, datastore.NewJSONMessage(tweetJSON, versions))
	if err != nil {
		t.Errorf("%v", err)
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"exists": map[string]string{"field": "state"}},
					{"term": map[string]string{"type": "tweet"}},
				},
			},
		},
	}

	testSearch, err := elastic.BuildSearchRequest(query, "twitter")

	if err != nil {
		t.Errorf("%s", err)
	}

	msgs, err := conn.Query(testSearch)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 msgs, got  %v", len(msgs))
	}

	//query for non-existant entity
	query = map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"exists": map[string]string{"field": "blam"}},
					{"term": map[string]string{"type": "tweet"}},
				},
			},
		},
	}

	testSearch, err = elastic.BuildSearchRequest(query, "twitter")

	if err != nil {
		t.Errorf("%s", err)
	}

	msgs, err = conn.Query(testSearch)
	if err != nil {
		t.Errorf("Unepected error %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 msgs, got %d", len(msgs))
	}

}
