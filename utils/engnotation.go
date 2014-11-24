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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Structure used for serializing/deserializing values represented as
// strings in engineering notation (e.g., 1K, 256M, etc.)
type EngNotation struct {
	source string
	Value uint64
}


func (e *EngNotation) UnmarshalJSON(b[] byte) (err error) {
	json.Unmarshal(b, &e.source)
	e.Value, err = parseEngineeringNotation(e.source)
	return
}


func (i EngNotation) MarshalJSON() (text []byte, err error) {
	return json.Marshal(i.source)
}


func parseEngineeringNotation(in string) (uint64, error) {
	if in == "" {
		return 0, nil
	}
	var numerics = func (r rune) bool {
		return  r >= '0' && r <= '9'
	}
	var nonNumerics = func (r rune) bool {
		return  ! numerics(r)
	}
	suffix := strings.TrimLeftFunc(in, numerics)
	numeric := strings.TrimRightFunc(in, nonNumerics)
	val, err := strconv.ParseUint(numeric, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Parsing engineering notation for '%s'", in)
	}
	switch(suffix) {
	case "K", "k" :
		val *= (1 << 10)
	case "M", "m":
		val *= (1 << 20)
	case "G", "g":
		val *= (1 << 30)
	case "T", "t":
		val *= (1 << 40)
	case "" :
		break
	default:
		return 0, fmt.Errorf("Parsing engineering notation for '%s'", in)
	}
	return val, nil
}
