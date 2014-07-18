// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package user

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	mappingString = `
{
     "user": {
      "properties":{
        "Name":           {"type": "string", "index":"not_analyzed"},
        "Password":       {"type": "string", "index":"not_analyzed"}
      }
    }
}
`
	//MAPPING is the elastic mapping for a host
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating user mapping: %v", mappingError)
	}
}
