// Copyright 2017 The Serviced Authors.
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

package logfilter

import (
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/logging"
)

var (
	kind          = "logfilter"
	plog          = logging.PackageLogger()
	mappingString = `
	{
	  "properties":{
		"Name":           {"type": "keyword", "index":"true"},
		"Version":        {"type": "keyword", "index":"true"},
		"Filter":         {"type": "keyword", "index":"true"}
	  }
	}`
	// MAPPING is the elastic mapping for a log filter
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		plog.WithError(mappingError).Fatal("error creating mapping for the logfilter object")
	}
}
