/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

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
	if err.Error() != "Minimum instances larger than maximum instances: Min=10; Max=5" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative min
	mm.Min, mm.Max = -1, 1
	err = mm.Validate()
	if err.Error() != "Instances constraints must be positive: Min=-1; Max=1" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative max
	mm.Min, mm.Max = 1, -1
	err = mm.Validate()
	if err.Error() != "Instances constraints must be positive: Min=1; Max=-1" {
		t.Errorf("Unexpected error: %v", err)
	}

	// negative min and max
	mm.Min, mm.Max = -10, -10
	err = mm.Validate()
	if err.Error() != "Instances constraints must be positive: Min=-10; Max=-10" {
		t.Errorf("Unexpected error: %v", err)
	}

}
