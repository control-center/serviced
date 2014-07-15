// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
