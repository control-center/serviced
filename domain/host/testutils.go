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

// +build integration

package host

type errorf interface {
	Errorf(format string, args ...interface{})
}

//HostEquals used for testing host equality in tests
func HostEquals(t errorf, h1 *Host, h2 *Host) bool {
	equals := true
	if h1 != nil && h2 == nil {
		t.Errorf("%v:Cannot compare non nil h1 to nil h2", t)
		return false
	}
	if h1.ID != h2.ID {
		t.Errorf("Host name %v did not equal %v", h1.ID, h2.ID)
		equals = false
	}
	if h1.Name != h2.Name {
		t.Errorf("Host id %v did not equal %v", h1.Name, h2.Name)
		equals = false
	}
	if h1.PoolID != h2.PoolID {
		t.Errorf("Host PoolID %v did not equal %v", h1.PoolID, h2.PoolID)
		equals = false
	}
	if h1.IPAddr != h2.IPAddr {
		t.Errorf("Host IpAddr %v did not equal %v", h1.IPAddr, h2.IPAddr)
		equals = false
	}
	if h1.Cores != h2.Cores {
		t.Errorf("Host Cores %v did not equal %v", h1.Cores, h2.Cores)
		equals = false
	}
	if h1.Memory != h2.Memory {
		t.Errorf("Host Memory %v did not equal %v", h1.Memory, h2.Memory)
		equals = false
	}
	if h1.PrivateNetwork != h2.PrivateNetwork {
		t.Errorf("Host PrivateNetwork %v did not equal %v", h1.PrivateNetwork, h2.PrivateNetwork)
		equals = false
	}
	if !h1.CreatedAt.Equal(h2.CreatedAt) {
		t.Errorf("Host CreatedAt %v did not equal %v", h1.CreatedAt, h2.CreatedAt)
		equals = false
	}
	if !h1.UpdatedAt.Equal(h2.UpdatedAt) {
		t.Errorf("Host UpdatedAt %v did not equal %v", h1.UpdatedAt, h2.UpdatedAt)
		equals = false
	}

	if len(h1.IPs) != len(h2.IPs) {
		t.Errorf("Host IPs %v does not match %v", h1.IPs, h2.IPs)
		equals = false
	}
	return equals
}
