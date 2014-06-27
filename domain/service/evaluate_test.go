// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"testing"

	"github.com/zenoss/serviced/domain/servicedefinition"
)

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

func TestConfigFiles(t *testing.T) {
	s := &Service{
		Volumes: []servicedefinition.Volume{
			servicedefinition.Volume{
				ResourcePath: "/path/to/x-{{ plus 1 .InstanceID}}",
			},
		},
	}
	gs := func(serviceID string) (Service, error) {
		return *s, nil
	}
	instanceID := 3
	s.EvaluateVolumesTemplate(gs, instanceID)
	if s.Volumes[0].ResourcePath != "/path/to/x-4" {
		t.Logf("Not equal: %s vs. %s", s.Volumes[0].ResourcePath, "/path/to/x-4")
		t.Fail()
	}

}
