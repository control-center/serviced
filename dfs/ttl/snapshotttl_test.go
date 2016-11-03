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

package ttl

import (
	"errors"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	datastoreMocks "github.com/control-center/serviced/datastore/mocks"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&SnapshotTTLTestSuite{})

type SnapshotTTLTestSuite struct {
	mockDriver *datastoreMocks.Driver
}

type TestSnapshotTTLInterface struct {
	tenantIDs  []string
	snaps      []dao.SnapshotInfo
}

func (iface *TestSnapshotTTLInterface) GetTenantIDs(unused struct {}, tenantIDs *[]string) error {
	if iface.tenantIDs == nil {
		return errors.New("error")
	}

	*tenantIDs = iface.tenantIDs
	return nil
}

func (iface *TestSnapshotTTLInterface) ListSnapshots(tenantID string, snaps *[]dao.SnapshotInfo) error {
	if iface.snaps == nil {
		return errors.New("error")
	}
	*snaps = iface.snaps
	return nil
}

func (iface *TestSnapshotTTLInterface) DeleteSnapshot(snapshotID string, _ *int) error {
	if iface.snaps == nil {
		return errors.New("error")
	}
	for i, snap := range iface.snaps {
		if snap.SnapshotID == snapshotID {
			iface.snaps = append(iface.snaps[:i], iface.snaps[i+1:]...)
			return nil
		}
	}
	return errors.New("snapshot not found")
}

func (s *SnapshotTTLTestSuite) SetUpTest(c *C) {
	s.mockDriver = &datastoreMocks.Driver{}
	datastore.Register(s.mockDriver)
}

func (s *SnapshotTTLTestSuite) TestSnapshotTTL_Purge_ServiceError(c *C) {
	iface := &TestSnapshotTTLInterface{tenantIDs: nil, snaps: []dao.SnapshotInfo{}}
	ttl := SnapshotTTL{iface}
	if _, err := ttl.Purge(100); err == nil {
		c.Errorf("Expected error!")
	}
}

func (s *SnapshotTTLTestSuite) TestSnapshotTTL_Purge_SnapshotError(c *C) {
	iface := &TestSnapshotTTLInterface{
		tenantIDs: []string{"test service id"},
		snaps: nil,
	}
	ttl := &SnapshotTTL{iface}
	if _, err := ttl.Purge(100); err == nil {
		c.Errorf("Expected error!")
	}
}

func (s *SnapshotTTLTestSuite) TestSnapshotTTL_Purge_NoSnapshot(c *C) {
	iface := &TestSnapshotTTLInterface{
		tenantIDs: []string{"test service id"},
		snaps: []dao.SnapshotInfo{},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(100); err != nil {
		c.Errorf("Unexpected error: %s", err)
	} else if age != 100 {
		c.Errorf("Expected %d; got %d", 100, age)
	}
}

func (s *SnapshotTTLTestSuite) TestSnapshotTTL_Purge_NewTTL(c *C) {
	snapTime := time.Now().UTC().Add(-5 * time.Second)
	iface := &TestSnapshotTTLInterface{
		tenantIDs: []string{"test service id"},
		snaps: []dao.SnapshotInfo{
			{SnapshotID: "snapshottag_" + snapTime.Format(timeFormat), Created: snapTime},
		},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(time.Minute); err != nil {
		c.Errorf("Unexpected error: %s", err)
	} else if age >= time.Minute {
		c.Errorf("Expected age less than %d", time.Minute)
	}
}

func (s *SnapshotTTLTestSuite) TestSnapshotTTL_Purge_Delete(c *C) {
	snapTime := time.Now().UTC().Add(-5 * time.Minute)
	iface := &TestSnapshotTTLInterface{
		tenantIDs: []string{"test service id"},
		snaps: []dao.SnapshotInfo{
			{SnapshotID: "snapshottag_" + snapTime.Format(timeFormat), Created: snapTime},
		},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(time.Minute); err != nil {
		c.Errorf("Unexpected error: %s", err)
	} else if age != time.Minute {
		c.Errorf("Expected %d; got %d", time.Minute, age)
	}

	if len(iface.snaps) > 0 {
		c.Errorf("Snaps should have been deleted")
	}
}

func (s *SnapshotTTLTestSuite) TestSnapshotTTL_Purge_DontDeleteTaggedSnap(c *C) {
	timeCreated1 := time.Now().UTC().Add(-5 * time.Minute)
	timeCreated2 := time.Now().UTC().Add(-6 * time.Minute)

	snapToPurge := dao.SnapshotInfo{
		SnapshotID: "snapshottag_" + timeCreated1.Format(timeFormat),
		Created:    timeCreated1,
	}

	snapToSave := dao.SnapshotInfo{
		SnapshotID: "snapshottag_" + timeCreated2.Format(timeFormat),
		Created:    timeCreated2,
		Tags:       []string{"some tag"},
	}

	iface := &TestSnapshotTTLInterface{
		tenantIDs: []string{"test service id"},
		snaps: []dao.SnapshotInfo{snapToPurge, snapToSave},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(time.Minute); err != nil {
		c.Errorf("Unexpected error: %s", err)
	} else if age != time.Minute {
		c.Errorf("Expected %d; got %d", time.Minute, age)
	}

	if len(iface.snaps) > 1 {
		c.Errorf("1 Snap should have been deleted")
	}

	if len(iface.snaps) < 1 {
		c.Errorf("Only 1 Snap should have been deleted")
	}

	if iface.snaps[0].SnapshotID != snapToSave.SnapshotID {
		c.Errorf("Wrong snapshot deleted")
	}

	if len(iface.snaps[0].Tags) < 1 {
		c.Errorf("Tags missing from remaning snapshot")
	}
}
