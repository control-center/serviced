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
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	mappingString = `
{
  "properties":{
	"ID" :            {"type": "keyword", "index":"true"},
	"Name":           {"type": "keyword", "index":"true"},
	"KernelVersion":  {"type": "keyword", "index":"true"},
	"KernelRelease":  {"type": "keyword", "index":"true"},
	"PoolID":         {"type": "keyword", "index":"true"},
	"IpAddr":         {"type": "keyword", "index":"true"},
	"Cores":          {"type": "long", "index":"true"},
	"Memory":         {"type": "long", "index":"true"},
	"PrivateNetwork": {"type": "keyword", "index":"true"},
	"CreatedAt" :     {"type": "date", "format" : "date_optional_time"},
	"UpdatedAt" :     {"type": "date", "format" : "date_optional_time"},
	"IPs" :{
	  "properties":{
		"IP" : {"type": "keyword", "index":"true"},
		"InterfaceName" : {"type": "keyword", "index":"true"},
		"State" : {"type": "keyword", "index":"true"}
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
		plog.WithError(mappingError).Fatal("error creating mapping for the host object")
	}
}
