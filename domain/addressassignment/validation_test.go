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

package addressassignment

import (
	"testing"

	"github.com/control-center/serviced/datastore"
)

var version datastore.VersionedEntity

func TestAddressAssignmentValidate(t *testing.T) {
	aa := AddressAssignment{}
	err := aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// valid static assignment
	aa = AddressAssignment{"id", "static", "hostid", "", "10.0.1.5", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected Error %v", err)
	}

	// valid Virtual assignment
	aa = AddressAssignment{"id", "virtual", "", "poolid", "10.0.1.5", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err != nil {
		t.Errorf("Unexpected Error %v", err)
	}

	//Some error cases
	// no pool id when virtual
	aa = AddressAssignment{"id", "virtual", "hostid", "", "10.0.1.5", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no host id when static
	aa = AddressAssignment{"id", "static", "", "poolid", "10.0.1.5", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no type
	aa = AddressAssignment{"id", "", "hostid", "poolid", "10.0.1.5", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no ip
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	//bad ip
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "blamIP", 100, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no port
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "10.0.1.5", 0, "serviceid", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no serviceid
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "10.0.1.5", 80, "", "endpointname", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}

	// no endpointname
	aa = AddressAssignment{"id", "static", "hostid", "poolid", "10.0.1.5", 80, "svcid", "", version}
	err = aa.ValidEntity()
	if err == nil {
		t.Error("Expected error")
	}
}
