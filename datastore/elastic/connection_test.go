// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"

	"encoding/json"
	"reflect"
	"testing"
	"time"
)

var driver ElasticDriver

func getConnection() (ElasticDriver, error) {
	return InitElasticTest("twitter")
}

func TestPutGetDelete(t *testing.T) {

	driver, err := getConnection()
	if err != nil {
		t.Fatalf("Error initializing driver: %v", err)
	}

	conn, err := driver.GetConnection()
	if err != nil {
		t.Fatalf("Error getting connection: %v", err)
	}

	key := datastore.NewKey("tweet", "1")
	tweet := map[string]string{
		"user":      "kimchy",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out Elasticsearch",
	}
	tweetJson, err := json.Marshal(tweet)
	err = conn.Put(key, datastore.NewJsonMessage(tweetJson))
	if err != nil {
		t.Errorf("%v", err)
	}

	//Get tweet
	raw, err := conn.Get(key)
	if err != nil {
		t.Fatalf("Unexpected: %v", err)
	}
	glog.Infof("raw is %v", string(raw.Bytes()))
	var tweetMap map[string]string
	json.Unmarshal(raw.Bytes(), &tweetMap)
	glog.Infof("tweet is %v", tweetMap)

	if tweetMap["user"] != "kimchy" {
		t.Errorf("Expected kimchy, found %s", tweetMap["user"])
	}

	//Delete tweet
	err = conn.Delete(key)
	if err != nil {
		t.Errorf("Unexpected delete error: %v", err)
	}

	//test not found
	raw, err = conn.Get(key)
	if raw != nil {
		t.Errorf("Expected nil return;")
	}
	if err == nil {
		t.Error("Expected error, not nil")
	} else if !datastore.IsErrNoSuchEntity(err) {
		glog.Infof("type is %s", reflect.ValueOf(err))
		t.Fatalf("Unexpected: %v", err)
	}

}

func TestQuery(t *testing.T) {

	driver, err := getConnection()
	if err != nil {
		t.Fatalf("Error initializing driver: %v", err)
	}

	conn, err := driver.GetConnection()
	if err != nil {
		t.Fatalf("Error getting connection: %v", err)
	}

	key := datastore.NewKey("tweet", "1")
	tweet := map[string]string{
		"user":      "kimchy",
		"state":     "NY",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out Elasticsearch",
	}
	tweetJson, err := json.Marshal(tweet)
	err = conn.Put(key, datastore.NewJsonMessage(tweetJson))
	if err != nil {
		t.Errorf("%v", err)
	}

	key = datastore.NewKey("tweet", "2")
	tweet = map[string]string{
		"user":      "kimchy2",
		"state":     "NY",
		"post_date": "2010-11-15T14:12:12",
		"message":   "trying out Elasticsearch again",
	}
	tweetJson, err = json.Marshal(tweet)
	err = conn.Put(key, datastore.NewJsonMessage(tweetJson))
	if err != nil {
		t.Errorf("%v", err)
	}

	query := search.Query().Search("_exists_:state")
	testSearch := search.Search("twitter").Type("tweet").Size("10000").Query(query)

	time.Sleep(1000 * time.Millisecond)
	msgs, err := conn.Query(testSearch)
	if err != nil {
		t.Errorf("Unepected error %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 msgs, got  %v", len(msgs))
	}

	//query for non-existant entity
	query = search.Query().Search("_exists_:blam")
	testSearch = search.Search("twitter").Type("tweet").Size("10000").Query(query)

	msgs, err = conn.Query(testSearch)
	if err != nil {
		t.Errorf("Unepected error %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 msgs, got  %V", len(msgs))
	}

}
