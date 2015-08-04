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
	"reflect"
	"testing"
	"time"
)

const testDTHjson = `{"Min":1,"Max":99,"TimePeriod":1.5,"Percentage":12}`

var one int64 = 1
var ninenine int64 = 99

var testDTH = DurationThreshold{
	Min:        &one,
	Max:        &ninenine,
	TimePeriod: time.Millisecond * 1500,
	Percentage: 12,
}

func TestDurationThresholdSerialize(t *testing.T) {
	var dt DurationThreshold
	if err := json.Unmarshal([]byte(testDTHjson), &dt); err != nil {
		t.Fatalf("Could not unmarshal test duration threshold: %s", err)
	}
	if !reflect.DeepEqual(dt, testDTH) {
		t.Fatalf("test duration theshold values are not equal: %v vs %v", dt, testDTH)
	}

	// test marshalling
	data, err := json.Marshal(testDTH)
	if err != nil {
		t.Fatalf("could not marshal test duration threshold: %s", err)
	}

	str := string(data)
	if str != testDTHjson {
		t.Fatalf("%s does not equal to  %s", str, testDTHjson)
	}
}
