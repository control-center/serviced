// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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

	var snapshots []string
	for _, s := range services {
		ss, err := a.GetSnapshotsByServiceID(s.ID)
		if err != nil {
			return nil, fmt.Errorf("error trying to retrieve snapshots for service %s: %s", s.ID, err)
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
	if err := client.Snapshots(serviceID, &snapshots); err != nil {
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
