// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package domain

import (
	"testing"
)

func TestMinMax(t *testing.T) {

	mm := MinMax{}
	//validate default
	err := mm.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//same
	mm.Min, mm.Max = 1, 1
	err = mm.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//0 to 100
	mm.Min, mm.Max = 0, 100
	err = mm.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//min > 0
	mm.Min, mm.Max = 10, 0
	err = mm.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//min > max
	mm.Min, mm.Max = 10, 5
	err = mm.Validate()
	if err.Error() != "minimum instances larger than maximum instances: Min=10; Max=5" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative min
	mm.Min, mm.Max = -1, 1
	err = mm.Validate()
	if err.Error() != "instances constraints must be positive: Min=-1; Max=1" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative max
	mm.Min, mm.Max = 1, -1
	err = mm.Validate()
	if err.Error() != "instances constraints must be positive: Min=1; Max=-1" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative min and max
	mm.Min, mm.Max = -10, -10
	err = mm.Validate()
	if err.Error() != "instances constraints must be positive: Min=-10; Max=-10" {
		t.Errorf("Unexpected error: %v", err)
	}

}
