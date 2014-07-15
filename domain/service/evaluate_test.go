// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
