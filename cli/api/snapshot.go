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
)

const ()

var ()

// Lists all snapshots on the DFS
func (a *api) GetSnapshots() ([]string, error) {
	services, err := a.GetServices()
	if err != nil {
		return nil, err
	}

	// Get only unique snapshots as defined by the tenant ID
	svcmap := NewServiceMap(services)
	var snapshots []string
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
func (a *api) GetSnapshotsByServiceID(serviceID string) ([]string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var snapshots []string
	if err := client.ListSnapshots(serviceID, &snapshots); err != nil {
		return nil, err
	}

	return snapshots, nil
}

// Snapshots a service
func (a *api) AddSnapshot(serviceID string) (string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}

	var snapshotID string
	if err := client.Snapshot(serviceID, &snapshotID); err != nil {
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

// Commit creates a snapshot and commits it as the service's image
func (a *api) Commit(dockerID string) (string, error) {
	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}

	var snapshotID string
	if err := client.Commit(dockerID, &snapshotID); err != nil {
		return "", err
	}

	return snapshotID, nil
}

// Rollback rolls back the system to the state of the given snapshot
func (a *api) Rollback(snapshotID string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.Rollback(snapshotID, &unusedInt); err != nil {
		return err
	}

	return nil
}
