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

package utils

import (
	"encoding/json"
	"testing"
)

func TestEngineeringNotationParse(t *testing.T) {
	tests := []struct {
		val      string
		expected uint64
		error    bool
	}{
		{"123", 123, false},
		{"123k", 123 * (1 << 10), false},
		{"123K", 123 * (1 << 10), false},
		{"123M", 123 * (1 << 20), false},
		{"123G", 123 * (1 << 30), false},
		{"123T", 123 * (1 << 40), false},
		{"", 0, false},
		{"123P", 0, true},
		{"10 K", 0, true},
		{"Foobar", 0, true},
	}

	for _, test := range tests {
		i, e := ParseEngineeringNotation(test.val)
		if i != test.expected || (e != nil) != test.error {
			t.Errorf("For \"%s\", expected %d, got %d", test.val, test.expected, i)
		}
	}
}

func TestEngineeringNotationMarshal(t *testing.T) {
	value := "100M"
	expected := "\"" + value + "\""
	en := EngNotation{value, 0}
	b, _ := json.Marshal(en)
	if string(b) != expected {
		t.Errorf("Expected \"%s\" but got \"%s\"", expected, b)
	}
}

func TestEngineeringNotationUnmarshal(t *testing.T) {
	var en EngNotation
	json.Unmarshal([]byte("\"1K\""), &en)
	if en.Value != 1024 || en.source != "1K" {
		t.Fail()
	}
}

func TestStruct(t *testing.T) {
	a := struct {
		foo int
		Bar EngNotation
	}{1, EngNotation{"X", 11}}
	expected := "{\"Bar\":\"X\"}"
	b, e := json.Marshal(a)
	if e != nil || string(b) != expected {
		t.Fail()
	}
}
