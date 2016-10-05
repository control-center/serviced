// +build linux,!darwin

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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/control-center/serviced/commons/atomicfile"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/devicemapper/devmapper"
	"github.com/docker/docker/pkg/devicemapper"
	"github.com/docker/go-units"
	"github.com/zenoss/glog"
)

var (
	ErrNoShrinkage          = errors.New("you can't shrink a device")
	ErrInvalidOption        = errors.New("invalid option")
	ErrInvalidArg           = errors.New("invalid argument")
	ErrIncompatibleSnapshot = errors.New("incompatible snapshot")
	ErrDeleteBaseDevice     = errors.New("will not attempt to delete base device")
	ErrBaseDeviceHash       = errors.New("can't load a volume that uses the base device")
)

func init() {
	volume.Register(volume.DriverTypeDeviceMapper, Init)
}

type devInfo struct {
	DeviceID      int    `json:"device_id"`
	Size          uint64 `json:"size"`
	TransactionID uint64 `json:"transaction_id"`
	Initialized   bool   `json:"initialized"`
	Deleted       bool   `json:"deleted"`
}

type volumeInfo struct {
	Size uint64
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

	driver.cleanUpSnapshots()

	return driver, nil
}

// If there any snapshots on disk that are not tied to a device in the metadata, the
// snapshot should be removed.
func (d *DeviceMapperDriver) cleanUpSnapshots() {
	snapshotsOnDisk, err := d.getSnapshotsOnDisk()
	if err != nil || snapshotsOnDisk == nil {
		return
	}

	glog.V(2).Infof("Snapshots on disk: %v", snapshotsOnDisk)

	snapshotsInMetadata, err := d.getSnapshotsFromMetadata()
	if err != nil || snapshotsInMetadata == nil {
		return
	}

	glog.V(2).Infof("Snapshots in metadata: %v", snapshotsInMetadata)

	for _, snapshotOnDisk := range snapshotsOnDisk {
		if _, ok := snapshotsInMetadata[snapshotOnDisk]; !ok {
			glog.V(2).Infof("Removing Snapshot: %v", snapshotOnDisk)
			os.RemoveAll(filepath.Join(d.MetadataDir(), snapshotOnDisk))
		} else {
			// Remove the snapshot on disk since it was found in the metadata map.  This
			// will leave the snapshotsInMetadata map with only snapshots that are in metadata but
			// not on disk, so they should all be removed from the metadata.
			delete(snapshotsInMetadata, snapshotOnDisk)
		}
	}

	for snapshotInMetadata, volumeName := range snapshotsInMetadata {
		glog.V(2).Infof("Removing Snapshot from metadata: %v", snapshotInMetadata)
		v, err := d.getVolume(volumeName, false)
		if err != nil {
			continue
		}

		v.Lock()
		defer v.Unlock()

		deviceHash, err := v.Metadata.LookupSnapshotDevice(snapshotInMetadata)
		if err != nil {
			glog.Errorf("Error removing snapshot: %v", err)
			continue
		}

		// Remove the snapshot info from the volume metadata
		if err := v.Metadata.RemoveSnapshot(snapshotInMetadata); err != nil {
			glog.Errorf("Error removing snapshot: %v", err)
			continue
		}

		glog.V(2).Infof("Deactivating snapshot device %s", deviceHash)
		v.driver.DeviceSet.Lock()
		if err := v.driver.deactivateDevice(deviceHash); err != nil {
			glog.Errorf("Error deactivating device (%s): %s", deviceHash, err)
			continue
		}
		v.driver.DeviceSet.Unlock()
		if err := v.driver.deleteDevice(deviceHash, false); err != nil {
			glog.Errorf("Error removing snapshot: %v", err)
			continue
		}
	}
}

func (d *DeviceMapperDriver) getSnapshotsFromMetadata() (map[string]string, error) {
	snapshots := make(map[string]string)

	for _, volname := range d.ListTenants() {
		volume, err := d.getVolume(volname, false)
		if err != nil {
			return nil, err
		}

		for _, s := range volume.Metadata.ListSnapshots() {
			snapshots[s] = volname
		}
	}

	return snapshots, nil
}

func (d *DeviceMapperDriver) getSnapshotsOnDisk() ([]string, error) {
	var snapshots []string

	files, err := ioutil.ReadDir(d.MetadataDir())
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		for _, volname := range d.ListTenants() {
			volume, err := d.getVolume(volname, false)
			if err != nil {
				return nil, err
			}

			if !volume.isInvalidSnapshot(file.Name()) {
				snapshots = append(snapshots, file.Name())
				break
			}
		}
	}

	return snapshots, nil
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

func (d *DeviceMapperDriver) ListTenants() (result []string) {
	set := make(map[string]struct{})
	for _, vol := range d.List() {
		set[getTenant(vol)] = struct{}{}
	}
	for k := range set {
		// Only include the tenant if its volume can be retrieved
		if _, err := d.getVolume(k, false); err == nil {
			result = append(result, k)
		}
	}
	return
}

