// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package event

import (
	"time"

	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	Fingerprint       string
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

	mappingString = `
{
    "event": {
      "properties":{
        "Fingerprint" :      {"type": "string", "index":"not_analyzed"},
        "Severity":          {"type": "long", "index":"not_analyzed"},
        "EventState":        {"type": "long", "index":"not_analyzed"},
        "Summary":           {"type": "string", "index":"not_analyzed"},
        "FirstTime":         {"type": "date", "format":"dateOptionalTime"},
        "LastTime":          {"type": "date", "format":"dateOptionalTime"},
        "Count":             {"type": "long", "index":"not_analyzed"},
        "Agent":             {"type": "string", "index":"not_analyzed"},
        "HostID":            {"type": "string", "index":"not_analyzed"},
        "ServiceID":         {"type": "string", "index":"not_analyzed"},
        "ServiceInstanceID": {"type": "long", "index":"not_analyzed"}
      }
    }
}
`
	//MAPPING is the elastic mapping for events
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating event mapping: %v", mappingError)
	}
}
