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
