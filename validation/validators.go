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
	"strconv"
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

// ExcludeChars makes sure there characters in a field are valid
func ExcludeChars(fieldName, value, chars string) error {
	if strings.ContainsAny(value, chars) {
		return NewViolation(fmt.Sprintf("invalid chars for %s", fieldName))
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

func ValidUIAddress(addr string) error {
	if strings.Index(addr, ":") == -1 {
		return NewViolation(fmt.Sprintf("not a valid ui address: %s", addr))
	}
	s := strings.Split(addr, ":")
	if len(s) != 2 {
		return NewViolation(fmt.Sprintf("not a valid ui address: %s", addr))
	}
	i, err := strconv.Atoi(s[1])
	if err != nil {
		return NewViolation(fmt.Sprintf("not a valid ui address: %s", addr))
	}
	return ValidPort(i)
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

func ValidHostID(hostID string) error {
	result, err := strconv.ParseUint(hostID, 16, 0)
	if err != nil {
		return NewViolation(fmt.Sprintf("unable to convert hostid: %v to uint", hostID))
	}
	if result <= 0 {
		return NewViolation(fmt.Sprintf("not valid hostid: %v", hostID))
	}
	return nil
}

func ValidPoolId(poolID string) error {
	if strings.ContainsAny(poolID, ".#") {
		return NewViolation(fmt.Sprintf("not a valid poolid: %v", poolID))
	}
	return nil
}

func ValidVirtualIP(bindInterface string) error {
	// VIP names append a prefix and an index, and cannot be more than 15
	// characters in length. See zzk/virtualips.
	vipname := bindInterface + ":z" + "000"
	if len(vipname) > 15 {
		return NewViolation(fmt.Sprintf("virtual ip name too long, must be less than 16 characters: %s", vipname))
	}
	return nil
}
