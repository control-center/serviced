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

package api

import (
	"math"
	"reflect"
	"testing"
)

func testConvertOffsets(t *testing.T, received []string, expected []uint64) {
	converted, err := convertOffsets(received)
	if err != nil {
		t.Fatalf("unexpected error converting offsets: %s", err)
	}
	if !reflect.DeepEqual(converted, expected) {
		t.Fatalf("got %v expected %v", converted, expected)
	}
}

func testUint64sAreSorted(t *testing.T, values []uint64, expected bool) {
	if uint64sAreSorted(values) != expected {
		t.Fatalf("expected %v for sortedness for values: %v", expected, values)
	}
}

func testGetMinValue(t *testing.T, values []uint64, expected uint64) {
	if getMinValue(values) != expected {
		t.Fatalf("expected min value %v from values: %v", expected, values)
	}
}

func testGenerateOffsets(t *testing.T, inMessages []string, inOffsets, expected []uint64) {
	converted := generateOffsets(inMessages, inOffsets)
	if !reflect.DeepEqual(converted, expected) {
		t.Fatalf("unexpected error generating offsets from %v:%v got %v expected %v", inMessages, inOffsets, converted, expected)
	}
}

func TestLogs_Offsets(t *testing.T) {
	testConvertOffsets(t, []string{"123", "456", "789"}, []uint64{123, 456, 789})
	testConvertOffsets(t, []string{"456", "123", "789"}, []uint64{456, 123, 789})

	testUint64sAreSorted(t, []uint64{123, 124, 125}, true)
	testUint64sAreSorted(t, []uint64{123, 125, 124}, false)
	testUint64sAreSorted(t, []uint64{125, 123, 124}, false)

	testGetMinValue(t, []uint64{}, math.MaxUint64)
	testGetMinValue(t, []uint64{125, 123, 124}, 123)

	testGenerateOffsets(t, []string{}, []uint64{}, []uint64{})
	testGenerateOffsets(t, []string{"abc", "def", "ghi"}, []uint64{456, 123, 789}, []uint64{123, 124, 125})
	testGenerateOffsets(t, []string{"abc", "def", "ghi"}, []uint64{456, 124}, []uint64{124, 125, 126})
}
