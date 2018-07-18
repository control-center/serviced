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
	"strings"
	"fmt"
)

// StringSliceEquals compare two string slices for equality
func StringSliceEquals(lhs []string, rhs []string) bool {
	if lhs == nil && rhs == nil {
		return true
	}

	if lhs == nil && rhs != nil {
		return false
	}

	if lhs != nil && rhs == nil {
		return false
	}

	if len(lhs) != len(rhs) {
		return false
	}

	for i := range lhs {
		if lhs[i] != rhs[i] {
			return false
		}
	}

	return true
}

// Extract all strings from an array of interfaces. If the input array contains items that are not strings, set the output flag (allConverted) to false.
func InterfaceArrayToStringArray(inArray []interface{}) ([]string, bool) {
	var outArray []string
	allConverted := true
	for _, item := range inArray {
		if istr, ok := item.(string); ok {
			outArray = append(outArray, istr)
		} else {
			allConverted = false
		}
	}
	return outArray, allConverted
}

//StringInSlice test if a string exists in a slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Compare two version strings by splitting by '.', making the slices the same length, then
// padding the strings with 0's.  The string comparison will now work against numeric
// versioning, as well as "1.5.2b1" style strings.
func CompareVersions(v1, v2 string) int {
	maxlen := 1
	s1 := strings.Split(v1, ".")
	s2 := strings.Split(v2, ".")
	for _, s := range append(s1, s2...) {
		if len(s) > maxlen {
			maxlen = len(s)
		}
	}
	if len(s1) > len(s2) {
		t := make([]string, len(s1))
		copy(t, s2)
		s2 = t
	}
	if len(s2) > len(s1) {
		t := make([]string, len(s2))
		copy(t, s1)
		s1 = t
	}
	format := fmt.Sprintf("%%0%ds", maxlen)
	for i := 0; i < len(s1); i++ {
		scomp := strings.Compare(fmt.Sprintf(format, s1[i]), fmt.Sprintf(format, s2[i]))
		if scomp != 0 {
			return scomp
		}
	}
	return 0
}
