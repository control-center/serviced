// Copyright 2015 The Serviced Authors.
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

// +build integration

package drivertest

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/volume"
	dm "github.com/control-center/serviced/volume/devicemapper"
	. "gopkg.in/check.v1"

	dockerclient "github.com/fsouza/go-dockerclient"
)

var (
	drv volume.Driver
)

type Driver struct {
	volume.Driver
	// Keep a reference to the root here just in case something below doesn't work
	root string
}

func newDriver(c *C, name volume.DriverType, root string, args []string) *Driver {
	var err error
	if root == "" {
		root = c.MkDir()
	}
	if err := volume.InitDriver(name, root, args); err != nil {
		c.Logf("drivertest: %v", err)
		if err == volume.ErrDriverNotSupported {
			c.Skip("Driver not supported")
		}
		c.Fatal(err)
	}
	d, err := volume.GetDriver(root)
	c.Assert(err, IsNil)
	c.Assert(d, NotNil)
	c.Assert(d.DriverType(), Equals, name)
	c.Assert(d.Root(), Equals, root)
	return &Driver{d, root}
}

func cleanup(c *C, d *Driver) {
	c.Check(d.Cleanup(), IsNil)
	os.RemoveAll(d.root)
}

func verifyFile(c *C, path string, mode os.FileMode, uid, gid uint32) {
	fi, err := os.Stat(path)
	c.Assert(err, IsNil)
	c.Check(fi.Mode()&os.ModeType, Equals, mode&os.ModeType)
	c.Check(fi.Mode()&os.ModePerm, Equals, mode&os.ModePerm)
	c.Check(fi.Mode()&os.ModeSticky, Equals, mode&os.ModeSticky)
	c.Check(fi.Mode()&os.ModeSetuid, Equals, mode&os.ModeSetuid)
	c.Check(fi.Mode()&os.ModeSetgid, Equals, mode&os.ModeSetgid)
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		c.Check(stat.Uid, Equals, uid)
		c.Check(stat.Gid, Equals, gid)
	}
}

func arrayContains(array []string, element string) bool {
	for _, x := range array {
		if x == element {
			return true
		}
	}
	return false
}

// filter out the lost+found directory created on ext4 filesystems
// filter out the .SNAPSHOTINFO file that gets created when a new snapshot is
// taken
func filterExtraFiles(fis []os.FileInfo) (filtered []os.FileInfo) {
	for _, fi := range fis {
		switch fi.Name() {
		case "lost+found", ".SNAPSHOTINFO":
		default:
			filtered = append(filtered, fi)
		}
	}
	return
}

// DriverTestCreateEmpty verifies that a driver can create a volume, and that
// is is empty (and owned by the current user) after creation.
func DriverTestCreateEmpty(c *C, drivername volume.DriverType, root string, args []string) {
	driver := newDriver(c, drivername, root, args)
	defer cleanup(c, driver)

	c.Assert(driver.DriverType(), Equals, drivername)

	volumeName := "empty"

	_, err := driver.Create(volumeName)
	c.Assert(err, IsNil)
	c.Assert(driver.Exists(volumeName), Equals, true)
	c.Assert(arrayContains(driver.List(), volumeName), Equals, true)
	vol, err := driver.Get(volumeName)
	c.Assert(err, IsNil)
	verifyFile(c, vol.Path(), 0755|os.ModeDir, uint32(os.Getuid()), uint32(os.Getgid()))
	fis, err := ioutil.ReadDir(vol.Path())
	c.Assert(err, IsNil)
	fis = filterExtraFiles(fis)
	c.Assert(fis, HasLen, 0)
	vol2, err := driver.GetTenant(volumeName)
	c.Assert(err, IsNil)
	verifyFile(c, vol2.Path(), 0755|os.ModeDir, uint32(os.Getuid()), uint32(os.Getgid()))
	fis, err = ioutil.ReadDir(vol2.Path())
	c.Assert(err, IsNil)
	fis = filterExtraFiles(fis)
	c.Assert(fis, HasLen, 0)

	driver.Release(volumeName)
	c.Assert(driver.Remove(volumeName), IsNil)
	c.Assert(driver.Exists(volumeName), Equals, false)
}

