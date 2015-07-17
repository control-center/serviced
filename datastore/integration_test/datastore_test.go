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

package integration_test

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/zenoss/elastigo/search"
	"github.com/zenoss/glog"

	. "gopkg.in/check.v1"

	"reflect"
	"testing"
)

var version datastore.VersionedEntity

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{ElasticTest: elastic.ElasticTest{Index: "twitter"}})

type S struct {
	elastic.ElasticTest
	ctx datastore.Context
}

func (s *S) SetUpTest(c *C) {
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
}

func (s *S) TestPutGetDelete(t *C) {
	ctx := s.ctx
	ds := datastore.New()

	key := datastore.NewKey("tweet", "1")
	tweet := tweettest{"kimchy", "", "2009-11-15T14:12:12", "trying out Elasticsearch", version}

	err := ds.Put(ctx, key, &tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	//Get tweet
	var storedtweet tweettest
	err = ds.Get(ctx, key, &storedtweet)
	if err != nil {
		t.Fatalf("Unexpected: %v", err)
	}
	glog.Infof("tweet is %v", &storedtweet)

	if storedtweet.User != "kimchy" {
		t.Errorf("Expected kimchy, found %s", storedtweet.User)
	}

	//Delete tweet
	err = ds.Delete(ctx, key)
	if err != nil {
		t.Errorf("Unexpected delete error: %v", err)
	}

	//test not found
	err = ds.Get(ctx, key, &storedtweet)
	if err == nil {
		t.Error("Expected error, not nil")
	} else if !datastore.IsErrNoSuchEntity(err) {
		glog.Infof("type is %s", reflect.ValueOf(err))
		t.Fatalf("Unexpected: %v", err)
	}
}

func (s *S) TestVersionConflict(t *C) {
	ctx := s.ctx
	ds := datastore.New()

	key := datastore.NewKey("tweet", "666")
	tweet := tweettest{"kimchy", "", "2009-11-15T14:12:12", "trying out Elasticsearch", version}

	err := ds.Put(ctx, key, &tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	//Get tweet
	var storedtweet tweettest
	err = ds.Get(ctx, key, &storedtweet)
	if err != nil {
		t.Fatalf("Unexpected: %v", err)
	}
	if storedtweet.DatabaseVersion != 1 {
		t.Fatalf("Version was not incremented")
	}

	// Update something and send it back with the same version; it should succeed
	storedtweet.Message = "This is a different message"
	err = ds.Put(ctx, key, &storedtweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	// Make a new tweet with a 1 version, which should conflict (since version
	// in the database is now 2)
	tweet.DatabaseVersion = 1
	err = ds.Put(ctx, key, &tweet)
	if err == nil {
		t.Errorf("Did not get a conflict")
	}

}

func (s *S) TestQuery(t *C) {
	ctx := s.ctx

	ds := datastore.New()

	k := datastore.NewKey("tweet", "123")
	tweet := &tweettest{"kimchy", "NY", "2010-11-15T14:12:12", "trying out Elasticsearch", version}

	err := ds.Put(ctx, k, tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	k = datastore.NewKey("tweet", "234")
	tweet = &tweettest{"kimchy2", "NY", "2010-11-15T14:12:12", "trying out Elasticsearch again", version}
	err = ds.Put(ctx, k, tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	query := search.Query().Search("_exists_:State")
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

type tweettest struct {
	User     string
	State    string
	PostDate string
	Message  string
	datastore.VersionedEntity
}

func (t *tweettest) ValidEntity() error {
	return nil
}
