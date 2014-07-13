// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package event

import (
	"time"
)

// Event represents the state of an event in the system
type Event struct {
	ID                string
	Class             string    // class of event, eg /status/health/redis_client
	Severity          uint8     // 0 clear, 1 debug, 2 info, 3 warning, 4 error, 5 critical
	EventState        uint8     // 0 new, 1 acknowledged, 2 suppressed
	Summary           string    // human readable summary
	FirstTime         time.Time // time the event first occurred
	LastTime          time.Time // last time the event occcured
	Count             int       // number of times the event has occurred
	Agent             string    // the program/agent who sent the event
	HostID            string    // hostid where the event was generated
	ServiceID         string    // serviceid of the service, if applicable
	ServiceInstanceID int       // instance id of service, if applicable
}

// New creates a new empty event
func New() *Event {
	return &Event{}
}
