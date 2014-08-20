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

package utils

import (
	"testing"
)

func TestStringSliceEquals(t *testing.T) {
	if !StringSliceEquals([]string{}, []string{}) {
		t.Fatalf("Expect %+v == %+v", []string{}, []string{})
	}

	if StringSliceEquals([]string{}, nil) {
		t.Fatalf("Expect %+v != %+v", []string{}, nil)
	}

	if StringSliceEquals(nil, []string{}) {
		t.Fatalf("Expect %+v != %+v", nil, []string{})
	}

	if StringSliceEquals([]string{"a", "b"}, []string{"a", "b", "c"}) {
		t.Fatalf("Expect %+v != %+v", []string{"a", "b"}, []string{"a", "b", "c"})
	}

	if StringSliceEquals([]string{"a", "b", "c"}, []string{"a", "b"}) {
		t.Fatalf("Expect %+v != %+v", []string{"a", "b", "c"}, []string{"a", "b"})
	}

	if !StringSliceEquals([]string{"a", "b", "c"}, []string{"a", "b", "c"}) {
		t.Fatalf("Expect %+v == %+v", []string{"a", "b", "c"}, []string{"a", "b", "c"})
	}
}

func TestStringRemoveFromSlice(t *testing.T) {

	a := []string{"a"}
	a_b := []string{"a", "b"}
	a_b_c := []string{"a", "b", "c"}
	empty := []string{}

	if !StringSliceEquals(a_b_c, StringRemoveFromSlice("d", a_b_c)) {
		t.Fatalf("Expected %+v got %+v", a_b_c, StringRemoveFromSlice("d", a_b_c))
	}

	if !StringSliceEquals(a, StringRemoveFromSlice("b", a_b)) {
		t.Fatalf("Expected %+v got %+v", a, StringRemoveFromSlice("d", a_b))
	}

	if !StringSliceEquals(empty, StringRemoveFromSlice("a", a)) {
		t.Fatalf("Expected %+v got %+v", empty, StringRemoveFromSlice("a", a))
	}
}