func createBase(c *C, driver *Driver, name string) volume.Volume {
	// We need to be able to set any perms
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	_, err := driver.Create(name)
	c.Assert(err, IsNil)

	volume, err := driver.Get(name)
	c.Assert(err, IsNil)

	subdir := path.Join(volume.Path(), "a subdir")
	err = os.Mkdir(subdir, 0705|os.ModeSticky)
	c.Assert(err, IsNil)
	err = os.Chown(subdir, 1, 2)
	c.Assert(err, IsNil)

	file := path.Join(volume.Path(), "a file")
	err = ioutil.WriteFile(file, []byte("Some data"), 0222|os.ModeSetuid)
	c.Assert(err, IsNil)
	return volume
}

func writeExtra(c *C, driver *Driver, vol volume.Volume, filename string) {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)
	file := path.Join(vol.Path(), filename)
	err := ioutil.WriteFile(file, []byte("more data"), 0222|os.ModeSetuid)
	c.Assert(err, IsNil)
}

func checkBase(c *C, driver *Driver, vol volume.Volume) {
	subdir := path.Join(vol.Path(), "a subdir")
	verifyFile(c, subdir, 0705|os.ModeDir|os.ModeSticky, 1, 2)

	file := path.Join(vol.Path(), "a file")
	verifyFile(c, file, 0222|os.ModeSetuid, 0, 0)
}

func verifyBase(c *C, driver *Driver, vol volume.Volume) {
	checkBase(c, driver, vol)
	fis, err := ioutil.ReadDir(vol.Path())
	c.Assert(err, IsNil)
	fis = filterExtraFiles(fis)
	c.Assert(fis, HasLen, 2)
}

func verifyBaseWithExtra(c *C, driver *Driver, vol volume.Volume) {
	checkBase(c, driver, vol)

	file := path.Join(vol.Path(), "differentfile")
	verifyFile(c, file, 0222|os.ModeSetuid, 0, 0)

	fis, err := ioutil.ReadDir(vol.Path())
	c.Assert(err, IsNil)
	fis = filterExtraFiles(fis)
	c.Assert(fis, HasLen, 3)
}

func DriverTestCreateBase(c *C, drivername volume.DriverType, root string, args []string) {
	driver := newDriver(c, drivername, root, args)
	root = driver.Root()
	defer cleanup(c, driver)

	vol := createBase(c, driver, "Base")
	verifyBase(c, driver, vol)

	err := driver.Release(vol.Name())
	c.Assert(err, IsNil)

	// Remount and make sure everything's ok
	vol2, err := volume.Mount("Base", root)
	c.Assert(err, IsNil)
	verifyBase(c, driver, vol2)
	c.Assert(driver.Remove("Base"), IsNil)
	c.Assert(driver.Exists("Base"), Equals, false)
}

