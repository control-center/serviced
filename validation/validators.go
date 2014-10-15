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

package validation

import (
	"fmt"
	"net"
	"strings"
)

//NotEmpty check to see if the value is not an empty string or a string with just whitespace characters, returns an
// error if empty. FieldName is used to create a meaningful error
func NotEmpty(fieldName string, value string) error {
	if strings.TrimSpace(value) == "" {
		return NewViolation(fmt.Sprintf("empty string for %v", fieldName))
	}
	return nil
}

//IsIP checks to see if the value is a valid IP address. Returns an error if not valid
func IsIP(value string) error {
	if nil == net.ParseIP(value) {
		return NewViolation(fmt.Sprintf("invalid IP Address %s", value))
	}
	return nil
}

//IsSubnet16 checks to see if the value is a valid /16 subnet.  Returns an error if not valid
func IsSubnet16(value string) error {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return NewViolation(fmt.Sprintf("invalid /16 subnet %s", value))
	}

	ip := fmt.Sprintf("%s.1.1", value)
	if nil == net.ParseIP(ip) {
		return NewViolation(fmt.Sprintf("invalid /16 subnet %s", value))
	}
	return nil
}

//StringsEqual checks to see that strings are equal, optional msg to use instead of default
func StringsEqual(expected string, other string, errMsg string) error {
	if expected != other {
		if errMsg == "" {
			errMsg = fmt.Sprintf("expected %s found %s", expected, other)
		}
		return NewViolation(errMsg)
	}
	return nil
}

//StringsEqual checks to see that strings are equal, optional msg to use instead of default
func StringIn(check string, others ...string) error {
	set := make(map[string]struct{}, len(others))
	for _, val := range others {
		set[val] = struct{}{}
	}
	if _, ok := set[check]; !ok {
		return NewViolation(fmt.Sprintf("string %v not in %v", check, others))
	}
	return nil
}

func ValidPort(port int) error {
	if port < 1 || port > 65535 {
		return NewViolation(fmt.Sprintf("not in valid port range: %v", port))
	}
	return nil
}

func IntIn(check int, others ...int) error {
	set := make(map[int]struct{}, len(others))
	for _, val := range others {
		set[val] = struct{}{}
	}
	if _, ok := set[check]; !ok {
		return NewViolation(fmt.Sprintf("int %v not in %v", check, others))
	}
	return nil
}
