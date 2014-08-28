// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	mappingString = `
{
    "host": {
      "properties":{
        "ID" :            {"type": "string", "index":"not_analyzed"},
        "Name":           {"type": "string", "index":"not_analyzed"},
        "KernelVersion":  {"type": "string", "index":"not_analyzed"},
        "KernelRelease":  {"type": "string", "index":"not_analyzed"},
        "PoolID":         {"type": "string", "index":"not_analyzed"},
        "IpAddr":         {"type": "string", "index":"not_analyzed"},
        "Cores":          {"type": "long", "index":"not_analyzed"},
        "Memory":         {"type": "long", "index":"not_analyzed"},
        "PrivateNetwork": {"type": "string", "index":"not_analyzed"},
        "CreatedAt" :     {"type": "date", "format" : "dateOptionalTime"},
        "UpdatedAt" :     {"type": "date", "format" : "dateOptionalTime"},
        "IPs" :{
          "properties":{
            "IP" : {"type": "string", "index":"not_analyzed"},
            "InterfaceName" : {"type": "string", "index":"not_analyzed"},
            "State" : {"type": "string", "index":"not_analyzed"}
          }
        }
      }
    }
}
`
	//MAPPING is the elastic mapping for a host
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating host mapping: %v", mappingError)
	}
}