func DriverTestSnapshots(c *C, drivername volume.DriverType, root string, args []string) {
	driver := newDriver(c, drivername, root, args)
	defer cleanup(c, driver)

	vol := createBase(c, driver, "Base")
	verifyBase(c, driver, vol)

	// Set some metadata on the snapshot
	wmetadata := []byte("snap-metadata")
	wHandle, err := vol.WriteMetadata("Snap", "lost+found/metadata")
	c.Assert(err, IsNil)
	c.Assert(wHandle, NotNil)
	n, err := wHandle.Write(wmetadata)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(wmetadata))
	err = wHandle.Close()
	c.Assert(err, IsNil)

	// Snapshot the verified base to produce a new volume
	err = vol.Snapshot("Snap", "snapshot-message-0", []string{"SnapTag", "tagA"})
	c.Assert(err, IsNil)

	snaps, err := vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(arrayContains(snaps, "Base_Snap"), Equals, true)

	info, err := vol.SnapshotInfo("Base_Snap")
	c.Assert(err, IsNil)
	c.Assert(info, NotNil)
	c.Check(info.Name, Equals, "Base_Snap")
	c.Check(info.Label, Equals, "Snap")
	c.Check(info.TenantID, Equals, "Base")
	c.Check(info.Message, Equals, "snapshot-message-0")
	c.Check(info.Tags, DeepEquals, []string{"SnapTag", "tagA"})

	// Get the tenant volume of a snapshot that doesn't exist
	tvol, err := driver.GetTenant("Base_Snap2")
	c.Assert(err, Equals, volume.ErrVolumeNotExists)
	c.Assert(tvol, IsNil)

	// Write another file to the active volume
	writeExtra(c, driver, vol, "differentfile")

	// Re-snapshot the active volume with the extra file on it
	err = vol.Snapshot("Snap2", "snapshot-message-1", []string{"Snap2Tag", "tag1", "tag2", "tag3"})
	c.Assert(err, IsNil)

	// Get the tenant volume of the snapshot
	tvol, err = driver.GetTenant("Base_Snap2")
	c.Assert(err, IsNil)
	c.Assert(tvol.Name(), Equals, "Base")

	// Make sure the metadata path exists for the snapshot
	rmetadata := make([]byte, len(wmetadata))
	rHandle, err := vol.ReadMetadata("Snap", "lost+found/metadata")
	c.Assert(err, IsNil)
	c.Assert(rHandle, NotNil)
	n, err = rHandle.Read(rmetadata)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(rmetadata))
	c.Assert(string(rmetadata), Equals, string(wmetadata))
	err = rHandle.Close()
	c.Assert(err, IsNil)

	// Rollback to the original snapshot and verify the base again
	err = vol.Rollback("Snap")
	c.Assert(err, IsNil)
	verifyBase(c, driver, vol)

	// Rollback to the new snapshot and verify the extra file
	err = vol.Rollback("Snap2")
	c.Assert(err, IsNil)
	verifyBaseWithExtra(c, driver, vol)

	// Make sure we still have all our snapshots
	snaps, err = vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(arrayContains(snaps, "Base_Snap"), Equals, true)
	c.Assert(arrayContains(snaps, "Base_Snap2"), Equals, true)

	// Find Tag (not found)
	info, err = vol.GetSnapshotWithTag("noTag")
	c.Assert(err, Equals, volume.ErrSnapshotDoesNotExist)
	c.Assert(info, IsNil)

	// Find Tag (found)
	info, err = vol.GetSnapshotWithTag("tagA")
	c.Assert(err, Equals, nil)
	c.Assert(info, NotNil)
	c.Check(info.Name, Equals, "Base_Snap")
	c.Check(info.Tags, DeepEquals, []string{"SnapTag", "tagA"})

	// Snapshot using an existing label and make sure it errors properly
	err = vol.Snapshot("Snap", "snapshot-message-2", []string{"tag4"})
	c.Assert(err, ErrorMatches, volume.ErrSnapshotExists.Error())

	// Resnapshot using the raw label and make sure it is equivalent
	err = vol.Snapshot("Base_Snap", "snapshot-message-3", []string{"tag5", "tag6"})
	c.Assert(err, ErrorMatches, volume.ErrSnapshotExists.Error())

	c.Assert(driver.Remove("Base"), IsNil)
	c.Assert(driver.Exists("Base"), Equals, false)
}

