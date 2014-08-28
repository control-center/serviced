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

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"
)

func (s *FacadeTest) Test_HostCRUD(t *C) {
	testid := "facadetestid"
	poolid := "pool-id"
	defer s.Facade.RemoveHost(s.CTX, testid)

	//fill host with required values
	h, err := host.Build("", poolid, []string{}...)
	h.ID = "facadetestid"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	glog.Infof("Facade test add host %v", h)
	err = s.Facade.AddHost(s.CTX, h)
	//should fail since pool doesn't exist
	if err == nil {
		t.Errorf("Expected error: %v", err)
	}

	//create pool for test
	rp := pool.New(poolid)
	rp.Realm = "default"
	if err := s.Facade.AddResourcePool(s.CTX, rp); err != nil {
		t.Fatalf("Could not add pool for test: %v", err)
	}
	defer s.Facade.RemoveResourcePool(s.CTX, poolid)

	err = s.Facade.AddHost(s.CTX, h)
	if err != nil {
		t.Errorf("Unexpected error adding host: %v", err)
	}

	//Test re-add fails
	err = s.Facade.AddHost(s.CTX, h)
	if err == nil {
		t.Errorf("Expected already exists error: %v", err)
	}

	h2, err := s.Facade.GetHost(s.CTX, testid)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if h2 == nil {
		t.Error("Unexpected nil host")

	} else if !host.HostEquals(t, h, h2) {
		t.Error("Hosts did not match")
	}

	//Test update
	h.Memory = 1024
	err = s.Facade.UpdateHost(s.CTX, h)
	h2, err = s.Facade.GetHost(s.CTX, testid)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !host.HostEquals(t, h, h2) {
		t.Error("Hosts did not match")
	}

	//test delete
	err = s.Facade.RemoveHost(s.CTX, testid)
	h2, err = s.Facade.GetHost(s.CTX, testid)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}
