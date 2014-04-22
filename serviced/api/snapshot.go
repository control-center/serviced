package api

import (
	"fmt"
)

const ()

var ()

// ListSnapshots lists all snapshots on the DFS
func (a *api) ListSnapshots() ([]string, error) {
	return nil, nil
}

// ListSnapshotsByServiceID lists all snapshots for a given service
func (a *api) ListSnapshotsByServiceID(serviceID string) ([]string, error) {
	client, err := a.connect()
	if err != nil {
		return nil, err
	}

	var snapshots []string
	if err := client.Snapshots(serviceID, &snapshots); err != nil {
		return nil, fmt.Errorf("could not get snapshots: %s", err)
	}

	return snapshots, nil
}

// AddSnapshot takes a snapshot for a service
func (a *api) AddSnapshot(serviceID string) (string, error) {
	client, err := a.connect()
	if err != nil {
		return "", err
	}

	var snapshotID string
	if err := client.Snapshot(serviceID, &snapshotID); err != nil {
		return "", fmt.Errorf("could not take snapshot for service: %s", err)
	}

	return snapshotID, nil
}

// RemoveSnapshot deletes a snapshot
func (a *api) RemoveSnapshot(snapshotID string) error {
	client, err := a.connect()
	if err != nil {
		return err
	}

	if err := client.DeleteSnapshot(snapshotID, &unusedInt); err != nil {
		return fmt.Errorf("could not remove snapshot: %s", err)
	}

	return nil
}

// Commit creates a snapshot and commits it as the service's image
func (a *api) Commit(dockerID string) (string, error) {
	client, err := a.connect()
	if err != nil {
		return "", err
	}

	var snapshotID string
	if err := client.Commit(dockerID, &snapshotID); err != nil {
		return "", fmt.Errorf("could not commit snapshot: %s", err)
	}

	return snapshotID, nil
}

// Rollback rolls back the system to the state of the given snapshot
func (a *api) Rollback(snapshotID string) error {
	client, err := a.connect()
	if err != nil {
		return err
	}

	if err := client.Rollback(snapshotID, &unusedInt); err != nil {
		return fmt.Errorf("could not rollback snapshot: %s", err)
	}

	return nil
}