func DriverTestSnapshotTags(c *C, drivername volume.DriverType, root string, args []string) {
	driver := newDriver(c, drivername, root, args)
	defer cleanup(c, driver)

	vol := createBase(c, driver, "Base")
	verifyBase(c, driver, vol)

	// Set some metadata on the snapshot
	wmetadata := []byte("snap-metadata")
	wHandle, err := vol.WriteMetadata("Snap", "lost+found/metadata")
	c.Assert(err, IsNil)
	c.Assert(wHandle, NotNil)
	n, err := wHandle.Write(wmetadata)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(wmetadata))
	err = wHandle.Close()
	c.Assert(err, IsNil)

	// Snapshot the verified base to produce a new volume
	err = vol.Snapshot("Snap", "snapshot-message-0", []string{"SnapTag", "tagA"})
	c.Assert(err, IsNil)

	snaps, err := vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(arrayContains(snaps, "Base_Snap"), Equals, true)

	info, err := vol.SnapshotInfo("Base_Snap")
	c.Assert(err, IsNil)
	c.Assert(info, NotNil)
	c.Check(info.Name, Equals, "Base_Snap")
	c.Check(info.Label, Equals, "Snap")
	c.Check(info.TenantID, Equals, "Base")
	c.Check(info.Message, Equals, "snapshot-message-0")
	c.Check(info.Tags, DeepEquals, []string{"SnapTag", "tagA"})

	// Add an extra tag to a snapshot (not exists)
	err = vol.TagSnapshot("Base_Snap", "tagB")
	c.Assert(err, IsNil)
	info, err = vol.SnapshotInfo("Base_Snap")
	c.Assert(err, IsNil)
	c.Assert(info, NotNil)
	c.Check(info.Tags, DeepEquals, []string{"SnapTag", "tagA", "tagB"})

	// Add an extra tag to a snapshot (exists)
	err = vol.TagSnapshot("Base_Snap", "tagB")
	c.Assert(err, Equals, volume.ErrTagAlreadyExists)

	// Take another snapshot with an existing tag
	err = vol.Snapshot("Snap2", "snapshot-message-1", []string{"tagB"})
	c.Assert(err, Equals, volume.ErrTagAlreadyExists)

	// Remove Tag (not found)
	label, err := vol.UntagSnapshot("noTag")
	c.Assert(err, Equals, volume.ErrSnapshotDoesNotExist)
	c.Assert(label, Equals, "")

	// Remove Tag (found)
	label, err = vol.UntagSnapshot("tagA")
	c.Assert(err, IsNil)
	c.Assert(label, Equals, "Snap")
	info, err = vol.SnapshotInfo(label)
	c.Assert(err, IsNil)
	c.Assert(info, NotNil)
	c.Check(info.Tags, DeepEquals, []string{"SnapTag", "tagB"})

	c.Assert(driver.Remove("Base"), IsNil)
	c.Assert(driver.Exists("Base"), Equals, false)
}

func DriverTestSnapshotContainerMounts(c *C, drivername volume.DriverType, root string, args []string) {
	driver := newDriver(c, drivername, root, args)
	defer cleanup(c, driver)

	vol := createBase(c, driver, "Base")
	verifyBase(c, driver, vol)

	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd: []string{"bash", "-c",
					"for ((i=1;i<=600;i++)); do ls /test/sentinel && exit 1 ; sleep 1; done"},
			},
		},
		dockerclient.HostConfig{
			Binds: []string{vol.Path() + ":/test"},
		},
	}

	ctr, err := docker.NewContainer(cd, true, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)

	err = vol.Snapshot("Snap", "snapshot-message-0", []string{"SnapTag", "tagA"})
	c.Assert(err, IsNil)

	writeExtra(c, driver, vol, "sentinel")

	status, err := ctr.Wait(15 * time.Second)
	c.Assert(err, IsNil) // Timeout implies that the snapshot disconnected the volume
	c.Assert(status, Equals, 1)
}

func DriverTestBadSnapshot(c *C, drivername volume.DriverType, root string, badsnapshot func(string, volume.Volume) error, args []string) {
	driver := newDriver(c, drivername, root, args)
	defer cleanup(c, driver)

	vol := createBase(c, driver, "Base")
	verifyBase(c, driver, vol)

	//create the bad snapshot
	err := badsnapshot("badsnapshot", vol)
	c.Assert(err, IsNil)

	// Make sure we can still list snapshots
	snaps, err := vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(len(snaps), Equals, 1)
	c.Assert(arrayContains(snaps, "Base_badsnapshot"), Equals, true)

	// GetSnapshotWithTag still works
	snapshot, err := vol.GetSnapshotWithTag("nonexistanttag")
	c.Assert(err, ErrorMatches, volume.ErrSnapshotDoesNotExist.Error())
	c.Assert(snapshot, IsNil)

	// Make sure we can still add another snapshot
	err = vol.Snapshot("Snap", "snapshot-message-0", []string{"SnapTag", "tagA"})
	c.Assert(err, IsNil)

	// And it shows up in the list
	snaps, err = vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(len(snaps), Equals, 2)
	c.Assert(arrayContains(snaps, "Base_Snap"), Equals, true)
	c.Assert(arrayContains(snaps, "Base_badsnapshot"), Equals, true)

	// Trying to get info on the first snapshot fails
	snapInfo, err := vol.SnapshotInfo("Base_badsnapshot")
	c.Assert(err, ErrorMatches, volume.ErrInvalidSnapshot.Error())
	c.Assert(snapInfo, IsNil)

	// Trying to get info on the second snapshot works
	snapInfo, err = vol.SnapshotInfo("Base_Snap")
	c.Assert(err, IsNil)
	c.Assert(snapInfo, NotNil)

	// Trying to roll back to the bad snapshot fails
	err = vol.Rollback("Base_badsnapshot")
	c.Assert(err, ErrorMatches, volume.ErrInvalidSnapshot.Error())

	// We can delete the bad snapshot
	err = vol.RemoveSnapshot("Base_badsnapshot")
	c.Assert(err, IsNil)

	// And it is actually removed
	snaps, err = vol.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(len(snaps), Equals, 1)
	c.Assert(arrayContains(snaps, "Base_badsnapshot"), Equals, false)

	// Add the bad snapshot back and make sure we can remove the volume
	err = badsnapshot("badsnapshot", vol)
	c.Assert(err, IsNil)
	c.Assert(driver.Remove("Base"), IsNil)
	c.Assert(driver.Exists("Base"), Equals, false)
}

