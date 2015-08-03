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

// +build unit

package domain

import (
	"encoding/json"
	"testing"
	"time"
)

const testHcJSON = `{"Script":"foo","Interval":1.5,"Timeout":1.2}`

var testHc = HealthCheck{Timeout: time.Millisecond * 1200, Script: "foo", Interval: time.Millisecond * 1500}

func TestHealthCheck(t *testing.T) {
	var hc HealthCheck
	if err := json.Unmarshal([]byte(testHcJSON), &hc); err != nil {
		t.Fatalf("Could not unmarshal test health check: %s", err)
	}
	if hc != testHc {
		t.Fatalf("test hc values is not equal: %v vs %v", hc, testHc)
	}

	// test marshalling
	data, err := json.Marshal(testHc)
	if err != nil {
		t.Fatalf("could not marshal test health check: %s", err)
	}

	str := string(data)
	if str != testHcJSON {
		t.Fatalf("%s does not equal to  %s", str, testHcJSON)
	}

}
