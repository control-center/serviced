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

package volume

import (
	"errors"

	"github.com/zenoss/glog"

	"fmt"
)

// DriverInit represents a function that can initialize a driver.
type DriverInit func(root string) (Driver, error)

var (
	drivers       map[string]DriverInit
	driversByRoot map[string]Driver

	ErrDriverNotSupported   = errors.New("driver not supported")
	ErrSnapshotExists       = errors.New("snapshot exists")
	ErrSnapshotDoesNotExist = errors.New("snapshot does not exist")
)

func init() {
	drivers = make(map[string]DriverInit)
	driversByRoot = make(map[string]Driver)
}

// Driver is the basic interface to the filesystem. It is able to create,
// manage and destroy volumes. It is initialized with and operates beneath
// a given directory.
type Driver interface {
	// Root returns the filesystem root this driver acts on
	Root() string
	// GetFSType returns the string describing the driver
	GetFSType() string
	// Create creates a volume with the given name and returns it. The volume
	// must not exist already.
	Create(volumeName string) (Volume, error)
	// Remove removes an existing device. If the device doesn't exist, the
	// removal is a no-op
	Remove(volumeName string) error
	// Get returns the volume with the given name. The volume must exist.
	Get(volumeName string) (Volume, error)
	// Release releases any runtime resources associated with a volume (e.g.,
	// unmounts a device)
	Release(volumeName string) error
	// List returns the names of all volumes managed by this driver
	List() []string
	// Exists returns whether or not a volume managed by this driver exists
	// with the given name
	Exists(volumeName string) bool
	// Cleanup releases any runtime resources held by the driver itself.
	Cleanup() error
}

// Volume maps, in the end, to a directory on the filesystem available to the
// application. It can be snapshotted and rolled back to snapshots. It can be
// exported to a file and restored from a file.
type Volume interface {
	// Name returns the name of this volume
	Name() string
	// Path returns the filesystem path to this volume
	Path() string
	// Driver returns the driver managing this volume
	Driver() Driver
	// Snapshot snapshots the current state of this volume and stores it
	// using the name <label>
	Snapshot(label string) (err error)
	// SnapshotMetadataPath returns the path to the directory storing this
	// snapshot's metadata.
	SnapshotMetadataPath(label string) string
	// Snapshots lists all snapshots of this volume
	Snapshots() ([]string, error)
	// RemoveSnapshot removes the snapshot with name <label>
	RemoveSnapshot(label string) error
	// Rollback replaces the current state of the volume with that snapshotted
	// as <label>
	Rollback(label string) error
	// Export exports the snapshot stored as <label> to <filename>
	Export(label, parent, filename string) error
	// Import imports the exported snapshot at <filename> as <label>
	Import(label, filename string) error
	// Tenant returns the base tenant of this volume
	Tenant() string
}

// Register registers a driver initializer under <name> so it can be looked up
func Register(name string, driverInit DriverInit) error {
	if driverInit == nil {
		return fmt.Errorf("Can't register a nil driver initializer")
	}
	if _, dup := drivers[name]; dup {
		return fmt.Errorf("Already registered driver %s", name)
	}
	drivers[name] = driverInit
	return nil
}

// Registered returns a boolean indicating whether driver <name> has been registered.
func Registered(name string) bool {
	_, ok := drivers[name]
	return ok
}

// Unregister the driver init func <name>. If it doesn't exist, it's a no-op.
func Unregister(name string) {
	delete(drivers, name)
	// Also delete any existing drivers using this name
	for root, drv := range driversByRoot {
		if drv.GetFSType() == name {
			delete(driversByRoot, root)
		}
	}
}

// GetDriver returns a driver of type <name> initialized to <root>.
func GetDriver(name, root string) (Driver, error) {
	// First make sure it's a driver that exists
	if init, exists := drivers[name]; exists {
		// Return the same driver instance every time for a root path
		if driver, ok := driversByRoot[root]; ok {
			return driver, nil
		}
		// No instance yet, so create one
		driver, err := init(root)
		if err != nil {
			return nil, err
		}
		driversByRoot[root] = driver
		return driver, nil
	}
	return nil, ErrDriverNotSupported
}

// Mount loads, mounting if necessary, a volume under a path using a specific
// driver.
func Mount(driverName, volumeName, rootDir string) (volume Volume, err error) {
	driver, err := GetDriver(driverName, rootDir)
	if err != nil {
		return nil, err
	}
	if driver.Exists(volumeName) {
		volume, err = driver.Get(volumeName)
	} else {
		volume, err = driver.Create(volumeName)
	}
	if err != nil {
		return nil, err
	}
	return volume, nil
}

// ShutdownAll shuts down all drivers that have been initialized
func ShutdownAll() error {
	errs := []error{}
	for _, driver := range driversByRoot {
		glog.Infof("Shutting down %s driver for %s", driver.GetFSType(), driver.Root())
		if err := driver.Cleanup(); err != nil {
			glog.Errorf("Unable to clean up %s driver for %s: %s", driver.GetFSType(), driver.Root(), err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		// TODO: Something better
		return fmt.Errorf("Errors unmounting volumes: %+v", errs)
	}
	return nil
}
