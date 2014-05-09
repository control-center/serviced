// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package addressassignment

import (
	"testing"
)

func TestAddressAssignmentValidate(t *testing.T) {
	aa := AddressAssignment{}
	err := aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// valid static assignment
	aa = AddressAssignment{"id", "static", "hostid", "", "10.0.1.5", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected Error %v", err)
	}

	// valid Virtual assignment
	aa = AddressAssignment{"id", "virtual", "", "poolid", "10.0.1.5", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected Error %v", err)
	}

	//Some error cases
	// no pool id when virtual
	aa = AddressAssignment{"id", "virtual", "hostid", "", "10.0.1.5", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no host id when static
	aa = AddressAssignment{"id", "static", "", "poolid", "10.0.1.5", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no type
	aa = AddressAssignment{"id", "", "hostid", "poolid", "10.0.1.5", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no ip
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	//bad ip
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "blamIP", 100, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no port
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "10.0.1.5", 0, "serviceid", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no serviceid
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "10.0.1.5", 80, "", "endpointname"}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no endpointname
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "10.0.1.5", 80, "svcid", ""}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}
}
