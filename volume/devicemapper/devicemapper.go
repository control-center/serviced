package devicemapper

import (
	"github.com/control-center/serviced/volume"
)

const (
	// DriverName is the name of this devicemapper driver implementation
	DriverName = "devicemapper"
)

func init() {
	volume.Register(DriverName, Init)
}

type DeviceMapperDriver struct {
	root string
}

type DeviceMapperVolume struct {
	name   string
	path   string
	tenant string
	driver volume.Driver
}

func Init(root string) (volume.Driver, error) {
	driver := &DeviceMapperDriver{
		root: root,
	}
	return driver, nil
}

// Root implements volume.Driver.Root
func (d *DeviceMapperDriver) Root() string {
	return d.root
}

// Create implements volume.Driver.Create
func (d *DeviceMapperDriver) Create(volumeName string) (volume.Volume, error) {
	return d.Get(volumeName)
}

// Get implements volume.Driver.Get
func (d *DeviceMapperDriver) Get(volumeName string) (volume.Volume, error) {
	return &DeviceMapperVolume{}, nil
}

// List implements volume.Driver.List
func (d *DeviceMapperDriver) List() []string {
	return nil
}

// Exists implements volume.Driver.Exists
func (d *DeviceMapperDriver) Exists(volumeName string) bool {
	// TODO: Implement
	return false
}

// Cleanup implements volume.Driver.Cleanup
func (d *DeviceMapperDriver) Cleanup() error {
	// TODO: Implement
	return nil
}

// Release implements volume.Driver.Release
func (d *DeviceMapperDriver) Release(volumeName string) error {
	// TODO: Implement
	return nil
}

// Remove implements volume.Driver.Remove
func (d *DeviceMapperDriver) Remove(volumeName string) error {
	return nil
}

// Name implements volume.Volume.Name
func (v *DeviceMapperVolume) Name() string {
	return v.name
}

// Path implements volume.Volume.Path
func (v *DeviceMapperVolume) Path() string {
	return v.path
}

// Driver implements volume.Volume.Driver
func (v *DeviceMapperVolume) Driver() volume.Driver {
	return v.driver
}

// Tenant implements volume.Volume.Tenant
func (v *DeviceMapperVolume) Tenant() string {
	return v.tenant
}

// SnapshotMetadataPath implements volume.Volume.SnapshotMetadataPath
func (v *DeviceMapperVolume) SnapshotMetadataPath(label string) string {
	// TODO: Implement
	return "/tmp"
}

// Snapshot implements volume.Volume.Snapshot
func (v *DeviceMapperVolume) Snapshot(label string) error {
	// TODO: Implement
	return nil
}

// Snapshots implements volume.Volume.Snapshots
func (v *DeviceMapperVolume) Snapshots() ([]string, error) {
	// TODO: Implement
	return nil, nil
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *DeviceMapperVolume) RemoveSnapshot(label string) error {
	// TODO: Implement
	return nil
}

// Rollback implements volume.Volume.Rollback
func (v *DeviceMapperVolume) Rollback(label string) error {
	// TODO: Implement
	return nil
}

// Export implements volume.Volume.Export
func (v *DeviceMapperVolume) Export(label, parent, outfile string) error {
	// TODO: Implement
	return nil
}

// Import implements volume.Volume.Import
func (v *DeviceMapperVolume) Import(label, infile string) error {
	// TODO: Implement
	return nil
}