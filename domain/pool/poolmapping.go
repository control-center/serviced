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

package pool

import (
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/zenoss/glog"
)

var (
	mappingString = `
{
    "resourcepool": {
      "properties":{
        "ID" :          {"type": "string", "index":"not_analyzed"},
        "Realm":        {"type": "string", "index":"not_analyzed"},
        "Description" : {"type": "string", "index":"not_analyzed"},
        "CoreLimit":    {"type": "long", "index":"not_analyzed"},
        "MemoryLimit":  {"type": "long", "index":"not_analyzed"},
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
