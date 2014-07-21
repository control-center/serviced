// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/event"
)

//---------------------------------------------------------------------------
// Event CRUD

// ProcessEvent looks up the event and inserts, updates or deletes it in the store
func (f *Facade) ProcessEvent(ctx datastore.Context, e *event.Event) error {
	glog.V(2).Infof("Facade.ProcessEvent: %v", e)
	evt, err := f.GetHost(ctx, e.ID)
	if err != nil {
		return err
	}
	if evt == nil {
		// insert a new event
		return nil
	}
	// update an existing event
	return nil
}

// RemoveEvent removes an Event from serviced
func (f *Facade) RemoveEvent(ctx datastore.Context, eventID string) (err error) {
	glog.V(2).Infof("Facade.RemoveEvent: %s", eventID)

	var _event *event.Event
	if _event, err = f.GetEvent(ctx, eventID); err != nil {
		return err
	} else if _event == nil {
		return nil
	}

	return f.eventStore.Delete(ctx, event.EventKey(eventID))
}

// GetEvent gets an event by id. Returns nil if event is not found
func (f *Facade) GetEvent(ctx datastore.Context, eventID string) (*event.Event, error) {
	glog.V(2).Infof("Facade.GetEvent: id=%s", eventID)

	var value event.Event
	err := f.eventStore.Get(ctx, event.EventKey(eventID), &value)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// GetEvents returns a list of all events
func (f *Facade) GetEvents(ctx datastore.Context) ([]*event.Event, error) {
	return f.eventStore.GetN(ctx, 10000)
}
