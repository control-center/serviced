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

package container

import (
	"testing"
)

func TestReadInt64Stats(t *testing.T) {

	stats, err := readInt64Stats("stats_test_ok")
	if err != nil {
		t.Fatalf("unexpected error reading stats")
	}
	if val, ok := stats["zero"]; !ok || val != 0 {
		t.Fatalf("zero value could not be verified")
	}
	if val, ok := stats["positive"]; !ok || val != 9876543210 {
		t.Fatalf("positive value could not be verified")
	}
	if val, ok := stats["negative"]; !ok || val != -9876543210 {
		t.Fatalf("negative value could not be verified")
	}
}

func TestGetOpenConnections(t *testing.T) {

	conns, err := getOpenConnections("testfiles/proc.net.tcp")
	if err != nil {
		t.Fatalf("unexpected error reading procfile")
	}
	if conns != 28 {
		t.Fatalf("expected 28 open connections, but got %d", conns)
	}

	conns, err = getOpenConnections("testfiles/proc.net.tcp.bad")
	if err != nil {
		t.Fatalf("unexpected error reading procfile")
	}
	if conns != 0 {
		t.Fatalf("expected 0 open connections, but got %d", conns)
	}
}
