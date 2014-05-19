// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package serviceconfigfile

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore/elastic"
)

var (
	mappingString = `
{
	"svcconfigfile": {
	  "properties": {
		"ID" :             {"type": "string", "index":"not_analyzed"},
		"ServiceTenantID": {"type": "string", "index":"not_analyzed"},
		"ServicePath":     {"type": "string", "index":"not_analyzed"}
	  }
	}
}
`
	//MAPPING is the elastic mapping for a service
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating svcconfigfile mapping: %v", mappingError)
	}
}
