package devicemapper

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/docker/docker/daemon/graphdriver/devmapper"
	"github.com/zenoss/glog"
)

func init() {
	volume.Register(volume.DriverTypeDeviceMapper, Init)
}

type DeviceMapperDriver struct {
	root      string
	options   []string
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

// Init initializes the devicemapper driver
func Init(root string, options []string) (volume.Driver, error) {
	driver := &DeviceMapperDriver{
		root:    root,
		options: options,
	}
	if err := driver.ensureInitialized(); err != nil {
		return nil, err
	}
	return driver, nil
}

// Root implements volume.Driver.Root
func (d *DeviceMapperDriver) Root() string {
	return d.root
}

// DriverType implements volume.Driver.DriverType
func (d *DeviceMapperDriver) DriverType() volume.DriverType {
	return volume.DriverTypeDeviceMapper
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

// Create implements volume.Driver.Create
func (d *DeviceMapperDriver) Create(volumeName string) (volume.Volume, error) {
	if d.Exists(volumeName) {
		return nil, volume.ErrVolumeExists
	}
	// Create a new device
	volumeDevice, err := utils.NewUUID62()
	if err != nil {
		return nil, err
	}
	if err := d.DeviceSet.AddDevice(volumeDevice, ""); err != nil {
		return nil, err
	}
	glog.V(1).Infof("Allocated new device %s", volumeDevice)
	// Create the mount target directory if it doesn't exist
	mountpoint := d.volumeDir(volumeName)
	if err := os.MkdirAll(mountpoint, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	glog.V(1).Infof("Ensured existence of mount point %s", mountpoint)
	md := d.MetadataDir()
	if err := os.MkdirAll(md, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	glog.V(1).Infof("Ensured existence of metadata dir %s", md)
	vol, err := d.newVolume(volumeName)
	if err != nil {
		return nil, err
	}
	if err := vol.Metadata.SetCurrentDevice(volumeDevice); err != nil {
		return nil, err
	}
	glog.V(1).Infof("Set current device to %s", volumeDevice)
	if err := d.DeviceSet.MountDevice(volumeDevice, mountpoint, volumeName); err != nil {
		return nil, err
	}
	glog.V(1).Infof("Mounted device %s to %s", volumeDevice, mountpoint)
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
	glog.V(2).Infof("Getting devicemapper volume %s", volumeName)
	vol, err := d.newVolume(volumeName)
	if err != nil {
		glog.Errorf("Error getting devicemapper volume: %s", err)
		return nil, err
	}
	if mounted, _ := devmapper.Mounted(vol.Path()); !mounted {
		device := vol.Metadata.CurrentDevice()
		mountpoint := vol.Path()
		label := vol.Tenant()
		if err := d.DeviceSet.MountDevice(device, mountpoint, label); err != nil {
			return nil, err
		}
		glog.V(2).Infof("Mounted device %s to %s", device, mountpoint)
	} else {
		glog.V(2).Infof("%s is already a mount point", vol.Path())
	}
	return vol, nil
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
	if finfo, err := os.Stat(filepath.Join(d.MetadataDir(), volumeName)); err != nil {
		return false
	} else {
		return finfo.IsDir()
	}
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
	if err := d.DeviceSet.UnmountDevice(device); err != nil {
		if !strings.HasPrefix(err.Error(), "UnmountDevice: device not-mounted id") {
			return err
		}
	}
	return nil
}

// Remove implements volume.Driver.Remove
func (d *DeviceMapperDriver) Remove(volumeName string) error {
	if !d.Exists(volumeName) {
		return nil
	}
	// get the volume
	v, err := d.newVolume(volumeName)
	if err != nil {
		return err
	}
	// remove the snapshots
	glog.V(1).Infof("Removing snapshots from %s", volumeName)
	snapshots, err := v.Snapshots()
	if err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		if err := v.RemoveSnapshot(snapshot); err != nil {
			return err
		}
	}
	if err := d.Release(volumeName); err != nil {
		return err
	}

	glog.V(1).Infof("Removing volume %s", volumeName)
	if err := d.DeviceSet.DeleteDevice(v.volumeDevice()); err != nil {
		glog.Errorf("Could not delete device %s: %s", volumeName, err)
		return err
	}
	if err := os.RemoveAll(filepath.Join(d.MetadataDir(), volumeName)); err != nil {
		return err
	}
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
		deviceSet, err := devmapper.NewDeviceSet(poolPath, true, d.options)
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

func (d *DeviceMapperDriver) Status() (*volume.Status, error) {
	glog.V(2).Info("devicemapper.Status()")
	dockerStatus := d.DeviceSet.Status()
	// convert dockerStatus to our status and return
	result := &volume.Status{
		Driver: DriverName,
		DriverData: map[string]string{
			"PoolName":          dockerStatus.PoolName,
			"DataFile":          dockerStatus.DataFile,
			"DataLoopback":      dockerStatus.DataLoopback,
			"MetadataFile":      dockerStatus.MetadataFile,
			"MetadataLoopback":  dockerStatus.MetadataLoopback,
			"SectorSize":        strconv.FormatUint(dockerStatus.SectorSize, 10),
			"UdevSyncSupported": strconv.FormatBool(dockerStatus.UdevSyncSupported),
		},
		UsageData: []volume.Usage{
			{Label: "Data", Type: "Available", Value: dockerStatus.Data.Available},
			{Label: "Data", Type: "Used", Value: dockerStatus.Data.Used},
			{Label: "Data", Type: "Total", Value: dockerStatus.Data.Total},
			{Label: "Metadata", Type: "Available", Value: dockerStatus.Metadata.Available},
			{Label: "Metadata", Type: "Used", Value: dockerStatus.Metadata.Used},
			{Label: "Metadata", Type: "Total", Value: dockerStatus.Metadata.Total},
		},
	}
	return result, nil
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

// WriteMetadata writes the metadata info for a snapshot
func (v *DeviceMapperVolume) WriteMetadata(label, name string) (io.WriteCloser, error) {
	filePath := filepath.Join(v.driver.MetadataDir(), v.rawSnapshotLabel(label), name)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		glog.Errorf("Could not create path for file %s: %s", name, err)
		return nil, err
	}
	return os.Create(filePath)
}

// ReadMetadata reads the metadata info from a snapshot
func (v *DeviceMapperVolume) ReadMetadata(label, name string) (io.ReadCloser, error) {
	filePath := filepath.Join(v.driver.MetadataDir(), v.rawSnapshotLabel(label), name)
	return os.Open(filePath)
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
		glog.Errorf("Unable to add devicemapper device: %s", err)
		return err
	}
	// Create the metadata path
	mdpath := filepath.Join(v.driver.MetadataDir(), label)
	if err := os.MkdirAll(mdpath, 0755); err != nil {
		glog.Errorf("Unable to create snapshot metadata directory at %s", mdpath)
		return err
	}
	// Unmount the current device and mount the new one
	if err := v.driver.DeviceSet.UnmountDevice(oldHead); err != nil {
		glog.Errorf("Unable to unmount device %s", oldHead)
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(newHead, v.path, v.name); err != nil {
		glog.Errorf("Unable to mount device %s at %s", newHead, v.path)
		return err
	}
	// Save the old HEAD as the snapshot
	if err := v.Metadata.AddSnapshot(label, oldHead); err != nil {
		glog.Errorf("Unable to save snapshot metadata: %s", err)
		return err
	}
	// Save the new HEAD as the current device
	if err := v.Metadata.SetCurrentDevice(newHead); err != nil {
		glog.Errorf("Unable to save device metadata: %s", err)
		return err
	}
	return nil
}

// Snapshots implements volume.Volume.Snapshots
func (v *DeviceMapperVolume) Snapshots() ([]string, error) {
	return v.Metadata.ListSnapshots(), nil
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *DeviceMapperVolume) RemoveSnapshot(label string) error {
	if !v.snapshotExists(label) {
		return volume.ErrSnapshotDoesNotExist
	}
	rawLabel := v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	device, err := v.Metadata.LookupSnapshotDevice(rawLabel)
	if err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}
	// Remove the snapshot info from the metadata
	if err := v.Metadata.RemoveSnapshot(rawLabel); err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}
	// Delete the device itself
	if err := v.driver.DeviceSet.DeleteDevice(device); err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}
	return nil
}

// Rollback implements volume.Volume.Rollback
func (v *DeviceMapperVolume) Rollback(label string) error {
	if !v.snapshotExists(label) {
		return volume.ErrSnapshotDoesNotExist
	}
	label = v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	current := v.volumeDevice()
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
	if err := v.driver.DeviceSet.DeleteDevice(current); err != nil {
		glog.Warningf("Error cleaning up device %s: %s", current, err)
	}
	return v.Metadata.SetCurrentDevice(newHead)
}

// Export implements volume.Volume.Export
func (v *DeviceMapperVolume) Export(label, parent string, writer io.Writer) error {
	if !v.snapshotExists(label) {
		return volume.ErrSnapshotDoesNotExist
	}
	label = v.rawSnapshotLabel(label)
	mountpoint, err := ioutil.TempDir("", "serviced-export-volume-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mountpoint)
	device, err := v.Metadata.LookupSnapshotDevice(label)
	if err != nil {
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(device, mountpoint, label); err != nil {
		return err
	}
	defer v.driver.DeviceSet.UnmountDevice(device)
	// Set up the file stream
	tarfile := tar.NewWriter(writer)
	defer tarfile.Close()
	// Write metadata
	mdpath := filepath.Join(v.driver.MetadataDir(), label)
	if err := volume.ExportDirectory(tarfile, mdpath, fmt.Sprintf("%s-metadata", label)); err != nil {
		return err
	}
	// Write volume
	if err := volume.ExportDirectory(tarfile, mountpoint, fmt.Sprintf("%s-volume", label)); err != nil {
		return err
	}
	return nil
}

// Import implements volume.Volume.Import
func (v *DeviceMapperVolume) Import(label string, reader io.Reader) error {
	if v.snapshotExists(label) {
		return volume.ErrSnapshotExists
	}
	label = v.rawSnapshotLabel(label)
	// set up the device
	mountpoint, err := ioutil.TempDir("", "serviced-import-volume-")
	if err != nil {
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
	if err := os.MkdirAll(filepath.Join(mountpoint, label), 0755); err != nil {
		return err
	}
	if err := v.driver.DeviceSet.MountDevice(device, filepath.Join(mountpoint, label), fmt.Sprintf("%s_import", label)); err != nil {
		return err
	}
	defer v.driver.DeviceSet.UnmountDevice(device)
	// write volume and metadata
	volumedir, metadatadir := fmt.Sprintf("%s-volume", label), fmt.Sprintf("%s-metadata", label)
	tarfile := tar.NewReader(reader)
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			glog.Errorf("Could not import archive: %s", err)
			return err
		}
		if strings.HasPrefix(header.Name, volumedir) {
			header.Name = strings.Replace(header.Name, volumedir, label, 1)
			if err := volume.ImportArchiveHeader(header, tarfile, mountpoint); err != nil {
				return err
			}
		} else if strings.HasPrefix(header.Name, metadatadir) {
			header.Name = strings.Replace(header.Name, metadatadir, label, 1)
			if err := volume.ImportArchiveHeader(header, tarfile, v.driver.MetadataDir()); err != nil {
				return err
			}
		}
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
