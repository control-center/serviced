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

package service

import "testing"

type roundTest struct {
	value    float64
	expected int64
}

var roundTests = []roundTest{
	roundTest{0.49, 0},
	roundTest{0.5, 1},
	roundTest{0.99, 1},
	roundTest{1.001, 1},
	roundTest{-0.4999, 0},
	roundTest{-0.999, -1},
	roundTest{-1.001, -1},
}

func TestRound(t *testing.T) {
	for _, test := range roundTests {
		if result := round(test.value); result != test.expected {
			t.Logf("round(%f) = %d, expected %d", test.value, result, test.expected)
			t.Fail()
		}
	}
}
