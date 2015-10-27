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

package registry

import (
	"encoding/base64"
	"fmt"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/zenoss/glog"
)

const kind = "imageregistry"

var (
	mappingString = fmt.Sprintf(`                                                              
{
    "%s": {
        "properties": {
            "Library":  {"type": "string", "index": "not_analyzed"},
            "Repo":     {"type": "string", "index": "not_analyzed"},
            "Tag":      {"type": "string", "index": "not_analyzed"},
            "UUID":     {"type": "string", "index": "not_analyzed"}
        }
    }
}
`, kind)
	// MAPPING is the elastic mapping for the docker registry
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		glog.Fatalf("error creating image registry mapping: %s", mappingError)
	}
}

func Key(id string) datastore.Key {
	enc := base64.URLEncoding.EncodeToString([]byte(id))
	return datastore.NewKey(kind, enc)
}

func DecodeKey(id string) (string, error) {
	dec, err := base64.URLEncoding.DecodeString(id)
	if err != nil {
		return "", err
	}
	return string(dec), nil
}
