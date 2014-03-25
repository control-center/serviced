// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"testing"
)

func HostEquals(t *testing.T, h1 *Host, h2 *Host) {
	if h1 != nil && h2 == nil {
		t.Error("Cannot compare non nil h1 to nil h2")
		return
	}
	if h1.Id != h2.Id {
		t.Errorf("Host name %v did not equal %v", h1.Id, h2.Id)
	}
	if h1.Name != h2.Name {
		t.Errorf("Host id %v did not equal %v", h1.Name, h2.Name)
	}
	if h1.PoolId != h2.PoolId {
		t.Errorf("Host PoolId %v did not equal %v", h1.PoolId, h2.PoolId)
	}
	if h1.IpAddr != h2.IpAddr {
		t.Errorf("Host IpAddr %v did not equal %v", h1.IpAddr, h2.IpAddr)
	}
	if h1.Cores != h2.Cores {
		t.Errorf("Host Cores %v did not equal %v", h1.Cores, h2.Cores)
	}
	if h1.Memory != h2.Memory {
		t.Errorf("Host Memory %v did not equal %v", h1.Memory, h2.Memory)
	}
	if h1.PrivateNetwork != h2.PrivateNetwork {
		t.Errorf("Host PrivateNetwork %v did not equal %v", h1.PrivateNetwork, h2.PrivateNetwork)
	}
	if !h1.CreatedAt.Equal(h2.CreatedAt) {
		t.Errorf("Host CreatedAt %v did not equal %v", h1.CreatedAt, h2.CreatedAt)
	}
	if !h1.UpdatedAt.Equal(h2.UpdatedAt) {
		t.Errorf("Host UpdatedAt %v did not equal %v", h1.UpdatedAt, h2.UpdatedAt)
	}

}
