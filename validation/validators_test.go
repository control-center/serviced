// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

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
		"10.3",
		"10.5",
		"10.20",
		"10.255",
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
		"10.y",
		"x.y",
	}

	for _, subnet := range subnetsInvalid {
		if err := IsSubnet16(subnet); err == nil {
			c.Fatalf("Unexpected non-error validating invalid subnet %s: %v", subnet, err)
		}
	}
}
