// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/control-center/serviced/domain/event"
)

//GetEvent gets the event for the given eventID or nil
func (c *Client) GetEvent(eventID string) (*event.Event, error) {
	response := event.New()
	if err := c.call("GetEvent", eventID, response); err != nil {
		return nil, err
	}
	return response, nil
}

//GetEvent gets the event for the given eventID or nil
func (c *Client) GetEvents() ([]*event.Event, error) {
	response := make([]*event.Event, 0)
	if err := c.call("GetEvents", empty, &response); err != nil {
		return []*event.Event{}, err
	}
	return response, nil
}

//ProcessEvent processes an event
func (c *Client) ProcessEvent(event event.Event) error {
	return c.call("ProcessEvent", event, nil)
}

//RemoveEvent removes a event
func (c *Client) RemoveEvent(eventID string) error {
	return c.call("RemoveEvent", eventID, nil)
}
