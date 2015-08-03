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
)

func TestDataPointRateOptionsJSONDefined(t *testing.T) {
	d := DataPointRateOptions{
		Counter:        true,
		CounterMax:     123,
		ResetThreshold: 456,
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Error Marshaling JSON err=%+v", err)
	}
	var d2 DataPointRateOptions
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatalf("Error Marshaling JSON err=%+v", err)
	}
	if d2.Counter != true {
		t.Fatalf("Improper Marshaling/Unmarshaling of data.")
	}
	if d2.CounterMax != 123 {
		t.Fatalf("Improper Marshaling/Unmarshaling of data.")
	}
	if d2.ResetThreshold != 456 {
		t.Fatalf("Improper Marshaling/Unmarshaling of data.")
	}

}

func TestDataPointRateOptionsJSONUndefined(t *testing.T) {
	d := DataPointRateOptions{
		Counter: true,
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Error Marshaling JSON err=%+v", err)
	}
	var d2 DataPointRateOptions
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatalf("Error Marshaling JSON err=%+v", err)
	}
	if d2.Counter != true {
		t.Fatalf("Improper Marshaling/Unmarshaling of data.")
	}
	if d2.CounterMax != 0 {
		t.Fatalf("Improper Marshaling/Unmarshaling of data.")
	}
	if d2.ResetThreshold != 0 {
		t.Fatalf("Improper Marshaling/Unmarshaling of data.")
	}

}
