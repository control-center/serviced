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

package servicetemplate

import (
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	mappingString = `
{
	"properties" : {
	  "ID" : {
		"type"  : "keyword",
		"index" : "true"
	  },
	  "Name" : {
		"type"  : "keyword",
		"index" : "true"
	  },
	  "Version" : {
		"type"  : "keyword",
		"index" : "true"
	  },
	  "Description" : {
		"type"  : "keyword",
		"index" : "true"
	  },
	  "APIVersion" : {
		"type"  : "long",
		"index" : "true"
	  },
	  "TemplateVersion" : {
		"type"  : "long",
		"index" : "true"
	  },
	  "Data" : {
		"type"  : "keyword",
		"index" : "true",
		"ignore_above": 10000
	  }
	}
}
`
	//MAPPING is the elastic mapping for a service template
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		plog.WithError(mappingError).Fatal("error creating mapping for the servicetemplate object")
	}
}
