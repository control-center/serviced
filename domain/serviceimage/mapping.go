// Copyright 2015 The Serviced Authors.
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

package serviceimage

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	mappingString = `
{
	"serviceimage": {
	  "properties":{
		"ImageID" :       {"type": "string", "index": "not_analyzed"},
		"UUID":           {"type": "string", "index": "not_analyzed"},
		"HostID":         {"type": "string", "index": "not_analyzed"},
		"Status":         {"type": "long", "index": "not_analyzed"},
		"Error":          {"type": "string", "index": "not_analyzed"},
		"CreatedAt" :     {"type": "date", "format" : "dateOptionalTime"},
		"DeployedAt" :    {"type": "date", "format" : "dateOptionalTime"}
	  }
	}
}
`
	// MAPPING is the elastic mapping for a docker image
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating docker image mapping: %v", mappingError)
	}
}
