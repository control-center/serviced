package devicemapper

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/docker/docker/daemon/graphdriver/devmapper"
	"github.com/zenoss/glog"
)

const (
	// DriverName is the name of this devicemapper driver implementation
	DriverName = "devicemapper"
)

func init() {
	volume.Register(DriverName, Init)
}

type DeviceMapperDriver struct {
	root      string
	DeviceSet *devmapper.DeviceSet
}

type DeviceMapperVolume struct {
	name     string
	path     string
	tenant   string
	driver   *DeviceMapperDriver
	Metadata *SnapshotMetadata
	sync.Mutex
}

func Init(root string) (volume.Driver, error) {
	driver := &DeviceMapperDriver{
		root: root,
	}
	driver.ensureInitialized()
	return driver, nil
}

// Root implements volume.Driver.Root
func (d *DeviceMapperDriver) Root() string {
	return d.root
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

// Create implements volume.Driver.Create
func (d *DeviceMapperDriver) Create(volumeName string) (volume.Volume, error) {
	if d.Exists(volumeName) {
		return nil, fmt.Errorf("Volume exists already")
	}
	// Create a new device
	volumeDevice, err := utils.NewUUID62()
	if err != nil {
		return nil, err
	}
	if err := d.DeviceSet.AddDevice(volumeDevice, ""); err != nil {
		return nil, err
	}
	// Create the mount target directory if it doesn't exist
	mountpoint := d.volumeDir(volumeName)
	if err := os.MkdirAll(mountpoint, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	md := d.MetadataDir()
	if err := os.MkdirAll(md, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	vol, err := d.newVolume(volumeName)
	if err != nil {
		return nil, err
	}
	if err := vol.Metadata.SetCurrentDevice(volumeDevice); err != nil {
		return nil, err
	}
	glog.Infof("Mounting device to %s", mountpoint)
	if err := d.DeviceSet.MountDevice(volumeDevice, mountpoint, volumeName); err != nil {
		return nil, err
	}
	return vol, nil
}

func (d *DeviceMapperDriver) newVolume(volumeName string) (*DeviceMapperVolume, error) {
	tenant := getTenant(volumeName)
	metadata, err := NewMetadata(d.MetadataPath(tenant))
	if err != nil {
		return nil, err
	}
	vol := &DeviceMapperVolume{
		name:     volumeName,
		path:     d.volumeDir(volumeName),
		tenant:   tenant,
		driver:   d,
		Metadata: metadata,
	}
	return vol, nil
}

// Get implements volume.Driver.Get
func (d *DeviceMapperDriver) Get(volumeName string) (volume.Volume, error) {
	return d.newVolume(volumeName)
}

// List implements volume.Driver.List
func (d *DeviceMapperDriver) List() (result []string) {
	md := d.MetadataDir()
	if files, err := ioutil.ReadDir(md); err != nil {
		glog.Errorf("Error trying to read from metadata directory: %s", md)
		return
	} else {
		for _, fi := range files {
			if fi.IsDir() {
				result = append(result, fi.Name())
			}
		}
	}
	return
}

// Exists implements volume.Driver.Exists
func (d *DeviceMapperDriver) Exists(volumeName string) bool {
	for _, vol := range d.List() {
		if vol == volumeName {
			return true
		}
	}
	return false
}

// Cleanup implements volume.Driver.Cleanup
func (d *DeviceMapperDriver) Cleanup() error {
	if d.DeviceSet == nil {
		return nil
	}
	return d.DeviceSet.Shutdown()
}

// Release implements volume.Driver.Release
func (d *DeviceMapperDriver) Release(volumeName string) error {
	tenant := getTenant(volumeName)
	metadata, err := NewMetadata(d.MetadataPath(tenant))
	if err != nil {
		return err
	}
	device := metadata.CurrentDevice()
	return d.DeviceSet.UnmountDevice(device)
}

// Remove implements volume.Driver.Remove
func (d *DeviceMapperDriver) Remove(volumeName string) error {
	return nil
}

func (d *DeviceMapperDriver) volumeDir(volumeName string) string {
	return filepath.Join(d.root, volumeName)
}

// poolDir returns the path under which all metadata and images will be stored
func (d *DeviceMapperDriver) poolDir() string {
	return filepath.Join(d.root, ".devicemapper")
}

// snapshotDir returns the path under which volume metadata will be stored
func (d *DeviceMapperDriver) MetadataDir() string {
	return filepath.Join(d.poolDir(), "volumes")
}

// metadataPath returns the path of the json metadata file
func (d *DeviceMapperDriver) MetadataPath(tenant string) string {
	return filepath.Join(d.MetadataDir(), tenant, "metadata.json")
}

// ensureInitialized makes sure this driver's root has been set up properly
// for devicemapper. It is idempotent.
func (d *DeviceMapperDriver) ensureInitialized() error {
	poolPath := d.poolDir()
	if err := os.MkdirAll(poolPath, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	if d.DeviceSet == nil {
		deviceSet, err := devmapper.NewDeviceSet(poolPath, true, []string{})
		if err != nil {
			return err
		}
		d.DeviceSet = deviceSet
	}
	if err := os.MkdirAll(d.MetadataDir(), 0755); err != nil && !os.IsExist(err) {
		return err
	}
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
	return filepath.Join(v.driver.MetadataDir(), v.rawSnapshotLabel(label))
}

// Snapshot implements volume.Volume.Snapshot
func (v *DeviceMapperVolume) Snapshot(label string) error {
	if v.snapshotExists(label) {
		return volume.ErrSnapshotExists
	}
	label = v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	// Create a new device based on the current one
	oldHead := v.volumeDevice()
	newHead, err := utils.NewUUID62()
	if err != nil {
		return err
	}
	if err := v.driver.DeviceSet.AddDevice(newHead, oldHead); err != nil {
		return err
	}
	// Create the metadata path
	if err := os.MkdirAll(v.SnapshotMetadataPath(label), 0755); err != nil {
		return err
	}
	// Save the old HEAD as the snapshot
	if err := v.Metadata.AddSnapshot(label, oldHead); err != nil {
		return err
	}
	// Unmount the current device and mount the new one
	if err := v.driver.DeviceSet.UnmountDevice(oldHead); err != nil {
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(newHead, v.path, v.name); err != nil {
		return err
	}
	// Save the new HEAD as the current device
	return v.Metadata.SetCurrentDevice(newHead)
}

// Snapshots implements volume.Volume.Snapshots
func (v *DeviceMapperVolume) Snapshots() ([]string, error) {
	return v.Metadata.ListSnapshots(), nil
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *DeviceMapperVolume) RemoveSnapshot(label string) error {
	if !v.snapshotExists(label) {
		return fmt.Errorf("snapshot %s does not exist", label)
	}
	rawLabel := v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	device, err := v.Metadata.LookupSnapshotDevice(rawLabel)
	if err != nil {
		return err
	}
	// Remove the snapshot info from the metadata
	if err := v.Metadata.RemoveSnapshot(rawLabel); err != nil {
		return err
	}
	// Delete the device itself
	if err := v.driver.DeviceSet.DeleteDevice(device); err != nil {
		return err
	}
	return nil
}

// Rollback implements volume.Volume.Rollback
func (v *DeviceMapperVolume) Rollback(label string) error {
	if !v.snapshotExists(label) {
		return fmt.Errorf("snapshot %s does not exist", label)
	}
	label = v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	current := v.Metadata.CurrentDevice()
	device, err := v.Metadata.LookupSnapshotDevice(label)
	if err != nil {
		return err
	}
	// Make a new device based on the snapshot
	newHead, err := utils.NewUUID62()
	if err != nil {
		return err
	}
	if err := v.driver.DeviceSet.AddDevice(newHead, device); err != nil {
		return err
	}
	// Now unmount the current device and mount the new one
	if err := v.driver.DeviceSet.UnmountDevice(current); err != nil {
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(newHead, v.path, v.name); err != nil {
		return err
	}
	return v.Metadata.SetCurrentDevice(newHead)
}

// Export implements volume.Volume.Export
func (v *DeviceMapperVolume) Export(label, parent, outdir string) error {
	if !v.snapshotExists(label) {
		return volume.ErrSnapshotDoesNotExist
	}
	label = v.rawSnapshotLabel(label)
	mountpoint, err := ioutil.TempDir("", "serviced-export-volume-")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return err
	}
	//defer os.RemoveAll(mountpoint)
	device, err := v.Metadata.LookupSnapshotDevice(label)
	if err != nil {
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(device, mountpoint, label+"_export"); err != nil {
		return err
	}
	defer v.driver.DeviceSet.UnmountDevice(device)
	rsync := exec.Command("rsync", "-azh", mountpoint+"/", outdir+"/")
	rsync.Stdout = os.Stdout
	rsync.Stderr = os.Stderr
	if err := rsync.Run(); err != nil {
		glog.V(2).Infof("Could not perform rsync")
		return err
	}
	return nil
}

// Import implements volume.Volume.Import
func (v *DeviceMapperVolume) Import(label, indir string) error {
	if v.snapshotExists(label) {
		return volume.ErrSnapshotExists
	}
	label = v.rawSnapshotLabel(label)
	mountpoint, err := ioutil.TempDir("", "serviced-import-volume-")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(mountpoint)
	device, err := utils.NewUUID62()
	if err != nil {
		return err
	}
	if err := v.driver.DeviceSet.AddDevice(device, ""); err != nil {
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(device, mountpoint, label+"_import"); err != nil {
		return err
	}
	defer v.driver.DeviceSet.UnmountDevice(device)
	rsync := exec.Command("rsync", "-azh", indir+"/", mountpoint+"/")
	rsync.Stdout = os.Stdout
	rsync.Stderr = os.Stderr
	if err := rsync.Run(); err != nil {
		glog.V(2).Infof("Could not perform rsync")
		return err
	}
	return v.Metadata.AddSnapshot(label, device)
}

func (v *DeviceMapperVolume) volumeDevice() string {
	return v.Metadata.CurrentDevice()
}

func (v *DeviceMapperVolume) getSnapshotPrefix() string {
	return v.Tenant() + "_"
}

// rawSnapshotLabel ensures that <label> has the tenant prefix for this volume
func (v *DeviceMapperVolume) rawSnapshotLabel(label string) string {
	prefix := v.getSnapshotPrefix()
	if !strings.HasPrefix(label, prefix) {
		return prefix + label
	}
	return label
}

func (v *DeviceMapperVolume) snapshotExists(label string) bool {
	label = v.rawSnapshotLabel(label)
	return v.Metadata.SnapshotExists(label)
}