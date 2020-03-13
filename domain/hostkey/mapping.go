// Copyright 2016 The Serviced Authors.
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

package hostkey

import (
	"fmt"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/logging"
)

const kind = "keyregistry"

var (
	plog          = logging.PackageLogger()
	mappingString = fmt.Sprintf(`
{
    "%s": {
        "properties": {
            "PEM": {"type": "string", "index": "no"}
        }
    }
}
`, kind)
	// MAPPING is the elastic mapping for the docker registry
	MAPPING, mappingError = elastic.NewMapping(mappingString)
)

func init() {
	if mappingError != nil {
		plog.WithError(mappingError).Fatal("error creating mapping for rsa key registry")
	}
}

// Key creates a datastore.Key object
func Key(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

// DecodeKey decodes the key
func DecodeKey(id string) (string, error) {
	return id, nil
}
