// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package domain

import (
	"encoding/json"
	"testing"
	"time"
)

const testHcJSON = `{ "Script": "foo", "Interval": 1.5 }`

var testHc = HealthCheck{Script: "foo", Interval: time.Millisecond * 1500}

func TestHealthCheck(t *testing.T) {
	var hc HealthCheck
	if err := json.Unmarshal([]byte(testHcJSON), &hc); err != nil {
		t.Fatalf("Could not unmarshal test health check: %s", err)
	}
	if hc != testHc {
		t.Fatalf("test hc values is not equal: %v vs %v", hc, testHc)
	}
}
