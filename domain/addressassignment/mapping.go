// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package addressassignment

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore/elastic"
)

var (
	mappingString = `
{
    "addressassignment": {
      "properties": {
        "AssignmentType" :  {"type": "string", "index":"not_analyzed"},
        "HostID":           {"type": "string", "index":"not_analyzed"},
        "PoolID":           {"type": "string", "index":"not_analyzed"},
        "IPAddr" :          {"type": "string", "index":"not_analyzed"},
        "Port" :            {"type": "long", "index":"not_analyzed"},
        "ServiceID" :       {"type": "string", "index":"not_analyzed"},
        "EndpointName" :    {"type": "string", "index":"not_analyzed"}
      }
    }
}
`
	//MAPPING is the elastic mapping for an address assignment
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating  addressassignment: %v", mappingError)
	}
}
