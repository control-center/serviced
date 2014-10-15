// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
