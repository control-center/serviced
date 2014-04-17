package api

import ()

const ()

var ()

// ListSnapshots lists all snapshots on the DFS
func (a *api) ListSnapshots() ([]string, error) {
	return nil, nil
}

// ListSnapshotsByServiceID lists all snapshots for a given service
func (a *api) ListSnapshotsByServiceID(serviceID string) ([]string, error) {
	return nil, nil
}

// AddSnapshot takes a snapshot for a service
func (a *api) AddSnapshot(serviceID string) (string, error) {
	return "", nil
}

// RemoveSnapshot deletes a snapshot
func (a *api) RemoveSnapshot(snapshotID string) error {
	return nil
}

// Commit creates a snapshot and commits it as the service's image
func (a *api) Commit(dockerID string) (string, error) {
	return "", nil
}

// Rollback rolls back the system to the state of the given snapshot
func (a *api) Rollback(snapshotID string) error {
	return nil
}
