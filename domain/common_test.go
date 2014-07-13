// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package domain

import (
	"encoding/json"
	"testing"
	"time"
)

const testHcJSON = `{"Type":"health","Script":"foo","Interval":1.5,"FailureSeverity":4}`

var testHc = StatusCheck{Type: "health", Script: "foo", FailureSeverity: 4, Interval: time.Millisecond * 1500}

func TestStatusCheck(t *testing.T) {
	var hc StatusCheck
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
