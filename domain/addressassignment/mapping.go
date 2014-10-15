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

package addressassignment

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore/elastic"
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
