// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore/elastic"
)

var (
	mappingString = `
{
    "resourcepool": {
      "properties":{
        "ID" :          {"type": "string", "index":"not_analyzed"},
        "Description" : {"type": "string", "index":"not_analyzed"},
        "ParentID":     {"type": "string", "index":"not_analyzed"},
        "CoreLimit":    {"type": "long", "index":"not_analyzed"},
        "MemoryLimit":  {"type": "long", "index":"not_analyzed"},
        "Priority":     {"type": "long", "index":"not_analyzed"},
        "CreatedAt" :   {"type": "date", "format" : "dateOptionalTime"},
        "UpdatedAt" :   {"type": "date", "format" : "dateOptionalTime"},
        "CoreCapacity": {"type": "long", "format": "not_analyzed"},
        "MemoryCapacity": {"type": "long", "format": "not_analyzed"}
      }
    }
}
`
	//MAPPING is the elastic mapping for a resource pool
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating pool mapping: %v", mappingError)
	}
}
