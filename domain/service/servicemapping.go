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

package service

import (
	"github.com/control-center/serviced/datastore/elastic"
)

//TODO Still can't understand the purpose of the below mapping in old ES
//"Tags":{"type": "text", "index_name":"tag"},

var (
	mappingString = `
{
  "properties": {
	"ID" :             {"type": "keyword", "index":"true"},
	"Name":            {"type": "keyword", "index":"true"},
	"Startup":         {"type": "keyword", "index":"true"},
	"Context":         {"type": "object", "enabled":"false"},
	"Description":     {"type": "keyword", "index":"true"},
	"DeploymentID":    {"type": "keyword", "index":"true"},
	"Environment":     {"type": "keyword", "index":"true"},
	"Tags":            {"type": "keyword"},
	"Instances":       {"type": "long", "index":"true"},
	"InstancesLimits":       {
	  "properties": {
		"Min": {"type": "long", "index":"true"},
		"Max": {"type": "long", "index":"true"}
	  }
	},
	"DesiredState":    {"type": "long", "index":"true"},
	"ImageID":         {"type": "keyword", "index":"true"},
	"PoolID":          {"type": "keyword", "index":"true"},
	"Launch":          {"type": "keyword", "index":"true"},
	"HostPolicy":      {"type": "keyword", "index":"true"},
	"Hostname":        {"type": "keyword", "index":"true"},
	"Privileged":      {"type": "keyword", "index":"true"},
	"ParentServiceID": {"type": "keyword", "index":"true"},
	"Volume":          {
	  "properties":    {
		"ResourcePath" : {"type": "keyword", "index":"true"},
		"ContainerPath": {"type": "keyword", "index":"true"}
	  }
	},
	"CreatedAt" :      {"type": "date", "format" : "date_optional_time"},
	"UpdatedAt" :      {"type": "date", "format" : "date_optional_time"},
	"EndPoints" :      {
	  "properties":    {
		"Protocol" :            {"type": "keyword", "index":"true"},
		"Application" :         {"type": "keyword", "index":"true"},
		"ApplicationTemplate" : {"type": "keyword", "index":"true"},
		"Purpose" :             {"type": "keyword", "index":"true"},
		"PortNumber" :          {"type": "long", "index":"true"},
		"VirtualAddress" :      {"type": "keyword", "index":"true"},
		"VHost" :               {"type": "keyword", "index":"true"}
	  }
	}
  },
  "dynamic_templates": [
	  {
		"ConfigFiles_strings_as_keywords": {
		  "match_mapping_type": "string",
		  "path_match":   "ConfigFiles.*",
		  "mapping": {
			"type": "keyword",
			"ignore_above": 256
		  }
		}
	  },
      {
		"RAMCommitment_EngNotation_as_keyword": {
		  "match_mapping_type": "long",
		  "path_match":   "*RAMCommitment*",
		  "mapping": {
			"type": "keyword",
			"index": "true"
		  }
		}
	  },
	  {
		"OriginalConfigs_strings_as_keywords": {
		  "match_mapping_type": "string",
		  "path_match":   "OriginalConfigs.*",
		  "mapping": {
			"type": "keyword",
			"ignore_above": 256
		  }
		}
	  }
  ]
}
`
	//MAPPING is the elastic mapping for a service
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		plog.WithError(mappingError).Fatal("error creating mapping for the service object")
	}
}
