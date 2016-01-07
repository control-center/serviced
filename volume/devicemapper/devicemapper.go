package devicemapper

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/docker/docker/daemon/graphdriver/devmapper"
	"github.com/docker/docker/pkg/devicemapper"
	"github.com/docker/docker/pkg/units"
	"github.com/zenoss/glog"
)

var (
	ErrNoShrinkage = errors.New("you can't shrink a device")
)

func init() {
	volume.Register(volume.DriverTypeDeviceMapper, Init)
}

type DeviceMapperDriver struct {
	root         string
	options      []string
	DeviceSet    *devmapper.DeviceSet
	DevicePrefix string
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
	glog.V(2).Infof("Create() (%s) START", volumeName)
	defer glog.V(2).Infof("Create() (%s) END", volumeName)
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

// GetTenant implements volume.Driver.GetTenant
func (d *DeviceMapperDriver) GetTenant(volumeName string) (volume.Volume, error) {
	if !d.Exists(volumeName) {
		return nil, volume.ErrVolumeNotExists
	}
	return d.Get(getTenant(volumeName))
}

// Resize implements volume.Driver.Resize.
func (d *DeviceMapperDriver) Resize(volumeName string, size uint64) error {
	vol, err := d.newVolume(volumeName)
	if err != nil {
		return err
	}

	dev := vol.Metadata.CurrentDevice()
	devicename := fmt.Sprintf("%s-%s", d.DevicePrefix, dev)

	// Get the active table for the device
	start, oldSectors, targetType, params, err := devicemapper.GetTable(devicename)
	if err != nil {
		return err
	}

	// Figure out how many sectors we need
	newSectors := size / 512

	// If the new table size isn't larger than the old, it's invalid
	if newSectors <= oldSectors {
		return ErrNoShrinkage
	}

	// Create the new table description using the sectors computed
	oldTable := fmt.Sprintf("%d %d %s %s", start, oldSectors, targetType, params)
	newTable := fmt.Sprintf("%d %d %s %s", start, newSectors, targetType, params)
	glog.V(2).Infof("Replacing old table (%s) with new table (%s)", oldTable, newTable)

	// Would love to do this with libdevmapper, but DM_TABLE_LOAD isn't
	// exposed, so we'll shell out rather than muck with ioctl
	dmsetupLoad := exec.Command("dmsetup", "load", devicename)
	dmsetupLoad.Stdin = strings.NewReader(newTable)
	if output, err := dmsetupLoad.CombinedOutput(); err != nil {
		glog.Errorf("Unable to load new table (%s)", string(output))
		return err
	}
	glog.V(2).Infof("Inactive table slot updated with new size")

	// "Resume" the device to load the inactive table into the active slot
	if err := devicemapper.ResumeDevice(devicename); err != nil {
		return err
	}
	glog.V(2).Infof("Loaded inactive table into the active slot")

	// Resize the filesystem to use the new space
	dmDevice := fmt.Sprintf("/dev/mapper/%s", devicename)
	resize2fs := exec.Command("resize2fs", dmDevice)
	if output, err := resize2fs.CombinedOutput(); err != nil {
		glog.Errorf("Unable to resize filesystem (%s)", string(output))
		return err
	}
	newSize := volume.FilesystemBytesSize(vol.Path())
	human := units.BytesSize(float64(newSize))
	glog.Infof("Resized filesystem. New size: %s", human)
	return nil
}

// Get implements volume.Driver.Get
func (d *DeviceMapperDriver) Get(volumeName string) (volume.Volume, error) {
	glog.V(2).Infof("Get() (%s) START", volumeName)
	defer glog.V(2).Infof("Get() (%s) END", volumeName)
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
			glog.Errorf("Error mounting device %q on %q for volume %q: %s", device, mountpoint, volumeName, err)
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
	glog.V(2).Infof("Cleanup() START")
	defer glog.V(2).Infof("Cleanup() END")
	if d.DeviceSet == nil {
		return nil
	}
	glog.V(1).Infof("Cleaning up devicemapper driver at %s", d.root)
	for _, volname := range d.List() {
		vol, err := d.newVolume(volname)
		if err != nil {
			glog.V(2).Infof("Unable to get volume %s; skipping", volname)
			continue
		}
		if mounted, _ := devmapper.Mounted(vol.Path()); mounted {
			glog.V(2).Infof("Unmounting %s", volname)
			if err := d.Release(volname); err != nil {
				return err
			}
		}
	}
	return d.DeviceSet.Shutdown()
}

// Release implements volume.Driver.Release
func (d *DeviceMapperDriver) Release(volumeName string) error {
	glog.V(2).Infof("Release() (%s) START", volumeName)
	defer glog.V(2).Infof("Release() (%s) END", volumeName)
	vol, err := d.newVolume(volumeName)
	if err != nil {
		return err
	}
	if err := vol.unmount(); err != nil {
		glog.Errorf("Error whilst unmounting %s: %s", vol.path, err)
		return err
	}
	device := vol.Metadata.CurrentDevice()
	glog.V(2).Infof("Deactivating device (%s)", device)
	d.DeviceSet.Lock()
	defer d.DeviceSet.Unlock()
	if err := d.deactivateDevice(device); err != nil {
		glog.Errorf("Error removing device %s for volume %s: %s", device, volumeName, err)
		return err
	}
	glog.V(2).Infof("Deactivated device")
	return nil
}

// Remove implements volume.Driver.Remove
func (d *DeviceMapperDriver) Remove(volumeName string) error {
	glog.V(2).Infof("Remove() (%s) START", volumeName)
	defer glog.V(2).Infof("Remove() (%s) END", volumeName)
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
		glog.V(1).Infof("Error releasing device: %s", err)
	}
	glog.V(1).Infof("Removing volume %s", volumeName)
	if err := d.DeviceSet.DeleteDevice(v.volumeDevice(), false); err != nil {
		glog.Errorf("Could not delete device %s: %s", volumeName, err)
		return err
	}
	if err := os.RemoveAll(filepath.Join(d.MetadataDir(), volumeName)); err != nil {
		return err
	}
	return nil
}

