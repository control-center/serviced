// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore/elastic"
)

var (
	mappingString = `
{
	"service": {
	  "properties": {
		"Id" :             {"type": "string", "index":"not_analyzed"},
		"Name":            {"type": "string", "index":"not_analyzed"},
		"Startup":         {"type": "string", "index":"not_analyzed"},
		"Context":         {"type": "string", "index":"not_analyzed"},
		"Description":     {"type": "string", "index":"not_analyzed"},
		"DeploymentID":    {"type": "string", "index":"not_analyzed"},
		"Tags":            {"type": "string", "index_name": "tag"},
		"Instances":       {"type": "long",   "index":"not_analyzed"},
		"InstancesLimits":       {
		  "properties": {
			"Min": {"type": "long", "index":"not_analyzed"},
			"Max": {"type": "long", "index":"not_analyzed"}
		  }
		},
		"DesiredState":    {"type": "long", "index":"not_analyzed"},
		"ImageID":           {"type": "string", "index":"not_analyzed"},
		"PoolID":          {"type": "string", "index":"not_analyzed"},
		"Launch":          {"type": "string", "index":"not_analyzed"},
		"HostPolicy":      {"type": "string", "index":"not_analyzed"},
		"Hostname":        {"type": "string", "index":"not_analyzed"},
		"Privileged":      {"type": "string", "index":"not_analyzed"},
		"ParentServiceID": {"type": "string", "index":"not_analyzed"},
		"Volume":          {
		  "properties":    {
			"ResourcePath" : {"type": "string", "index":"not_analyzed"},
			"ContainerPath": {"type": "string", "index":"not_analyzed"}
		  }
		},
		"CreatedAt" :      {"type": "date", "format" : "dateOptionalTime"},
		"UpdatedAt" :      {"type": "date", "format" : "dateOptionalTime"},
		"ConfigFiles":     {
		  "properties": {
			"": {"type": "string", "index": "not_analyzed"},
			"": {"type": "string", "index": "not_analyzed"},
			"": {"type": "string", "index": "not_analyzed"}
		  }
		},
		"OriginalConfigs":     {
		  "properties": {
			"": {"type": "string", "index": "not_analyzed"},
			"": {"type": "string", "index": "not_analyzed"},
			"": {"type": "string", "index": "not_analyzed"}
		  }
		},
		"EndPoints" :      {
		  "properties":    {
			"Protocol" :            {"type": "string", "index":"not_analyzed"},
			"Application" :         {"type": "string", "index":"not_analyzed"},
			"ApplicationTemplate" : {"type": "string", "index":"not_analyzed"},
			"Purpose" :             {"type": "string", "index":"not_analyzed"},
			"PortNumber" :          {"type": "long",   "index":"not_analyzed"},
			"VirtualAddress" :      {"type": "string", "index":"not_analyzed"},
			"VHost" :               {"type": "string", "index":"not_analyzed"}
		  }
		},
		"Tasks": {
		  "properties": {
			"Name" :           {"type": "string", "index":"not_analyzed"},
			"Schedule" :       {"type": "string", "index":"not_analyzed"},
			"Command" :        {"type": "string", "index":"not_analyzed"},
			"LastRunAt" :      {"type": "date", "format" : "dateOptionalTime"},
			"TotalRunCount" :  {"type": "long",   "index":"not_analyzed"}
		  }
		}
	  }
	}
}
`
	//MAPPING is the elastic mapping for a service
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating service mapping: %v", mappingError)
	}
}
