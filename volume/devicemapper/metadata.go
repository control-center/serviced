// +build linux

package devicemapper

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

var (
	ErrInvalidMetadata = errors.New("invalid metadata")
)

type snapshotMetadata struct {
	CurrentDevice string            `json:"CurrentDevice"`
	Snapshots     map[string]string `json:"Snapshots"`
}

type SnapshotMetadata struct {
	path string
	snapshotMetadata
	sync.Mutex
}

func NewMetadata(path string) (*SnapshotMetadata, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	metadata := &SnapshotMetadata{
		path: path,
		snapshotMetadata: snapshotMetadata{
			Snapshots: make(map[string]string),
		},
	}
	metadata.Lock()
	defer metadata.Unlock()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		metadata.save()
	} else {
		if err := metadata.load(); err != nil {
			return nil, err
		}
	}
	return metadata, nil
}

// load the metadata from file.
func (m *SnapshotMetadata) load() error {
	// This is not safe for concurrent access.
	// Exported methods will deal with locking.
	jsonData, err := ioutil.ReadFile(m.path)
	if jsonData == nil || os.IsNotExist(err) {
		m.save()
	} else {
		if err := json.Unmarshal(jsonData, &m.snapshotMetadata); err != nil {
			return err
		}
	}
	return nil
}

// save metadata to file
func (m *SnapshotMetadata) save() error {
	// This is not safe for concurrent access.
	// Exported methods will deal with locking.
	jsonData, err := json.Marshal(m.snapshotMetadata)
	if err != nil {
		glog.Errorf("Error encoding metadata to json: %s", err)
		return ErrInvalidMetadata
	}
	return ioutil.WriteFile(m.path, jsonData, 0644)
}

// remove metadata file
func (m *SnapshotMetadata) remove() error {
	return os.RemoveAll(m.path)
}

// Get the current device for this volume
func (m *SnapshotMetadata) CurrentDevice() string {
	return m.snapshotMetadata.CurrentDevice
}

func (m *SnapshotMetadata) SetCurrentDevice(device string) error {
	m.Lock()
	defer m.Unlock()
	m.snapshotMetadata.CurrentDevice = device
	return m.save()
}

func (m *SnapshotMetadata) AddSnapshot(snapshot, device string) error {
	m.Lock()
	defer m.Unlock()
	m.snapshotMetadata.Snapshots[snapshot] = device
	return m.save()
}

// Remove snapshot info from the metadata. If the snapshot doesn't exist, it's a no-op.
func (m *SnapshotMetadata) RemoveSnapshot(snapshot string) error {
	m.Lock()
	defer m.Unlock()
	delete(m.snapshotMetadata.Snapshots, snapshot)
	return m.save()
}

func (m *SnapshotMetadata) ListSnapshots() (snaps []string) {
	for snap, _ := range m.snapshotMetadata.Snapshots {
		snaps = append(snaps, snap)
	}
	return snaps
}

func (m *SnapshotMetadata) ListDevices() (devices []string) {
	// Add the current device first so any device-based operations can act on it before any snapshots
	devices = append(devices, m.CurrentDevice())
	for _, device := range m.snapshotMetadata.Snapshots {
		devices = append(devices, device)
	}
	return devices
}

func (m *SnapshotMetadata) LookupSnapshotDevice(snapshot string) (string, error) {
	m.Lock()
	defer m.Unlock()
	device, ok := m.snapshotMetadata.Snapshots[snapshot]
	if !ok {
		return "", volume.ErrSnapshotDoesNotExist
	}
	return device, nil
}

func (m *SnapshotMetadata) SnapshotExists(snapshot string) bool {
	_, ok := m.snapshotMetadata.Snapshots[snapshot]
	return ok
}