// Issues the underlying dm remove operation.
func (d *DeviceMapperDriver) deactivateDevice(devname string) error {
	glog.V(2).Infof("deactivateDevice START(%s)", devname)
	defer glog.V(2).Infof("deactivateDevice END(%s)", devname)

	var err error

	devicename := fmt.Sprintf("%s-%s", d.DevicePrefix, devname)
	for i := 0; i < 200; i++ {
		err = devicemapper.RemoveDevice(devicename)
		if err == nil {
			break
		}
		if err != devicemapper.ErrBusy {
			return err
		}

		// If we see EBUSY it may be a transient error,
		// sleep a bit a retry a few times.
		d.DeviceSet.Unlock()
		time.Sleep(100 * time.Millisecond)
		d.DeviceSet.Lock()
	}

	return err
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
		deviceSet, err := devmapper.NewDeviceSet(poolPath, true, d.options, nil, nil)
		if err != nil {
			return err
		}
		d.DeviceSet = deviceSet
		prefix, err := GetDevicePrefix(poolPath)
		if err != nil {
			return err
		}
		d.DevicePrefix = prefix
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
		Driver: volume.DriverTypeDeviceMapper,
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
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil && !os.IsExist(err) {
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

// unmount unmounts the current device from the volume mount point. Docker's
// code has a function for this, but it depends on some internal state that we
// both can't count on and don't care about.
func (v *DeviceMapperVolume) unmount() error {
	glog.V(2).Infof("unmount() (%s) START", v.name)
	defer glog.V(2).Infof("unmount() (%s) END", v.name)
	v.driver.DeviceSet.Lock()
	defer v.driver.DeviceSet.Unlock()

	mountPath := v.Path()
	glog.V(2).Infof("Unmounting from path %s START", mountPath)
	if err := unmount(mountPath); err != nil {
		glog.Errorf("Got an error unmounting %s (%s)", mountPath, err)
		return err
	}
	glog.V(2).Infof("Unmounting from path %s END", mountPath)
	// pessimistically clean up the DeviceInfo object in memory. If it isn't
	// there the next time the device is requested, it'll be recreated from
	// disk.
	delete(v.driver.DeviceSet.Devices, v.Metadata.CurrentDevice())
	return nil
}

func unmount(mountpoint string) error {
	if mounted, _ := devmapper.Mounted(mountpoint); mounted {
		if err := syscall.Unmount(mountpoint, syscall.MNT_DETACH); err != nil {
			glog.Errorf("Error unmounting %s: %s", mountpoint, err)
			return err
		}
	}
	return nil
}

// writeSnapshotInfo writes metadata about a snapshot
func (v *DeviceMapperVolume) writeSnapshotInfo(label string, info *volume.SnapshotInfo) error {
	writer, err := v.WriteMetadata(label, ".SNAPSHOTINFO")
	if err != nil {
		glog.Errorf("Could not write meta info for snapshot %s: %s", label, err)
		return err
	}
	defer writer.Close()
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(info); err != nil {
		glog.Errorf("Could not export meta info for snapshot %s: %s", label, err)
		return err
	}
	return nil
}

// SnapshotInfo returns the meta info for a snapshot
func (v *DeviceMapperVolume) SnapshotInfo(label string) (*volume.SnapshotInfo, error) {
	reader, err := v.ReadMetadata(label, ".SNAPSHOTINFO")
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s: %s", label, err)
		return nil, err
	}
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	var info volume.SnapshotInfo
	if err := decoder.Decode(&info); err != nil {
		glog.Errorf("Could not decode snapshot info for %s: %s", label, err)
		return nil, err
	}
	return &info, err
}

// Snapshot implements volume.Volume.Snapshot
func (v *DeviceMapperVolume) Snapshot(label, message string, tags []string) error {
	glog.V(2).Infof("Snapshot() (%s) START", v.name)
	defer glog.V(2).Infof("Snapshot() (%s) END", v.name)
	if v.snapshotExists(label) {
		return volume.ErrSnapshotExists
	}

	//make sure none of the tags already exist
	for _, tagName := range tags {
		if info, err := v.GetSnapshotWithTag(tagName); err != nil {
			glog.Errorf("Error checking for tag conflicts: %v", err)
			return err
		} else if info != nil {
			glog.Errorf("Can not create snapshot %s, tag '%s' already exists", label, tagName)
			return volume.ErrTagAlreadyExists
		}
	}

	label = v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	// Create a new device based on the current one
	activeDevice := v.volumeDevice()
	snapshotDevice, err := utils.NewUUID62()
	if err != nil {
		return err
	}
	glog.V(2).Infof("Creating snapshot device %s based on %s", snapshotDevice, activeDevice)
	if err := v.driver.DeviceSet.AddDevice(snapshotDevice, activeDevice); err != nil {
		glog.Errorf("Unable to add devicemapper device: %s", err)
		return err
	}
	// write snapshot info
	info := volume.SnapshotInfo{
		Name:     label,
		TenantID: v.Tenant(),
		Label:    strings.TrimPrefix(label, v.Tenant()+"_"),
		Tags:     tags,
		Message:  message,
		Created:  time.Now(),
	}
	if err := v.writeSnapshotInfo(label, &info); err != nil {
		return err
	}
	// Save the new HEAD as the snapshot
	glog.V(2).Infof("Saving device %s as snapshot %s", snapshotDevice, label)
	if err := v.Metadata.AddSnapshot(label, snapshotDevice); err != nil {
		glog.Errorf("Unable to save snapshot metadata: %s", err)
		return err
	}
	return nil
}

// TagSnapshot implements volume.Volume.TagSnapshot
func (v *DeviceMapperVolume) TagSnapshot(label string, tagName string) ([]string, error) {
	//make sure the snapshot exists
	if !v.snapshotExists(label) {
		return nil, volume.ErrSnapshotDoesNotExist
	}

	//make sure the tag doesn't already exist
	if info, err := v.GetSnapshotWithTag(tagName); err != nil {
		return nil, err
	} else if info != nil {
		glog.Errorf("Tag '%s', is already used by snapshot %s", tagName, info.Label)
		return nil, volume.ErrTagAlreadyExists
	}

	v.Lock()
	defer v.Unlock()
	//get the current info for the snapshot
	info, err := v.SnapshotInfo(label)
	if err != nil {
		glog.Errorf("Unable to retrieve info for existing snapshot: %s", err)
		return nil, err
	}

	//add the new tag name
	info.Tags = append(info.Tags, tagName)

	//write out the updated info
	if err := v.writeSnapshotInfo(label, info); err != nil {
		glog.Errorf("Error writing updated snapshot info with new tag: %s", err)
		return nil, err
	}

	return info.Tags, nil
}

// RemoveSnapshotTag implements volume.Volume.RemoveSnapshotTag
func (v *DeviceMapperVolume) RemoveSnapshotTag(label string, tagName string) ([]string, error) {
	//make sure the snapshot exists
	if !v.snapshotExists(label) {
		return nil, volume.ErrSnapshotDoesNotExist
	}

	v.Lock()
	defer v.Unlock()
	//get the current info for the snapshot
	info, err := v.SnapshotInfo(label)
	if err != nil {
		glog.Errorf("Unable to retrieve info for existing snapshot: %s", err)
		return nil, err
	}

	newTagList := []string{}

	//add the tag names that don't match the one we are removing
	for _, name := range info.Tags {
		//add the tag to the new list
		if name != tagName {
			newTagList = append(newTagList, name)
		}
	}

	//Replace the tag list with the new one
	info.Tags = newTagList

	//write out the updated info
	if err := v.writeSnapshotInfo(label, info); err != nil {
		glog.Errorf("Error writing updated snapshot info with removed tag: %s", err)
		return nil, err
	}

	return info.Tags, nil
}

// GetSnapshotWithTag implements volume.Volume.GetSnapshotWithTag
func (v *DeviceMapperVolume) GetSnapshotWithTag(tagName string) (*volume.SnapshotInfo, error) {
	var (
		snaps []string
		info  *volume.SnapshotInfo
		err   error
	)

	if snaps, err = v.Snapshots(); err != nil {
		glog.Errorf("Could not get current snapshot list : %v", err)
		return nil, err
	} else {
		for _, snaplabel := range snaps {
			if info, err = v.SnapshotInfo(snaplabel); err != nil {
				glog.Errorf("Could not get info for %s: %v", snaplabel, err)
				return nil, err
			}

			for _, t := range info.Tags {
				if t == tagName {
					return info, nil
				}
			}
		}
	}
	return nil, nil
}

// Snapshots implements volume.Volume.Snapshots
func (v *DeviceMapperVolume) Snapshots() ([]string, error) {
	return v.Metadata.ListSnapshots(), nil
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *DeviceMapperVolume) RemoveSnapshot(label string) error {
	glog.V(2).Infof("RemoveSnapshot() (%s) START", v.name)
	defer glog.V(2).Infof("RemoveSnapshot() (%s) END", v.name)
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
	glog.V(2).Infof("Deactivating snapshot device %s", device)
	v.driver.DeviceSet.Lock()
	if err := v.driver.deactivateDevice(device); err != nil {
		glog.V(2).Infof("Error deactivating device (%s): %s", device, err)
	}
	v.driver.DeviceSet.Unlock()
	if err := v.driver.DeviceSet.DeleteDevice(device, false); err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}
	return nil
}

