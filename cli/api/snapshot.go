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

package api

import (
	"fmt"

	"github.com/control-center/serviced/dao"
)

type SnapshotConfig struct {
	ServiceID string
	Message   string
	Tags      []string
	DockerID  string
}

// Lists all snapshots on the DFS
func (a *api) GetSnapshots() ([]dao.SnapshotInfo, error) {
	services, err := a.GetServices()
	if err != nil {
		return nil, err
	}

	// Get only unique snapshots as defined by the tenant ID
	svcmap := NewServiceMap(services)
	var snapshots []dao.SnapshotInfo
	for _, s := range svcmap.Tree()[""] {
		ss, err := a.GetSnapshotsByServiceID(s)
		if err != nil {
			return nil, fmt.Errorf("error trying to retrieve snapshots for service %s: %s", s, err)
		}
		snapshots = append(snapshots, ss...)
	}

	return snapshots, nil
}

// Lists all snapshots for a given service
func (a *api) GetSnapshotsByServiceID(serviceID string) ([]dao.SnapshotInfo, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var snapshots []dao.SnapshotInfo
	if err := client.ListSnapshots(serviceID, &snapshots); err != nil {
		return nil, err
	}

	return snapshots, nil
}

// Snapshots a service
func (a *api) AddSnapshot(cfg SnapshotConfig) (string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}
	req := dao.SnapshotRequest{
		ServiceID:   cfg.ServiceID,
		Message:     cfg.Message,
		Tags:        cfg.Tags,
		ContainerID: cfg.DockerID,
	}
	var snapshotID string
	if err := client.Snapshot(req, &snapshotID); err != nil {
		return "", err
	}

	return snapshotID, nil
}

// Deletes a snapshot
func (a *api) RemoveSnapshot(snapshotID string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.DeleteSnapshot(snapshotID, &unusedInt); err != nil {
		return err
	}

	return nil
}

// Rollback rolls back the system to the state of the given snapshot
func (a *api) Rollback(snapshotID string, forceRestart bool) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.Rollback(dao.RollbackRequest{snapshotID, forceRestart}, &unusedInt); err != nil {
		return err
	}

	return nil
}

// TagSnapshot tags an existing snapshot with 1 or more strings
func (a *api) TagSnapshot(snapshotID string, tagNames []string) ([]string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var newTagList []string
	if err := client.TagSnapshot(dao.TagSnapshotRequest{snapshotID, tagNames}, &newTagList); err != nil {
		return newTagList, err
	}

	return newTagList, nil
}

// RemoveSnapshotTags removes specific tags from an existing snapshot
func (a *api) RemoveSnapshotTags(snapshotID string, tagNames []string) ([]string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var newTagList []string
	if err := client.RemoveSnapshotTags(dao.TagSnapshotRequest{snapshotID, tagNames}, &newTagList); err != nil {
		return newTagList, err
	}

	return newTagList, nil
}

// RemoveAllSnapshotTags removes all tags from an existing snapshot
func (a *api) RemoveAllSnapshotTags(snapshotID string) (error) {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.RemoveAllSnapshotTags(snapshotID, &unusedInt); err != nil {
		return err
	}

	return nil
}
