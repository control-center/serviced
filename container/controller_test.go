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

// +build unit

package container

import (
	"testing"
)

func TestExportEndpoint(t *testing.T) {
	c := Controller{}

	endpoints := c.createControlplaneEndpoints()

	portlist := []string{"443", "8443", "5042", "5043", "5601"}
	for _, port := range portlist {
		key := "tcp:" + port
		if _, ok := endpoints[key]; !ok {
			t.Fatalf(" mapping failed missing key[\"%s\"]", key)
		}
		if len(endpoints[key]) != 1 {
			t.Fatalf(" mapping failed len(\"%s\"])=%d expected 1", key, len(endpoints[key]))
		}

	}

}