// Rollback implements volume.Volume.Rollback
func (v *DeviceMapperVolume) Rollback(label string) error {
	glog.V(2).Infof("Rollback() (%s) START", v.name)
	defer glog.V(2).Infof("Rollback() (%s) END", v.name)
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
	glog.V(2).Infof("Creating new head device %s based on snapshot %s", newHead, device)
	if err := v.driver.DeviceSet.AddDevice(newHead, device); err != nil {
		return err
	}
	// Now unmount the current device and mount the new one
	glog.V(2).Infof("Unmounting old head device %s", current)
	if err := v.unmount(); err != nil {
		return err
	}
	glog.V(2).Infof("Rollback(): mounting new head device %s", newHead)
	if err := v.driver.DeviceSet.MountDevice(newHead, v.path, v.name); err != nil {
		return err
	}
	glog.V(2).Infof("Deactivating old head device %s", current)
	v.driver.DeviceSet.Lock()
	if err := v.driver.deactivateDevice(current); err != nil {
		glog.V(2).Infof("Error removing device: %s", err)
	}
	v.driver.DeviceSet.Unlock()
	glog.V(2).Infof("Deleting old head device %s", current)
	if err := v.driver.DeviceSet.DeleteDevice(current, false); err != nil {
		glog.Warningf("Error cleaning up device %s: %s", current, err)
	}
	return v.Metadata.SetCurrentDevice(newHead)
}

