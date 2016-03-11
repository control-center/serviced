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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package utils

import (
	"testing"
)

// Test GetIPv4Addresses()
func TestGetIPv4Addresses(t *testing.T) {
	ips, err := GetIPv4Addresses()
	if err != nil {
		t.Errorf("Failed to get ipv4 addresses: %s", err)
		t.Fail()
	}

	expectedMinimumLen := 1
	if len(ips) < expectedMinimumLen {
		t.Errorf("minimum IPs expected %d > retrieved %d  ips:%v", expectedMinimumLen, len(ips), ips)
		t.Fail()
	}
}
