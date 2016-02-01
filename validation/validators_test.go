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

package validation

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ValidationSuite struct{}

var _ = Suite(&ValidationSuite{})

func (vs *ValidationSuite) Test_IsSubnet16(c *C) {

	subnetsValid := []string{
		"9.0",     // non-private subnet
		"10.0",    // start of private subnet 10.0 - 10.255
		"10.3",    //   private subnet
		"10.20",   //   private subnet
		"10.255",  // end of private subnet
		"11.0",    // non-private subnet
		"172.15",  // non-private subnet
		"172.16",  // start of private subnet 172.16 - 172.31
		"172.31",  // end of private subnet
		"172.32",  // non-private subnet
		"192.167", // non-private subnet
		"192.168", // private subnet 192.168
		"192.169", // non-private subnet
	}

	for _, subnet := range subnetsValid {
		if err := IsSubnet16(subnet); err != nil {
			c.Fatalf("Unexpected error validating valid subnet %s: %v", subnet, err)
		}
	}

	subnetsInvalid := []string{
		"10",
		"10.10.10",
		"10.10.10.10",
		"10.300",
		"10.y",
		"x.y",
	}

	for _, subnet := range subnetsInvalid {
		if err := IsSubnet16(subnet); err == nil {
			c.Fatalf("Unexpected non-error validating invalid subnet %s: %v", subnet, err)
		}
	}
}

func (vs *ValidationSuite) Test_IsValidHostID(c *C) {
	hostIDsValid := []string{
		"570a276e", // 10.87.110.39
		"6f0ae003", // 10.111.3.224
	}

	for _, hostID := range hostIDsValid {
		if err := ValidHostID(hostID); err != nil {
			c.Fatalf("Unexpected error validating valid hostid %s: %v", hostID, err)
		}
	}

	hostIDsInvalid := []string{
		"",
		"0",
		"00000000",
	}

	for _, hostID := range hostIDsInvalid {
		if err := ValidHostID(hostID); err == nil {
			c.Fatalf("Unexpected non-error validating invalid hostid %s: %v", hostID, err)
		}
	}
}

func (vs *ValidationSuite) Test_ValidUIAddress(c *C) {
	addrValid := []string{
		":123",
		"abc:123",
	}

	for _, addr := range addrValid {
		if err := ValidUIAddress(addr); err != nil {
			c.Fatalf("Unexpected error validating valid UI address: %s: %v", addr, err)
		}
	}

	addrInvalid := []string{
		"abc:",
		":",
		":0",
		":65536",
		":abc",
	}

	for _, addr := range addrInvalid {
		if err := ValidUIAddress(addr); err == nil {
			c.Fatalf("Unexpected lack of err validating invalid UI address: %s", addr)
		}
	}
}

func (vs *ValidationSuite) Test_ExcludeString(c *C) {
	err := ExcludeChars("field", "team", "i")
	c.Assert(err, IsNil)
	err = ExcludeChars("field", "failure", "u & i")
	c.Assert(err, NotNil)
	err = ExcludeChars("field", "foo", "")
	c.Assert(err, IsNil)
	err = ExcludeChars("field", "", "")
	c.Assert(err, IsNil)
}
