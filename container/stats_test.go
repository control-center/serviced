// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
