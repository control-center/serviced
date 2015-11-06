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

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

type TestSnapshotTTLInterface struct {
	svcs  []service.Service
	snaps []dao.SnapshotInfo
}

func (iface *TestSnapshotTTLInterface) GetServices(req dao.ServiceRequest, svcs *[]service.Service) error {
	if iface.svcs == nil {
		return errors.New("error")
	}

	*svcs = iface.svcs
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

func TestSnapshotTTL_Purge_ServiceError(t *testing.T) {
	iface := &TestSnapshotTTLInterface{svcs: nil, snaps: []dao.SnapshotInfo{}}
	ttl := SnapshotTTL{iface}
	if _, err := ttl.Purge(100); err == nil {
		t.Errorf("Expected error!")
	}
}

func TestSnapshotTTL_Purge_NoService(t *testing.T) {
	iface := &TestSnapshotTTLInterface{svcs: []service.Service{}, snaps: nil}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(100); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if age != 100 {
		t.Errorf("Expected %d; got %d", 100, age)
	}

	iface = &TestSnapshotTTLInterface{
		svcs: []service.Service{
			{
				ParentServiceID: "test parent id",
				ID:              "test service id",
			},
		},
		snaps: nil,
	}
	ttl = &SnapshotTTL{iface}
	if age, err := ttl.Purge(100); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if age != 100 {
		t.Errorf("Expected %d; got %d", 100, age)
	}
}

func TestSnapshotTTL_Purge_SnapshotError(t *testing.T) {
	iface := &TestSnapshotTTLInterface{
		svcs: []service.Service{
			{
				ParentServiceID: "",
				ID:              "test service id",
			},
		},
		snaps: nil,
	}
	ttl := &SnapshotTTL{iface}
	if _, err := ttl.Purge(100); err == nil {
		glog.Errorf("Expected error!")
	}
}

func TestSnapshotTTL_Purge_NoSnapshot(t *testing.T) {
	iface := &TestSnapshotTTLInterface{
		svcs: []service.Service{
			{
				ParentServiceID: "",
				ID:              "test service id",
			},
		},
		snaps: []dao.SnapshotInfo{},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(100); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if age != 100 {
		t.Errorf("Expected %d; got %d", 100, age)
	}
}

func TestSnapshotTTL_Purge_NewTTL(t *testing.T) {
	snapTime := time.Now().UTC().Add(-5 * time.Second)
	iface := &TestSnapshotTTLInterface{
		svcs: []service.Service{
			{
				ParentServiceID: "",
				ID:              "test service id",
			},
		},
		snaps: []dao.SnapshotInfo{
			{SnapshotID: "snapshottag_" + snapTime.Format(timeFormat), Created: snapTime},
		},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(time.Minute); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if age >= time.Minute {
		t.Errorf("Expected age less than %d", time.Minute)
	}
}

func TestSnapshotTTL_Purge_Delete(t *testing.T) {
	snapTime := time.Now().UTC().Add(-5 * time.Minute)
	iface := &TestSnapshotTTLInterface{
		svcs: []service.Service{
			{
				ParentServiceID: "",
				ID:              "test service id",
			},
		},
		snaps: []dao.SnapshotInfo{
			{SnapshotID: "snapshottag_" + snapTime.Format(timeFormat), Created: snapTime},
		},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(time.Minute); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if age != time.Minute {
		t.Errorf("Expected %d; got %d", time.Minute, age)
	}

	if len(iface.snaps) > 0 {
		t.Errorf("Snaps should have been deleted")
	}
}

func TestSnapshotTTL_Purge_DontDeleteTaggedSnap(t *testing.T) {
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
		svcs: []service.Service{
			{
				ParentServiceID: "",
				ID:              "test service id",
			},
		},
		snaps: []dao.SnapshotInfo{snapToPurge, snapToSave},
	}
	ttl := &SnapshotTTL{iface}
	if age, err := ttl.Purge(time.Minute); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if age != time.Minute {
		t.Errorf("Expected %d; got %d", time.Minute, age)
	}

	if len(iface.snaps) > 1 {
		t.Errorf("1 Snap should have been deleted")
	}

	if len(iface.snaps) < 1 {
		t.Errorf("Only 1 Snap should have been deleted")
	}

	if iface.snaps[0].SnapshotID != snapToSave.SnapshotID {
		t.Errorf("Wrong snapshot deleted")
	}

	if len(iface.snaps[0].Tags) < 1 {
		t.Errorf("Tags missing from remaning snapshot")
	}
}
