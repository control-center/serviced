// Copyright 2015 The Serviced Authors.
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
	"strconv"
	"strings"
)

type TestConfigReader map[string]string

func (r TestConfigReader) StringVal(name string, dflt string) string {
	if val, _ := r[name]; val != "" {
		return val
	} else {
		return dflt
	}
}

func (r TestConfigReader) StringSlice(name string, dflt []string) []string {
	if val, _ := r[name]; val != "" {
		return strings.Split(val, ",")
	}
	return dflt
}

func (r TestConfigReader) StringNumberedList(name string, dflt []string) []string {
	values := ""
	i := 0
	for {
		if strval, ok := r[name+"_"+strconv.Itoa(i)]; ok {
			if values == "" {
				values = strval
			} else {
				values += "," + strval
			}
		} else {
			if values == "" {
				r[name] = strings.Join(dflt, ",")
				return dflt
			} else {
				r[name] = values
				return strings.Split(values, ",")
			}
		}

		i += 1
	}
}

func (r TestConfigReader) IntVal(name string, dflt int) int {
	if val, _ := r[name]; val != "" {
		if intval, err := strconv.Atoi(val); err != nil {
			return intval
		}
	}
	return dflt
}

func (r TestConfigReader) BoolVal(name string, dflt bool) bool {
	if val, _ := r[name]; val != "" {
		val = strings.ToLower(val)

		trues := []string{"1", "true", "t", "yes"}
		for _, t := range trues {
			if val == t {
				return true
			}
		}

		falses := []string{"0", "false", "f", "no"}
		for _, f := range falses {
			if val == f {
				return false
			}
		}
	}
	return dflt
}

func (p TestConfigReader) GetConfigValues() map[string]ConfigValue {
	return map[string]ConfigValue{}
}

func (r TestConfigReader) Float64Val(name string, dflt float64) float64 {
	if val, _ := r[name]; val != "" {
		if floatval, err := strconv.ParseFloat(val, 64); err != nil {
			return floatval
		}
	}
	return dflt
}
