// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package integration_test

import (
	"github.com/mattbaird/elastigo/search"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/context"
	"github.com/zenoss/serviced/datastore/key"
	"github.com/zenoss/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"reflect"
	"testing"
	"time"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{ElasticTest: elastic.ElasticTest{Index: "twitter"}})

type S struct {
	elastic.ElasticTest
	ctx context.Context
}

func (s *S) SetUpTest(c *C) {
	context.Register(s.Driver())
	s.ctx = context.Get()
}

func (s *S) TestPutGetDelete(t *C) {
	ctx := s.ctx
	ds := datastore.New()

	key := key.New("tweet", "1")
	tweet := tweettest{"kimchy", "", "2009-11-15T14:12:12", "trying out Elasticsearch"}

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

func (s *S) TestQuery(t *C) {
	ctx := s.ctx

	ds := datastore.New()

	k := key.New("tweet", "1")
	tweet := &tweettest{"kimchy", "NY", "2010-11-15T14:12:12", "trying out Elasticsearch"}

	err := ds.Put(ctx, k, tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	k = key.New("tweet", "2")
	tweet = &tweettest{"kimchy2", "NY", "2010-11-15T14:12:12", "trying out Elasticsearch again"}
	err = ds.Put(ctx, k, tweet)
	if err != nil {
		t.Errorf("%v", err)
	}

	time.Sleep(2000 * time.Millisecond)

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
	User      string
	State     string
	Post_date string
	Message   string
}

func (t *tweettest) ValidEntity() error {
	return nil
}
