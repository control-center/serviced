// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/control-center/serviced/domain/event"

	"errors"
)

// GetEvent gets an event
func (s *Server) GetEvent(eventID string, reply *event.Event) error {
	response, err := s.f.GetEvent(s.context(), eventID)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("events_server.go: event not found")
	}
	*reply = *response
	return nil
}

// GetEvents returns the system events
func (s *Server) GetEvents(_ *struct{}, reply *[]*event.Event) error {
	events, err := s.f.GetEvents(s.context())
	if err != nil {
		return err
	}
	*reply = events
	return nil
}

// ProcessEvent processes an event
func (s *Server) ProcessEvent(evt event.Event, _ *struct{}) error {
	return s.f.ProcessEvent(s.context(), &evt)
}

// RemoveEvent removes the event
func (s *Server) RemoveEvent(eventID string, _ *struct{}) error {
	return s.f.RemoveEvent(s.context(), eventID)
}
