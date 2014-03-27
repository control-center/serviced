// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package integration_test

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"

	"reflect"
	"testing"
)

var driver datastore.Driver

func getContext() (datastore.Context, error) {
	if driver == nil {
		esDriver, err := elastic.InitElasticTest("twitter")
		if err != nil {
			return nil, err
		}
		driver = esDriver
	}
	return datastore.NewContext(driver), nil
}

func TestPutGetDelete(t *testing.T) {

	ctx, err := getContext()
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if ctx == nil {
		t.Fatal("Expected context")
	}
	ds := datastore.New()

	key := datastore.NewKey("tweet", "1")
	tweet := map[string]string{
		"user":      "kimchy",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out Elasticsearch",
	}
	err = ds.Put(ctx, key, tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	//Get tweet
	var tweetMap map[string]string
	err = ds.Get(ctx, key, &tweetMap)
	if err != nil {
		t.Fatalf("Unexpected: %v", err)
	}
	glog.Infof("tweet is %v", tweetMap)

	if tweetMap["user"] != "kimchy" {
		t.Errorf("Expected kimchy, found %s", tweetMap["user"])
	}

	//Delete tweet
	err = ds.Delete(ctx, key)
	if err != nil {
		t.Errorf("Unexpected delete error: %v", err)
	}

	//test not found
	err = ds.Get(ctx, key, &tweetMap)
	if err == nil {
		t.Error("Expected error, not nil")
	} else if !datastore.IsErrNoSuchEntity(err) {
		glog.Infof("type is %s", reflect.ValueOf(err))
		t.Fatalf("Unexpected: %v", err)
	}
}

func TestQuery(t *testing.T) {

	ctx, err := getContext()
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if ctx == nil {
		t.Fatal("Expected context")
	}
	ds := datastore.New()

	key := datastore.NewKey("tweet", "1")
	tweet := map[string]string{
		"user":      "kimchy",
		"state":     "NY",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out Elasticsearch",
	}

	err = ds.Put(ctx, key, tweet)
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
	err = ds.Put(ctx, key, tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	query := search.Query().Search("_exists_:state")
	testSearch := search.Search("twitter").Type("tweet").Size("10000").Query(query)

	q := datastore.NewQuery(ctx)
	msgs, err := q.Execute(testSearch)

	if err != nil {
		t.Errorf("Unepected error %v", err)
	}
	if msgs.Len() != 2 {
		t.Errorf("Expected 2 msgs, got  %v", msgs.Len())
	}

	//query for non-existant entity
	query = search.Query().Search("_exists_:blam")
	testSearch = search.Search("twitter").Type("tweet").Size("10000").Query(query)

	q = datastore.NewQuery(ctx)
	msgs, err = q.Execute(testSearch)

	if err != nil {
		t.Errorf("Unepected error %v", err)
	}
	if msgs.Len() != 0 {
		t.Errorf("Expected 0 msgs, got  %v", msgs.Len())
	}

}
