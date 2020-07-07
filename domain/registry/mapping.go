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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/logging"
)

const kind = "imageregistry"

// initialize the package logger
var plog = logging.PackageLogger()

var (
	mappingString = `                                                              
{
	"properties": {
		"Library":  {"type": "keyword", "index":"true"},
		"Repo":     {"type": "keyword", "index":"true"},
		"Tag":      {"type": "keyword", "index":"true"},
		"UUID":     {"type": "keyword", "index":"true"}
	}
}
`
	// MAPPING is the elastic mapping for the docker registry
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		plog.WithError(mappingError).Fatal("error creating mapping for the imageregistry object")
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
