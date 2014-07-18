// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package servicetemplate

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore/elastic"
)

var (
	mappingString = `
{
  "servicetemplatewrapper" : {
    "properties" : {
      "ID" : {
        "type"  : "string",
        "index" : "not_analyzed"
      },
      "Name" : {
        "type"  : "string",
        "index" : "not_analyzed"
      },
      "Description" : {
        "type"  : "string",
        "index" : "not_analyzed"
      },
      "APIVersion" : {
        "type"  : "long",
        "index" : "not_analyzed"
      },
      "TemplateVersion" : {
        "type"  : "long",
        "index" : "not_analyzed"
      },
      "Data" : {
        "type"  : "string",
        "index" : "not_analyzed"
      }
    }
  }
}
`
	//MAPPING is the elastic mapping for a service template
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating host mapping: %v", mappingError)
	}
}
