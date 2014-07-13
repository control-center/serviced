// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package event

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"testing"
	"time"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&S{
	ElasticTest: elastic.ElasticTest{
		Index:    "controlplane",
		Mappings: []elastic.Mapping{MAPPING},
	}})

type S struct {
	elastic.ElasticTest
	ctx datastore.Context
	es  *Store
}

func (s *S) SetUpTest(c *C) {
	s.ElasticTest.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.Get()
	s.es = NewStore()
}

var testEvent = Event{
	ID:        "testid",
	HostID:    "testHostID",
	Agent:     "go test",
	Severity:  1,
	Count:     1,
	Class:     "/test/class",
	FirstTime: time.Date(2014, 2, 1, 0, 0, 0, 0, time.UTC),
	LastTime:  time.Date(2014, 2, 1, 0, 0, 0, 0, time.UTC),
}

func (s *S) Test_EventCRUD(t *C) {
	defer s.es.Delete(s.ctx, EventKey("testid"))

	var event Event
	var event2 Event

	if err := s.es.Get(s.ctx, EventKey("testid"), &event2); !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Expected ErrNoSuchEntity, got: %v", err)
	}

	err := s.es.Put(s.ctx, EventKey("testid"), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.ID = testEvent.ID
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.HostID = testEvent.HostID
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.Agent = testEvent.Agent
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.Severity = testEvent.Severity
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.Count = testEvent.Count
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.Class = testEvent.Class
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.FirstTime = testEvent.FirstTime
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err == nil {
		t.Errorf("Unexpected success creating event %-v", &event)
	}

	event.LastTime = testEvent.LastTime
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)

	// at this point event should be valid
	if err != nil {
		t.Errorf("Unexpected failure creating event %-v", &event)
	}

	// Test Get
	err = s.es.Get(s.ctx, EventKey(event.ID), &event2)
	if err != nil {
		t.Errorf("Unexpected failure getting event %s", err)
	}

	if event != event2 {
		t.Errorf("events should be equal: %+v != %+v", event, event2)
	}

	// Test Update
	event.LastTime = time.Now()
	event.Count += 1
	err = s.es.Put(s.ctx, EventKey(event.ID), &event)
	if err != nil {
		t.Errorf("could not update event: %s", err)
	}

	err = s.es.Get(s.ctx, EventKey(event.ID), &event2)
	if err != nil {
		t.Errorf("Unexpected failure getting event %s", err)
	}
	if event != event2 {
		t.Errorf("events should be equal: %+v != %+v", event, event2)
	}

	// Test delete
	err = s.es.Delete(s.ctx, EventKey(event.ID))
	if err != nil {
		t.Errorf("could not delete event: %s", err)
	}

	err = s.es.Get(s.ctx, EventKey(event.ID), &event2)
	if err == nil {
		t.Errorf("Unexpected success getting event %s", err)
	}
}