// Create implements volume.Driver.Create
func (d *DeviceMapperDriver) Create(volumeName string) (volume.Volume, error) {
	glog.V(2).Infof("Create() (%s) START", volumeName)
	defer glog.V(2).Infof("Create() (%s) END", volumeName)

	// Do not create if the device already exists
	if d.Exists(volumeName) {
		return nil, volume.ErrVolumeExists
	}

	// Create a new device
	deviceHash, err := d.addDevice("")
	if err != nil {
		return nil, err
	}
	glog.V(1).Infof("Allocated new device %s", deviceHash)

	// Create the mount target directory if it doesn't exist
	mountpoint := d.volumeDir(volumeName)
	if err := os.MkdirAll(mountpoint, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	glog.V(1).Infof("Ensured existence of mount point %s", mountpoint)

	// Create the metadata directory if it doesn't exist.
	md := d.MetadataDir()
	if err := os.MkdirAll(md, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	glog.V(1).Infof("Ensured existence of metadata dir %s", md)

	// Instantiate the volume
	vol, err := d.getVolume(volumeName, true)
	if err != nil {
		return nil, err
	}

	// Set the device as HEAD
	if err := vol.Metadata.SetCurrentDevice(deviceHash); err != nil {
		return nil, err
	}
	glog.V(1).Infof("Set current device to %s", deviceHash)

	// Mount the device
	if err := d.DeviceSet.MountDevice(deviceHash, mountpoint, volumeName); err != nil {
		return nil, err
	}
	glog.V(1).Infof("Mounted device %s to %s", deviceHash, mountpoint)

	return vol, nil
}

// addDevice creates a new dm device and returns its device ID
func (d *DeviceMapperDriver) addDevice(headDeviceHash string) (string, error) {

	// Generate the device ID.
	deviceHash, err := utils.NewUUID62()
	if err != nil {
		return "", err
	}

	// Create the device.
	if err := d.DeviceSet.AddDevice(deviceHash, headDeviceHash); err != nil {
		glog.Errorf("Unable to create device %s: %s", deviceHash, err)
		return "", err
	}

	if headDeviceHash == "" {

		// Activate the device.
		d.DeviceSet.GetDeviceStatus(deviceHash)

		// CC-2219: Ensure the device base size is equal or greater than the
		// configured dm.basesize.
		size, err := d.deviceSize(deviceHash)
		if err != nil {
			glog.Errorf("Could not read the size of device %s: %s", deviceHash, err)
			return "", err
		}
		if baseSize := d.baseSize(); baseSize > size {

			// Make the device to match dm.basesize
			glog.V(2).Infof("Device size is smaller than dm.basesize; expanding")
			if err := d.resize(deviceHash, baseSize); err != nil {
				glog.Errorf("Could not update the size of the device %s: %s", deviceHash, err)
				return "", err
			}
		}
	}

	return deviceHash, nil
}

// baseSize returns value of dm.basesize if it is set, otherwise returns 100G.
func (d *DeviceMapperDriver) baseSize() uint64 {
	for _, option := range d.options {
		if strings.HasPrefix(option, "dm.basesize=") {
			if size, err := units.RAMInBytes(strings.TrimPrefix(option, "dm.basesize=")); err == nil {
				return uint64(size)
			}
		}
	}
	return 100 * units.GiB
}

// deviceSize returns the size of a device.
func (d *DeviceMapperDriver) deviceSize(deviceHash string) (uint64, error) {
	return getDeviceSize(d.devicePath(deviceHash))
}

// devicePath returns the /dev/mapper path of a device
func (d *DeviceMapperDriver) devicePath(deviceHash string) string {
	return fmt.Sprintf("/dev/mapper/%s", d.deviceName(deviceHash))
}

// deviceName returns the name of a device.
func (d *DeviceMapperDriver) deviceName(deviceHash string) string {
	return fmt.Sprintf("%s-%s", d.DevicePrefix, deviceHash)
}

// getVolume builds the volume object from its volume name
//  If create is true, we will create the metadata file if it doesn't already exist
func (d *DeviceMapperDriver) getVolume(volumeName string, create bool) (*DeviceMapperVolume, error) {
	tenant := getTenant(volumeName)
	metadata, err := NewMetadata(d.MetadataPath(tenant), create)
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

	// if create is false, check to make sure the metadata info isn't empty or equal to base
	if !create {
		deviceHash := vol.deviceHash()
		if deviceHash == "" || deviceHash == "base" {
			return nil, ErrBaseDeviceHash
		}
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
	vol, err := d.getVolume(volumeName, false)
	if err != nil {
		return err
	}
	if err := d.resize(vol.deviceHash(), size); err != nil {
		return err
	}
	newSize := volume.FilesystemBytesSize(vol.Path())
	human := units.BytesSize(float64(newSize))
	glog.Infof("Resized filesystem. New size: %s", human)
	return nil
}

func (d *DeviceMapperDriver) resize(deviceHash string, size uint64) error {

	// Get the current size of the device
	curSize, err := d.deviceSize(deviceHash)
	if err != nil {
		glog.Errorf("Could not get the current size of the device %s: %s", deviceHash, err)
		return err
	}

	// Get the active table for the device
	deviceName := d.deviceName(deviceHash)
	start, oldSectors, targetType, params, err := devicemapper.GetTable(deviceName)
	if err != nil {
		return err
	}

	// Figure out how many sectors we need
	newSectors := size / 512
	if newSectors <= oldSectors {
		return ErrNoShrinkage
	}

	// Create the new table description using the sectors computed
	oldTable := fmt.Sprintf("%d %d %s %s", start, oldSectors, targetType, params)
	newTable := fmt.Sprintf("%d %d %s %s", start, newSectors, targetType, params)
	glog.V(2).Infof("Replacing old table (%s) with new table (%s)", oldTable, newTable)

	// Update the device info
	if err := d.resizeDevice(deviceHash, size); err != nil {
		glog.Errorf("Could not reset size of device %s: %s", deviceName, err)
		return err
	}
	if err := func() (err error) {

		// Would love to do this with libdevmapper, but DM_TABLE_LOAD isn't
		// exposed, so we'll shell out rather than muck with ioctl
		dmsetupLoad := exec.Command("dmsetup", "load", deviceName)
		dmsetupLoad.Stdin = strings.NewReader(newTable)
		if output, err := dmsetupLoad.CombinedOutput(); err != nil {
			glog.Errorf("Unable to load new table (%s)", string(output))
			return err
		}
		glog.V(2).Infof("Inactive table slot updated with new size")

		// "Resume" the device to load the inactive table into the active slot
		if err := devicemapper.ResumeDevice(deviceName); err != nil {
			return err
		}
		glog.V(2).Infof("Loaded inactive table into the active slot")

		// Resize the filesystem to use the new space
		if err := resize2fs(d.devicePath(deviceHash)); err != nil {
			glog.Errorf("Unable to resize filesystem: %s", err)
			return err
		}
		return nil
	}(); err != nil {
		d.resizeDevice(deviceHash, curSize)
		return err
	}
	return nil
}

// Get implements volume.Driver.Get
func (d *DeviceMapperDriver) Get(volumeName string) (volume.Volume, error) {
	glog.V(2).Infof("Get() (%s) START", volumeName)
	defer glog.V(2).Infof("Get() (%s) END", volumeName)
	glog.V(2).Infof("Getting devicemapper volume %s", volumeName)
	vol, err := d.getVolume(volumeName, false)
	if err != nil {
		glog.Errorf("Error getting devicemapper volume: %s", err)
		return nil, err
	}
	if mounted, _ := devmapper.Mounted(vol.Path()); !mounted {
		deviceHash := vol.deviceHash()
		mountpoint := vol.Path()
		label := vol.Tenant()

		if err := d.DeviceSet.MountDevice(deviceHash, mountpoint, label); err != nil {
			glog.Errorf("Error mounting device %q on %q for volume %q: %s", deviceHash, mountpoint, volumeName, err)
			return nil, err
		}
		glog.V(2).Infof("Mounted device %s to %s", deviceHash, mountpoint)
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
		_, err := d.getVolume(volname, false)
		if err != nil {
			glog.V(1).Infof("Unable to get volume %s; skipping", volname)
			continue
		}
		if err := d.Release(volname); err != nil {
			return err
		}
	}
	return d.DeviceSet.Shutdown()
}

// Release implements volume.Driver.Release
func (d *DeviceMapperDriver) Release(volumeName string) error {
	glog.V(2).Infof("Release() (%s) START", volumeName)
	defer glog.V(2).Infof("Release() (%s) END", volumeName)
	vol, err := d.getVolume(volumeName, false)
	if err != nil {
		return err
	}
	if mounted, _ := devmapper.Mounted(vol.Path()); mounted {
		glog.V(1).Infof("Unmounting %s", volumeName)
		if err := vol.unmount(); err != nil {
			glog.Errorf("Error whilst unmounting %s: %s", vol.path, err)
			return err
		}
	}
	deviceHashes := vol.Metadata.ListDevices()
	for _, deviceHash := range deviceHashes {
		if deviceHash == "" {
			// this can happen when all previously active devices have been deactivated
			continue
		}

		// Perversely, deactivateDevice() will not actually work unless the device is activated.
		// GetDeviceStatus() will both verify that device is valid, and it has the side-effect of activating
		// the device if the device is not active.
		if status, err := d.DeviceSet.GetDeviceStatus(deviceHash); err != nil {
			glog.Errorf("For volume %s, unable to get status for device %q: %s", volumeName, deviceHash, err)
			continue
		} else if status == nil {
			glog.V(2).Infof("For volume %s, no status available for device %q", volumeName, deviceHash)
		} else {
			glog.V(2).Infof("For volume %s, status for device %q: %v", volumeName, deviceHash, status)
		}

		glog.V(1).Infof("Deactivating device (%s)", deviceHash)
		d.DeviceSet.Lock()
		err := d.deactivateDevice(deviceHash)
		d.DeviceSet.Unlock()
		if err != nil {
			glog.Errorf("Error removing device %q for volume %s: %s", deviceHash, volumeName, err)
			return err
		}
		glog.V(2).Infof("Deactivated device")
	}
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
	v, err := d.getVolume(volumeName, false)
	if err != nil {
		//log the error, but continue trying to remove things
		glog.Errorf("Error loading volume %s: %s", volumeName, err)
	} else {
		// remove the snapshots
		glog.V(1).Infof("Removing snapshots from %s", volumeName)
		snapshots, err := v.Snapshots()
		if err != nil {
			glog.Errorf("Error getting list of snapshots for volume %s: %s", volumeName, err)
		} else {
			for _, snapshot := range snapshots {
				if err := v.RemoveSnapshot(snapshot); err != nil {
					glog.Errorf("Could not remove snapshot: %s", err)
				}
			}
		}

		// Release the device (requires another call to getVolume)
		if err := d.Release(volumeName); err != nil {
			glog.V(1).Infof("Error releasing device: %s", err)
		}

		// Delete the device
		glog.V(1).Infof("Removing volume %s", volumeName)
		if err := d.deleteDevice(v.deviceHash(), false); err != nil {
			glog.Errorf("Could not delete device %s: %s", volumeName, err)
		}
	}

	// Remove the metadata directory
	if err := os.RemoveAll(filepath.Join(d.MetadataDir(), volumeName)); err != nil {
		return err
	}
	return nil
}

// Issues the underlying dm remove operation.
func (d *DeviceMapperDriver) deactivateDevice(deviceHash string) error {
	glog.V(2).Infof("deactivateDevice START(%s)", deviceHash)
	defer glog.V(2).Infof("deactivateDevice END(%s)", deviceHash)

	var err error

	deviceName := d.deviceName(deviceHash)
	for i := 0; i < 200; i++ {
		err = devicemapper.RemoveDevice(deviceName)
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

// unmountDevice unmounts the device
func (d *DeviceMapperDriver) unmountDevice(deviceHash, mountpoint string) {

	// We use the provided UnmountDevice func here, rather than our own
	// unmount(), because we DO care about Docker's internal bookkeeping
	// here. Without this, DeviceSet.DeleteDevice will fail.
	if err := d.DeviceSet.UnmountDevice(deviceHash, mountpoint); err != nil {
		glog.V(2).Infof("Error unmounting %s (device: %s): %s", mountpoint, deviceHash, err)
	}
	d.DeviceSet.Lock()
	if err := d.deactivateDevice(deviceHash); err != nil {
		glog.V(2).Infof("Error deactivating device %s: %s", deviceHash, err)
	}
	d.DeviceSet.Unlock()
}

// readDeviceInfo loads the device info from metadata
func (d *DeviceMapperDriver) readDeviceInfo(deviceHash string, devInfo *devInfo) error {
	fh, err := os.Open(d.deviceInfoPath(deviceHash))
	if err != nil {
		glog.Errorf("Could not read device info for %s: %s", deviceHash, err)
		return err
	}
	defer fh.Close()
	if err := json.NewDecoder(fh).Decode(devInfo); err != nil {
		glog.Errorf("Could not decode device info for %s: %s", deviceHash, err)
		return err
	}
	return nil
}

// writeDeviceInfo performs an atomic write to the device metadata file
func (d *DeviceMapperDriver) writeDeviceInfo(deviceHash string, devInfo devInfo) error {
	devInfoPath := d.deviceInfoPath(deviceHash)
	fs, err := os.Stat(devInfoPath)
	if err != nil {
		glog.Errorf("Could not stat file %s: %s", devInfoPath, err)
		return err
	}
	data, err := json.Marshal(devInfo)
	if err != nil {
		glog.Errorf("Could not marshal info for device %s: %s", deviceHash, err)
		return err
	}
	if err := atomicfile.WriteFile(devInfoPath, data, fs.Mode()); err != nil {
		glog.Errorf("Could not update info for device %s: %s", deviceHash, err)
		return err
	}
	return nil
}

// resizeDevice sets the device size in its metadata property
func (d *DeviceMapperDriver) resizeDevice(deviceHash string, size uint64) error {
	var devInfo devInfo
	if err := d.readDeviceInfo(deviceHash, &devInfo); err != nil {
		glog.Errorf("Could not read info for device %s: %s", deviceHash, err)
		return err
	}
	devInfo.Size = size
	if err := d.writeDeviceInfo(deviceHash, devInfo); err != nil {
		glog.Errorf("Could not write info for device %s: %s", deviceHash, err)
		return err
	}
	return nil
}

func (d *DeviceMapperDriver) deviceInfoPath(deviceHash string) string {
	return filepath.Join(d.poolDir(), "metadata", deviceHash)
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
	if err := volume.TouchFlagFile(poolPath); err != nil {
		return err
	}
	if d.DeviceSet == nil {
		var (
			thinPoolDev         string
			enableLVMMonitoring string
			dmoptions           []string
		)
		for _, option := range d.options {
			if strings.HasPrefix(option, "dm.") {
				dmoptions = append(dmoptions, option)
				if strings.HasPrefix(option, "dm.thinpooldev=") {
					thinPoolDev = strings.TrimPrefix(option, "dm.thinpooldev=")
					thinPoolDev = fmt.Sprintf("/dev/mapper/%s", strings.TrimPrefix(thinPoolDev, "/dev/mapper/"))
				}
			} else if strings.HasPrefix(option, "enablelvmmonitoring=") {
				enableLVMMonitoring = strings.TrimPrefix(option, "enablelvmmonitoring=")
			} else {
				glog.Errorf("Unable to parse option %s", option)
				return ErrInvalidOption
			}
		}
		dmoptions = append(dmoptions, "dm.fs=ext4")
		deviceSet, err := devmapper.NewDeviceSet(poolPath, true, dmoptions, nil, nil)
		if err != nil {
			if _, thinError := err.(devmapper.ThinpoolInitError); thinError {
				//Try recreating the base image because sometimes something deletes it
				glog.Errorf("Error intializing thin pool device, %s, attempting to create to new base device", err)
				deviceSet, err = devmapper.NewDeviceSet(poolPath, false, d.options, nil, nil)
				if err != nil {
					return err
				}
				err = deviceSet.CreateBaseImage()
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		d.DeviceSet = deviceSet
		prefix, err := GetDevicePrefix(poolPath)
		if err != nil {
			return err
		}
		d.DevicePrefix = prefix
		switch enableLVMMonitoring {
		case "y":
			if thinPoolDev != "" {
				glog.V(1).Infof("Enabling LVM Monitoring for thin pool device %s", thinPoolDev)
				cmd := exec.Command("lvchange", "--monitor", "y", thinPoolDev)
				output, err := cmd.CombinedOutput()
				if err != nil {
					glog.Errorf("Could not run command %v: %s (%s)", cmd, string(output), err)
					return err
				}
			} else {
				glog.Warningf("Ignoring option 'enablelvmmonitoring'; no thin pool device specified")
			}
		case "n", "":
			if thinPoolDev != "" {
				glog.V(1).Infof("Disabling LVM Monitoring for thin pool device %s", thinPoolDev)
				cmd := exec.Command("lvchange", "--monitor", "n", thinPoolDev)
				output, err := cmd.CombinedOutput()
				if err != nil {
					glog.Errorf("Could not run command %v: %s (%s)", cmd, string(output), err)
					return err
				}
			} else if enableLVMMonitoring != "" {
				glog.Warningf("Ignoring option 'enablelvmmonitoring'; no thin pool device specified")
			}
		default:
			glog.Errorf("Attribute %s is not defined for enablelvmmonitoring", enableLVMMonitoring)
			return ErrInvalidArg
		}
	}
	if err := os.MkdirAll(d.MetadataDir(), 0755); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (d *DeviceMapperDriver) deleteDevice(deviceName string, syncDelete bool) error {
	if deviceName == "base" || deviceName == "" {
		glog.Errorf("Request to delete base device '%s' will not be honored", deviceName)
		return ErrDeleteBaseDevice
	}
	return d.DeviceSet.DeleteDevice(deviceName, syncDelete)
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
	if v.isInvalidSnapshot(label) {
		return nil, volume.ErrInvalidSnapshot
	}
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
	// check the tags for duplicates
	for _, tagName := range tags {
		if tagInfo, err := v.GetSnapshotWithTag(tagName); err != volume.ErrSnapshotDoesNotExist {
			if err != nil {
				glog.Errorf("Could not look up snapshot for tag %s: %s", tagName, err)
				return err
			} else {
				glog.Errorf("Tag '%s' is already in use by snapshot %s", tagName, tagInfo.Name)
				return volume.ErrTagAlreadyExists
			}
		}
	}
	label = v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()
	// Create a new device based on the current one
	activeDeviceHash := v.deviceHash()
	snapshotDeviceHash, err := v.driver.addDevice(activeDeviceHash)
	if err != nil {
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
	glog.V(2).Infof("Saving device %s as snapshot %s", snapshotDeviceHash, label)
	if err := v.Metadata.AddSnapshot(label, snapshotDeviceHash); err != nil {
		glog.Errorf("Unable to save snapshot metadata: %s", err)
		return err
	}
	return nil
}

// TagSnapshot implements volume.Volume.TagSnapshot
func (v *DeviceMapperVolume) TagSnapshot(label string, tagName string) error {
	v.Lock()
	defer v.Unlock()
	// get the snapshot
	info, err := v.SnapshotInfo(label)
	if err != nil {
		glog.Errorf("Could not look up snapshot %s: %s", label, err)
		return err
	}
	// verify the tag doesn't already exist
	if tagInfo, err := v.GetSnapshotWithTag(tagName); err != volume.ErrSnapshotDoesNotExist {
		if err != nil {
			glog.Errorf("Could not look up snapshot for tag %s: %s", tagName, err)
			return err
		} else {
			glog.Errorf("Tag '%s' is already in use by snapshot %s", tagName, tagInfo.Name)
			return volume.ErrTagAlreadyExists
		}
	}
	// add the tag and update the snapshot
	info.Tags = append(info.Tags, tagName)
	if err := v.writeSnapshotInfo(info.Label, info); err != nil {
		glog.Errorf("Could not update tags for snapshot %s: %s", info.Label, err)
		return err
	}
	return nil
}

// UntagSnapshot implements volume.Volume.UntagSnapshot
func (v *DeviceMapperVolume) UntagSnapshot(tagName string) (string, error) {
	v.Lock()
	defer v.Unlock()
	// find the snapshot with the provided tag
	info, err := v.GetSnapshotWithTag(tagName)
	if err != nil {
		glog.Errorf("Could not find snapshot with tag %s: %s", tagName, err)
		return "", err
	}
	// remove the tag and update the snapshot
	var tags []string
	for _, tag := range info.Tags {
		if tag != tagName {
			tags = append(tags, tag)
		}
	}
	info.Tags = tags
	if err := v.writeSnapshotInfo(info.Label, info); err != nil {
		glog.Errorf("Could not remove tag '%s' from snapshot %s: %s", tagName, info.Name, err)
		return "", err
	}
	return info.Label, err
}

// GetSnapshotWithTag implements volume.Volume.GetSnapshotWithTag
func (v *DeviceMapperVolume) GetSnapshotWithTag(tagName string) (*volume.SnapshotInfo, error) {
	// Get all the snapshots on the volume
	snapshotLabels, err := v.Snapshots()
	if err != nil {
		glog.Errorf("Could not get current snapshot list: %s", err)
		return nil, err
	}
	// Get info for each snapshot and return if a matching tag is found
	for _, snapshotLabel := range snapshotLabels {
		if info, err := v.SnapshotInfo(snapshotLabel); err != volume.ErrInvalidSnapshot {
			if err != nil {
				glog.Errorf("Could not get info for snapshot %s: %s", snapshotLabel, err)
				return nil, err
			}
			for _, tag := range info.Tags {
				if tag == tagName {
					return info, nil
				}
			}
		}
	}
	return nil, volume.ErrSnapshotDoesNotExist
}

// Snapshots implements volume.Volume.Snapshots
func (v *DeviceMapperVolume) Snapshots() ([]string, error) {
	return v.Metadata.ListSnapshots(), nil
}

// isInvalidSnapshot checks to see if the snapshot is missing a .SNAPSHOTINFO file
func (v *DeviceMapperVolume) isInvalidSnapshot(rawLabel string) bool {
	reader, err := v.ReadMetadata(rawLabel, ".SNAPSHOTINFO")
	if err != nil {
		return true
	}
	reader.Close()
	return false
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
	deviceHash, err := v.Metadata.LookupSnapshotDevice(rawLabel)
	if err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}

	// Delete the device itself first, so we retain our internal snapshot entry
	// (metadata.json) in the event of a failure on this step
	glog.V(2).Infof("Deactivating snapshot device %s", deviceHash)
	v.driver.DeviceSet.Lock()
	if err := v.driver.deactivateDevice(deviceHash); err != nil {
		glog.V(2).Infof("Error deactivating device (%s): %s", deviceHash, err)
	}
	v.driver.DeviceSet.Unlock()
	if err := v.driver.deleteDevice(deviceHash, false); err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}

	// Remove the snapshot info from the volume metadata
	if err := v.Metadata.RemoveSnapshot(rawLabel); err != nil {
		glog.Errorf("Error removing snapshot: %v", err)
		return volume.ErrRemovingSnapshot
	}
	// Remove the snapshot-specific metadata directory
	if err := os.RemoveAll(filepath.Join(v.driver.MetadataDir(), rawLabel)); err != nil {
		return err
	}
	return nil
}

// Rollback implements volume.Volume.Rollback
func (v *DeviceMapperVolume) Rollback(label string) error {
	if v.isInvalidSnapshot(label) {
		return volume.ErrInvalidSnapshot
	}

	glog.V(2).Infof("Rollback() (%s) START", v.name)
	defer glog.V(2).Infof("Rollback() (%s) END", v.name)
	if !v.snapshotExists(label) {
		return volume.ErrSnapshotDoesNotExist
	}
	label = v.rawSnapshotLabel(label)
	v.Lock()
	defer v.Unlock()

	// Get the hash of the current device and of the snapshot
	curDeviceHash := v.deviceHash()
	snapDeviceHash, err := v.Metadata.LookupSnapshotDevice(label)
	if err != nil {
		return err
	}

	// Make a new device for the snapshot to load
	newDeviceHash, err := v.driver.addDevice(snapDeviceHash)
	if err != nil {
		return err
	}
	glog.V(2).Infof("Created new head device %s based on snapshot %s", newDeviceHash, snapDeviceHash)

	// Unmount the current device and mount the new one
	if err := v.unmount(); err != nil {
		return err
	}
	glog.V(2).Infof("Unmounted old head device %s", curDeviceHash)

	// Rollback to the new device
	if err := v.driver.DeviceSet.MountDevice(newDeviceHash, v.path, v.name); err != nil {
		return err
	}
	glog.V(2).Infof("Rollback(): mounting new head device %s", newDeviceHash)

	// Clean up the old device
	v.driver.DeviceSet.Lock()
	if err := v.driver.deactivateDevice(curDeviceHash); err != nil {
		glog.V(2).Infof("Could not remove device: %s", err)
	} else {
		glog.V(2).Infof("Deactivated old head device %s", curDeviceHash)
	}
	v.driver.DeviceSet.Unlock()
	if err := v.driver.deleteDevice(curDeviceHash, false); err != nil {
		glog.Warningf("Error cleaning up old head device %s: %s", curDeviceHash, err)
	} else {
		glog.V(2).Infof("Deleted old head device %s", curDeviceHash)
	}

	return v.Metadata.SetCurrentDevice(newDeviceHash)
}

// Export implements volume.Volume.Export
func (v *DeviceMapperVolume) Export(label, parent string, writer io.Writer, excludes []string) error {
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
	defer os.RemoveAll(mountpoint)
	deviceHash, err := v.Metadata.LookupSnapshotDevice(label)
	if err != nil {
		return err
	}
	glog.V(2).Infof("Mounting temporary export device %s", deviceHash)
	if err := v.driver.DeviceSet.MountDevice(deviceHash, mountpoint, label); err != nil {
		return err
	}
	defer func(d *DeviceMapperDriver, deviceHash, mountpoint string) {
		// We use the provided UnmountDevice func here, rather than our own
		// unmount(), because we DO care about Docker's internal bookkeeping
		// here. Without this, DeviceSet.DeleteDevice will fail.
		if err := d.DeviceSet.UnmountDevice(deviceHash, mountpoint); err != nil {
			glog.V(2).Infof("Error unmounting %s (device: %s): %s", mountpoint, deviceHash, err)
		}
		d.DeviceSet.Lock()
		if err := d.deactivateDevice(deviceHash); err != nil {
			glog.V(2).Infof("Error deactivating device %s: %s", deviceHash, err)
		}
		d.DeviceSet.Unlock()
	}(v.driver, deviceHash, mountpoint)

	tarOut := tar.NewWriter(writer)

	// Set the driver type
	drivertype := []byte(v.Driver().DriverType())
	header := &tar.Header{Name: fmt.Sprintf("%s-driver", label), Size: int64(len(drivertype))}
	if err := tarOut.WriteHeader(header); err != nil {
		glog.Errorf("Could not export driver type header for snapshot %s: %s", label, err)
		return err
	}
	if _, err := tarOut.Write(drivertype); err != nil {
		glog.Errorf("Could not export driver type for snapshot %s: %s", label, err)
		return err
	}

	// Set the device metadata
	size, err := v.SizeOf()
	if err != nil {
		glog.Errorf("Could not get the size of the device for snapshot %s: %s", label, err)
		return err
	}
	volInfo, err := json.Marshal(volumeInfo{Size: size})
	if err != nil {
		glog.Errorf("Could not marshal device info for snapshot %s: %s", label, err)
		return err
	}
	header = &tar.Header{Name: fmt.Sprintf("%s-device", label), Size: int64(len(volInfo))}
	if err := tarOut.WriteHeader(header); err != nil {
		glog.Errorf("Could not export device info header for snapshot %s: %s", label, err)
		return err
	}
	if _, err := tarOut.Write(volInfo); err != nil {
		glog.Errorf("Could not export device info for snapshot %s: %s", label, err)
		return err
	}

	// Write metadata
	mdpath := filepath.Join(v.driver.MetadataDir(), label)

	if err := exportDirectoryAsTar(mdpath, fmt.Sprintf("%s-metadata", label), tarOut, []string{}); err != nil {
		return err
	}
	if err := exportDirectoryAsTar(mountpoint, fmt.Sprintf("%s-volume", label), tarOut, excludes); err != nil {
		return err
	}

	return tarOut.Close()
}

func (d *DeviceMapperDriver) Status() (volume.Status, error) {
	glog.V(2).Info("devicemapper.Status()")
	dockerStatus := d.DeviceSet.Status()
	tss, err := d.GetTenantStorageStats()
	if err != nil {
		return nil, err
	}
	driverType := "direct-lvm"
	if dockerStatus.DataLoopback != "" {
		driverType = "loop-lvm"
	}
	usageData := []volume.Usage{
		// Store under older value names in case anybody's looking for it
		{Label: "Data", Type: "Available", Value: dockerStatus.Data.Available,
			MetricName: "storage.available"},
		{Label: "Data", Type: "Used", Value: dockerStatus.Data.Used,
			MetricName: "storage.used"},
		{Label: "Data", Type: "Total", Value: dockerStatus.Data.Total,
			MetricName: "storage.total"},
		// Now store under useful names
		{Value: dockerStatus.Data.Available, MetricName: "storage.pool.data.available"},
		{Value: dockerStatus.Data.Used, MetricName: "storage.pool.data.used"},
		{Value: dockerStatus.Data.Total, MetricName: "storage.pool.data.total"},
		{Value: dockerStatus.Metadata.Available, MetricName: "storage.pool.metadata.available"},
		{Value: dockerStatus.Metadata.Used, MetricName: "storage.pool.metadata.used"},
		{Value: dockerStatus.Metadata.Total, MetricName: "storage.pool.metadata.total"},
	}

	// Disabled due to CC-2417
	//var unallocated uint64

	// Add in tenant storage metrics
	for _, tenant := range tss {
		usageData = append(usageData, []volume.Usage{
			{MetricName: fmt.Sprintf("storage.filesystem.total.%s", tenant.TenantID),
				Value: tenant.FilesystemTotal},
			{MetricName: fmt.Sprintf("storage.filesystem.available.%s", tenant.TenantID),
				Value: tenant.FilesystemAvailable},
			{MetricName: fmt.Sprintf("storage.filesystem.used.%s", tenant.TenantID),
				Value: tenant.FilesystemUsed},
			{MetricName: fmt.Sprintf("storage.device.total.%s", tenant.TenantID),
				Value: tenant.DeviceTotalBlocks},
			/* Disabled due to CC-2417
			{MetricName: fmt.Sprintf("storage.device.allocated.%s", tenant.TenantID),
				Value: tenant.DeviceAllocatedBlocks},
			{MetricName: fmt.Sprintf("storage.snapshot.allocated.%s", tenant.TenantID),
				Value: tenant.SnapshotAllocatedBlocks},
			*/
			{MetricName: fmt.Sprintf("storage.snapshot.count.%s", tenant.TenantID),
				Value: uint64(tenant.NumberSnapshots)},
		}...)
		// Disabled due to CC-2417
		//unallocated += tenant.DeviceUnallocatedBlocks
	}

	// convert dockerStatus to our status and return
	result := &volume.DeviceMapperStatus{
		Driver:     volume.DriverTypeDeviceMapper,
		DriverType: driverType,
		DriverPath: d.root,
		PoolName:   dockerStatus.PoolName,

		PoolDataTotal:     dockerStatus.Data.Total,
		PoolDataAvailable: dockerStatus.Data.Available,
		PoolDataUsed:      dockerStatus.Data.Used,

		PoolMetadataTotal:     dockerStatus.Metadata.Total,
		PoolMetadataAvailable: dockerStatus.Metadata.Available,
		PoolMetadataUsed:      dockerStatus.Metadata.Used,

		UsageData: usageData,
		Tenants:   tss,
	}

	/* Disabled due to CC-2417
	if unallocated > volume.BytesToBlocks(dockerStatus.Data.Available) {
		overage := volume.BlocksToBytes(unallocated - volume.BytesToBlocks(dockerStatus.Data.Available))
		result.Errors = append(result.Errors, fmt.Sprintf(`!!!	Warning: your thin pool is currently oversubscribed by %s. You should
	enlarge it by at least %s using LVM tools and/or delete some snapshots.`, overage, overage))
	}
	*/

	return result, nil
}

// GetTenantStorageStats returns storage stats for each tenant in the system.
func (d *DeviceMapperDriver) GetTenantStorageStats() ([]volume.TenantStorageStats, error) {
	var result []volume.TenantStorageStats
	status := d.DeviceSet.Status()
	// If this is loop-lvm, the metadata device will be in status.MetadataFile
	mdDevice := status.MetadataFile
	if mdDevice == "" {
		// It's direct-lvm, so build the metadata device from the pool name
		mdDevice = fmt.Sprintf("/dev/mapper/%s_tmeta", status.PoolName)
	}
	// CC-2418: Disabling block stats gathering until a kernel bug in dm-thin
	// is fixed
	/*
		blockstats, err := d.getDeviceBlockStats(status.PoolName, mdDevice)
		if err != nil {
			return nil, err
		}
	*/
	for _, tenant := range d.ListTenants() {
		var devInfo devInfo
		vol, err := d.getVolume(tenant, false)
		if err != nil {
			return nil, err
		}
		if err := d.readDeviceInfo(vol.Metadata.CurrentDevice(), &devInfo); err != nil {
			return nil, err
		}
		// CC-2417
		//if stats, ok := blockstats[devInfo.DeviceID]; ok {
		tss := volume.TenantStorageStats{TenantID: tenant, VolumePath: vol.Path()}
		// CC-2417
		//tss.DeviceAllocatedBlocks = stats.diet.Total()
		tss.NumberSnapshots = len(vol.Metadata.snapshotMetadata.Snapshots)
		dev := vol.Metadata.CurrentDevice()
		// This will activate the device
		if _, err := d.DeviceSet.GetDeviceStatus(dev); err != nil {
			return nil, err
		}
		devicename := fmt.Sprintf("/dev/mapper/%s-%s", d.DevicePrefix, dev)
		total, free, err := getFilesystemStats(devicename)
		if err != nil {
			return nil, err
		}
		size, err := getDeviceSize(devicename)
		if err != nil {
			return nil, err
		}
		tss.FilesystemTotal = total
		tss.FilesystemAvailable = free
		tss.FilesystemUsed = total - free
		tss.DeviceTotalBlocks = volume.BytesToBlocks(size)
		//tss.DeviceUnallocatedBlocks = tss.DeviceTotalBlocks - tss.DeviceAllocatedBlocks
		/* CC-2417
				last := stats
				for _, device := range vol.Metadata.snapshotMetadata.Snapshots {
					if err := d.readDeviceInfo(device, &devInfo); err != nil {
						return nil, err
					}
					if snapstats, ok := blockstats[devInfo.DeviceID]; ok {
						tss.SnapshotAllocatedBlocks += snapstats.UniqueBlocks(last)
						last = snapstats
					}
				}
				/*
				if volume.BytesToBlocks(tss.FilesystemUsed) < tss.DeviceAllocatedBlocks {
					tss.Errors = append(tss.Errors, fmt.Sprintf(` !	Note: %s of blocks are allocated to an application virtual device but
		are unused by the filesystem. This is not a problem; however, if you want
		the thin pool to reclaim the space for use by snapshots or another
		application, run:

			$ fstrim %s`, volume.BlocksToBytes(tss.DeviceAllocatedBlocks-volume.BytesToBlocks(tss.FilesystemUsed)), vol.Path()))
				}
		*/
		result = append(result, tss)
		/* CC-2417
		} else {
			// There is no device matching the tenant
			return nil, fmt.Errorf("Tenant %s DFS has not yet been initialized", tenant)
		}
		*/
	}
	return result, nil
}

// Import implements volume.Volume.Import
func (v *DeviceMapperVolume) Import(label string, reader io.Reader) (err error) {
	glog.V(2).Infof("Import() (%s) START", v.name)
	defer glog.V(2).Infof("Import() (%s) END", v.name)

	// Do not try to import if the snapshot label exists on the volume.
	if v.snapshotExists(label) {
		return volume.ErrSnapshotExists
	}

	// Ensure that we are getting the full snapshot label and not just the
	// suffix.
	label = v.rawSnapshotLabel(label)

	// Set up the mountpoint to load the snapshot
	mountpoint, err := ioutil.TempDir(v.driver.Root(), label+"_import-")
	if err != nil {
		glog.Errorf("Could not create mountpoint to import snapshot %s: %s", label, err)
		return err
	}
	glog.V(2).Infof("Created mountpoint at %s for snapshot %s", mountpoint, label)
	defer os.RemoveAll(mountpoint)

	// Set up the staging device for the snapshot
	deviceHash, err := v.driver.addDevice("")
	if err != nil {
		glog.Errorf("Unable to create staging device at mountpoint %s for snapshot %s: %s", mountpoint, label, err)
		return err
	}
	glog.V(2).Infof("Created a staging device %s for snapshot %s", deviceHash, label)
	defer func() {
		if err != nil {
			v.driver.deleteDevice(deviceHash, false)
		}
	}()

	// Set up the metadata directory
	metaPath := filepath.Join(v.driver.MetadataDir(), filepath.Base(mountpoint))
	if err = os.MkdirAll(metaPath, 0755); err != nil {
		glog.Errorf("Could not create metadata path for snapshot %s at %s: %s", label, metaPath, err)
		return err
	}
	glog.V(2).Infof("Created metadata path %s for snapshot %s", metaPath, label)
	defer func() {
		if err != nil {
			os.RemoveAll(metaPath)
		}
	}()

	if err = v.loadSnapshotImport(reader, label, deviceHash, mountpoint, metaPath); err != nil {
		return err
	}
	glog.V(2).Infof("Successfully loaded snapshot %s", label)

	return nil
}

// loadSnapshotImport loads a volume from a reader to the provided mointpoint
func (v *DeviceMapperVolume) loadSnapshotImport(reader io.Reader, label, deviceHash, mountpoint, metaPath string) error {

	// Mount the staging device
	if err := v.driver.DeviceSet.MountDevice(deviceHash, mountpoint, label+"_import"); err != nil {
		glog.Errorf("Could not stage the device %s at mountpoint %s for snapshot %s: %s", deviceHash, mountpoint, label, err)
		return err
	}
	glog.V(2).Infof("Mounted staging device %s at mountpoint %s for snapshot %s", deviceHash, mountpoint, label)
	defer func(d *DeviceMapperDriver, deviceHash, mountpoint string) {
		// We use the provided UnmountDevice func here, rather than our own
		// unmount(), because we DO care about Docker's internal bookkeeping
		// here. Without this, DeviceSet.DeleteDevice will fail.
		if err := d.DeviceSet.UnmountDevice(deviceHash, mountpoint); err != nil {
			glog.V(2).Infof("Error unmounting %s (device: %s): %s", mountpoint, deviceHash, err)
		}
		d.DeviceSet.Lock()
		if err := d.deactivateDevice(deviceHash); err != nil {
			glog.V(2).Infof("Error deactivating device %s: %s", deviceHash, err)
		}
		d.DeviceSet.Unlock()
	}(v.driver, deviceHash, mountpoint)

	// Read the volume and metadata from the stream and write to disk
	var (
		driverFile = label + "-driver"   // Filesystem type of export volume
		deviceFile = label + "-device"   // Information about the device (if available)
		volumeDir  = label + "-volume"   // Volume data
		metaDir    = label + "-metadata" // Metadata
	)
	driverType := ""
	tarfile := tar.NewReader(reader)
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			if driverType == "" {
				return ErrIncompatibleSnapshot
			}
			break
		} else if err != nil {
			glog.Errorf("Could not import archive for label %s: %s", label, err)
			return err
		}
		if header.Name == driverFile {

			// Get the driver type of the volume and figure out if it is a
			// valid driver.
			bfr := &bytes.Buffer{}
			if _, err := bfr.ReadFrom(tarfile); err != nil {
				glog.Errorf("Could not read from %s: %s", driverFile, err)
				return err
			}
			driverType = bfr.String()
			glog.V(2).Infof("Volume driver type is %s", driverType)
		} else if header.Name == deviceFile {

			// Update the staging device to match the provided settings
			volInfo := volumeInfo{}
			if err := json.NewDecoder(tarfile).Decode(&volInfo); err != nil {
				glog.Errorf("Could not read from %s: %s", deviceFile, err)
				return err
			}

			// Ensure the size of the device is greater than or equal to the
			// value specified in the settings.
			size, err := v.driver.deviceSize(deviceHash)
			if err != nil {
				glog.Errorf("Could not determine the size of the staging device %s: %s", deviceHash, err)
				return err
			}
			if volInfo.Size > size {
				glog.Warning("Device size from import %s is greater than size of staging device; expanding.", label, deviceHash)
				if err := v.driver.resize(deviceHash, volInfo.Size); err != nil {
					glog.Errorf("Could not resize device %s; not importing snapshot %s: %s", deviceHash, label, err)
					return err
				}
				glog.V(2).Infof("Device %s is now %s", deviceHash, units.HumanSize(float64(volInfo.Size)))
			}
		} else if strings.HasPrefix(header.Name, volumeDir) {

			// Untar into mountpoint
			header.Name = strings.TrimPrefix(header.Name, volumeDir)
			if err := volume.ImportArchiveHeader(header, tarfile, mountpoint); err != nil {
				return err
			}
		} else if strings.HasPrefix(header.Name, metaDir) {

			// Untar into the metadata
			header.Name = strings.TrimPrefix(header.Name, metaDir)
			if err := volume.ImportArchiveHeader(header, tarfile, metaPath); err != nil {
				return err
			}
		}
	}

	// Add device as a snapshot of this volume.
	if err := v.Metadata.AddSnapshot(label, deviceHash); err != nil {
		glog.Errorf("Could not save device %s as snapshot %s: %s", deviceHash, label, err)
		return err
	}

	// Set the snapshot metadata path
	if err := os.Rename(metaPath, filepath.Join(v.driver.MetadataDir(), label)); err != nil {
		glog.Errorf("Could not set device metadata for snapshot %s: %s", label, err)
		return err
	}

	return nil
}

func (v *DeviceMapperVolume) SizeOf() (uint64, error) {
	return v.driver.deviceSize(v.deviceHash())
}

func (v *DeviceMapperVolume) deviceHash() string {
	return v.Metadata.CurrentDevice()
}

// rawSnapshotLabel ensures that <label> has the tenant prefix for this volume
func (v *DeviceMapperVolume) rawSnapshotLabel(label string) string {
	return volume.DefaultSnapshotLabel(v.Tenant(), label)
}

func (v *DeviceMapperVolume) snapshotExists(label string) bool {
	label = v.rawSnapshotLabel(label)
	return v.Metadata.SnapshotExists(label)
}

func getfstype(dmDevice string) (fstype []byte, err error) {
	output, err := exec.Command("blkid", "-s", "TYPE", dmDevice).CombinedOutput()
	if err != nil {
		glog.Errorf("Could not get the device type of %s: %s (%s)", dmDevice, string(output), err)
		return nil, err
	}
	if fields := bytes.Fields(output); len(fields) == 2 {
		if bytes.HasPrefix(fields[1], []byte("TYPE=")) {
			value := bytes.TrimPrefix(fields[1], []byte("TYPE="))
			fstype = bytes.Trim(value, "\"")
			return
		}
	}
	// This should not happen
	return nil, errors.New("invalid output")
}

func resize2fs(dmDevice string) error {
	fstype, err := getfstype(dmDevice)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch string(fstype) {
	case "xfs":
		glog.Infof("Device type: %s, Command: xfs_growfs %s", fstype, dmDevice)
		cmd = exec.Command("xfs_growfs", dmDevice)
	default:
		glog.Infof("Device type: %s, Command: resize2fs %s", fstype, dmDevice)
		cmd = exec.Command("resize2fs", dmDevice)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Could not resize filesystem for device %s: %s (%s)", dmDevice, string(output), err)
		return err
	}
	return nil
}

func exportDirectoryAsTar(path, prefix string, out *tar.Writer, excludes []string) error {
	cmdString := []string{"-C", path, "-cf", "-", "--transform", fmt.Sprintf("s,^,%s/,", prefix)}
	for _, excludeDir := range excludes {
		cmdString = append(cmdString, []string{"--exclude", excludeDir, "--exclude", fmt.Sprintf(".%s.serviced.initialized", excludeDir)}...)
	}
	cmdString = append(cmdString, ".")
	cmd := exec.Command("tar", cmdString...)
	defer cmd.Wait()
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer pipe.Close()
	if err := cmd.Start(); err != nil {
		return err
	}
	tarReader := tar.NewReader(pipe)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := out.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.Copy(out, tarReader); err != nil {
			return err
		}
	}
	return nil
}