func DriverTestResize(c *C, drivername volume.DriverType, root string, args []string) {
	switch drivername {
	case volume.DriverTypeDeviceMapper:
	default:
		c.Skip("Resize tests only apply to devicemapper")
	}
	driver := newDriver(c, drivername, root, args)
	defer cleanup(c, driver)

	vol := createBase(c, driver, "Base")

	origSize := volume.FilesystemBytesSize(vol.Path())

	// Resize to 600MB, since the test device size is 300MB
	err := driver.Resize(vol.Name(), 600*1024*1024)
	c.Assert(err, IsNil)

	// newSize will be double origSize minus a sizeable fs overhead
	newSize := volume.FilesystemBytesSize(vol.Path())
	diff := newSize - origSize*2
	c.Assert(diff <= 50*1024*1024, Equals, true)

	// Try to shrink it, which should fail
	err = driver.Resize(vol.Name(), 100*1024*1024)
	c.Assert(err, ErrorMatches, dm.ErrNoShrinkage.Error())
	c.Assert(volume.FilesystemBytesSize(vol.Path()), Equals, newSize)
}

func DriverTestExportImport(c *C, drivername volume.DriverType, exportfs, importfs string, args []string) {
	buffer := new(bytes.Buffer)

	exportDriver := newDriver(c, drivername, exportfs, args)
	defer cleanup(c, exportDriver)
	importDriver := newDriver(c, drivername, importfs, args)
	defer cleanup(c, importDriver)

	vol := createBase(c, exportDriver, "Base")
	writeExtra(c, exportDriver, vol, "differentfile")
	verifyBaseWithExtra(c, exportDriver, vol)

	// Set some metadata on the snapshot
	wmetadata := []byte("snap-metadata")
	wHandle, err := vol.WriteMetadata("Backup", "lost+found/metadata")
	c.Assert(err, IsNil)
	c.Assert(wHandle, NotNil)
	n, err := wHandle.Write(wmetadata)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(wmetadata))
	err = wHandle.Close()
	c.Assert(err, IsNil)

	// Export the snapshot
	c.Assert(vol.Snapshot("Backup", "", []string{}), IsNil)
	err = vol.Export("Base_Backup", "", buffer)
	c.Assert(err, IsNil)

	// Import the snapshot
	vol2 := createBase(c, importDriver, "Base")
	err = vol2.Import("Base_Backup", buffer)
	c.Assert(err, IsNil)
	snapshots, err := vol2.Snapshots()
	c.Assert(err, IsNil)
	c.Assert(snapshots, DeepEquals, []string{"Base_Backup"})

	// Make sure the metadata path exists for the snapshot
	rmetadata := make([]byte, len(wmetadata))
	rHandle, err := vol2.ReadMetadata("Backup", "lost+found/metadata")
	c.Assert(err, IsNil)
	c.Assert(rHandle, NotNil)
	n, err = rHandle.Read(rmetadata)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(rmetadata))
	c.Assert(string(rmetadata), Equals, string(wmetadata))
	err = rHandle.Close()
	c.Assert(err, IsNil)

	c.Assert(vol2.Rollback("Backup"), IsNil)
	verifyBaseWithExtra(c, importDriver, vol2)
}