// Export implements volume.Volume.Export
func (v *DeviceMapperVolume) Export(label, parent string, writer io.Writer) error {
	glog.V(2).Infof("Export() (%s) START", v.name)
	defer glog.V(2).Infof("Export() (%s) END", v.name)
	if !v.snapshotExists(label) {
		return volume.ErrSnapshotDoesNotExist
	}
	label = v.rawSnapshotLabel(label)
	mountpoint, err := ioutil.TempDir("", "serviced-export-volume-")
	if err != nil {
		return err
	}
	//defer os.RemoveAll(mountpoint)
	device, err := v.Metadata.LookupSnapshotDevice(label)
	if err != nil {
		return err
	}
	glog.V(2).Infof("Mounting temporary export device %s", device)
	if err := v.driver.DeviceSet.MountDevice(device, mountpoint, label); err != nil {
		return err
	}
	defer func() {
		glog.V(2).Infof("Unmounting temporary export device %s", device)
		if err := unmount(mountpoint); err != nil {
			glog.V(2).Infof("Error unmounting (%s): %s", mountpoint, err)
		}
		glog.V(2).Infof("Deactivating temporary export device %s", device)
		v.driver.DeviceSet.Lock()
		if err := v.driver.deactivateDevice(device); err != nil {
			glog.V(2).Infof("Error deactivating device (%s): %s", device, err)
		}
		v.driver.DeviceSet.Unlock()
	}()
	// Set up the file stream
	tarfile := tar.NewWriter(writer)
	defer tarfile.Close()
	// Set the driver type
	header := &tar.Header{Name: fmt.Sprintf("%s-driver", label), Size: int64(len([]byte(v.Driver().DriverType())))}
	if err := tarfile.WriteHeader(header); err != nil {
		glog.Errorf("Could not export driver type header: %s", err)
		return err
	}
	if _, err := fmt.Fprint(tarfile, v.Driver().DriverType()); err != nil {
		glog.Errorf("Could not export driver type: %s", err)
		return err
	}
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
	glog.V(2).Infof("Import() (%s) START", v.name)
	defer glog.V(2).Infof("Import() (%s) END", v.name)
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
	glog.V(2).Infof("Creating imported snapshot device %s", device)
	if err := v.driver.DeviceSet.AddDevice(device, ""); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(mountpoint, label), 0755); err != nil {
		return err
	}
	glog.V(2).Infof("Mounting imported snapshot device %s", device)
	if err := v.driver.DeviceSet.MountDevice(device, filepath.Join(mountpoint, label), fmt.Sprintf("%s_import", label)); err != nil {
		return err
	}
	defer func() {
		mp := filepath.Join(mountpoint, label)
		glog.V(2).Infof("Unmounting imported snapshot device %s", device)
		if err := unmount(mp); err != nil {
			glog.V(2).Infof("Error unmounting (%s): %s", mp, err)
		}
		glog.V(2).Infof("Deactivating imported snapshot device %s", device)
		v.driver.DeviceSet.Lock()
		if err := v.driver.deactivateDevice(device); err != nil {
			glog.V(2).Infof("Error deactivating device (%s): %s", device, err)
		}
		v.driver.DeviceSet.Unlock()
	}()
	// write volume and metadata
	driverfile := fmt.Sprintf("%s-driver", label)
	volumedir := fmt.Sprintf("%s-volume", label)
	metadatadir := fmt.Sprintf("%s-metadata", label)
	var drivertype string
	tarfile := tar.NewReader(reader)
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			glog.Errorf("Could not import archive: %s", err)
			return err
		}
		if header.Name == driverfile {
			buf := bytes.NewBufferString("")
			if _, err := buf.ReadFrom(tarfile); err != nil {
				return err
			}
			drivertype = buf.String()
		} else if strings.HasPrefix(header.Name, volumedir) {
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
	if drivertype == "" {
		return errors.New("incompatible snapshot")
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
