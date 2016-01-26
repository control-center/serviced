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

package btrfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/control-center/serviced/volume"
	"github.com/dustin/go-humanize"
	"github.com/zenoss/glog"
)

var (
	ErrBtrfsInvalidFilesystem = errors.New("not a btrfs filesystem")
	ErrBtrfsInvalidDriver     = errors.New("invalid driver")
	ErrBtrfsCreatingSubvolume = errors.New("could not create subvolume")
	ErrBtrfsInvalidLabel      = errors.New("invalid label")
	ErrBtrfsListingSnapshots  = errors.New("couldn't list snapshots")
	ErrBtrfsNotSupported      = errors.New("operation not supported on btrfs driver")
)

func init() {
	volume.Register(volume.DriverTypeBtrFS, Init)
}

// BtrfsDriver is a driver for the btrfs volume
type BtrfsDriver struct {
	sudoer   bool
	root     string
	objectID string
	sync.Mutex
}

// BtrfsVolume is a btrfs volume
type BtrfsVolume struct {
	sudoer bool
	name   string
	path   string
	tenant string
	driver volume.Driver
	sync.Mutex
}

type BtrfsDFData struct {
	DataType string
	Level    string
	Total    uint64
	Used     uint64
}

// Btrfs driver initialization
func Init(root string, _ []string) (volume.Driver, error) {
	if !volume.IsBtrfsFilesystem(root) {
		return nil, ErrBtrfsInvalidFilesystem
	}
	// get the driver object id
	objectID := "5" // root driver is always 5 unless it is a subvolume
	raw, err := volume.RunBtrFSCmd(volume.IsSudoer(), "subvolume", "show", root)
	if err != nil {
		glog.Errorf("Could not initialize btrfs driver for %s: %s (%s)", root, raw, err)
		return nil, volume.ErrBtrfsCommand
	}
	if obidmatch := regexp.MustCompile("Object ID:\\s+(\\w+)").FindSubmatch(raw); len(obidmatch) == 2 {
		objectID = string(obidmatch[1])
	}
	driver := &BtrfsDriver{
		sudoer:   volume.IsSudoer(),
		root:     root,
		objectID: objectID,
	}
	// Create future metadata dir, into which we can put stuff, but for now is
	// used for detection of type
	if err := os.MkdirAll(driver.MetadataDir(), 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	if err := volume.TouchFlagFile(driver.poolDir()); err != nil {
		return nil, err
	}
	return driver, nil
}

// Root implements volume.Driver.Root
func (d *BtrfsDriver) Root() string {
	return d.root
}

// DriverType implements volume.Driver.DriverType
func (d *BtrfsDriver) DriverType() volume.DriverType {
	return volume.DriverTypeBtrFS
}

// Exists implements volume.Driver.Exists
func (d *BtrfsDriver) Exists(volumeName string) bool {
	if _, err := volume.RunBtrFSCmd(d.sudoer, "subvolume", "show", filepath.Join(d.root, volumeName)); err != nil {
		return false
	}
	return true
}

// Cleanup implements volume.Driver.Cleanup
func (d *BtrfsDriver) Cleanup() error {
	// Btrfs driver has no hold on system resources
	return nil
}

// Release implements volume.Driver.Release
func (d *BtrfsDriver) Release(volumeName string) error {
	// Btrfs volumes are essentially just a directory; nothing to release
	return nil
}

// Create implements volume.Driver.Create
func (d *BtrfsDriver) Create(volumeName string) (volume.Volume, error) {
	d.Lock()
	defer d.Unlock()
	rootDir := d.root
	if _, err := volume.RunBtrFSCmd(d.sudoer, "subvolume", "list", rootDir); err != nil {
		if _, err := volume.RunBtrFSCmd(d.sudoer, "subvolume", "create", rootDir); err != nil {
			glog.Errorf("Could not create subvolume at: %s", rootDir)
			return nil, ErrBtrfsCreatingSubvolume
		}
	}
	vdir := path.Join(rootDir, volumeName)
	if _, err := volume.RunBtrFSCmd(d.sudoer, "subvolume", "list", vdir); err != nil {
		if _, err = volume.RunBtrFSCmd(d.sudoer, "subvolume", "create", vdir); err != nil {
			glog.Errorf("Could not create volume at: %s", vdir)
			return nil, ErrBtrfsCreatingSubvolume
		}
	}
	return d.Get(volumeName)
}

func (d *BtrfsDriver) poolDir() string {
	return filepath.Join(d.root, ".btrfs")
}

// MetadataDir returns the path to a volume's metadata directory
func (d *BtrfsDriver) MetadataDir() string {
	return filepath.Join(d.poolDir(), "volumes")
}

// Remove implements volume.Driver.Remove
func (d *BtrfsDriver) Remove(volumeName string) error {
	d.Lock()
	defer d.Unlock()
	if !d.Exists(volumeName) {
		glog.Warningf("Volume %s does not exist", volumeName)
		return nil
	}
	v, err := d.Get(volumeName)
	if err != nil {
		return err
	}
	snapshots, err := v.Snapshots()
	if err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		if err := v.RemoveSnapshot(snapshot); err != nil {
			return err
		}
	}
	if output, err := volume.RunBtrFSCmd(d.sudoer, "subvolume", "delete", v.Path()); err != nil {
		glog.Errorf("Could not remove volume %s: %s (%s)", v.Name(), output, err)
		return volume.ErrRemovingVolume
	}
	return nil
}

func (d *BtrfsDriver) Status() (*volume.Status, error) {
	glog.V(2).Info("btrfs.Status()")
	rootDir := d.root
	dfstatus, err := volume.RunBtrFSCmd(d.sudoer, "filesystem", "df", "-b", rootDir)
	if err != nil {
		glog.V(2).Infof("Error running df command with -b: %v. Trying without for older btrfs.", err)
		dfstatus, err = volume.RunBtrFSCmd(d.sudoer, "filesystem", "df", rootDir)
		if err != nil {
			glog.Errorf("Could not get status of filestystem at %s", rootDir)
			return nil, err
		}
	}
	glog.V(2).Infof("Output from btrfs filesystem df %s: %s", rootDir, dfstatus)
	dfData, err := parseDF(strings.Split(string(dfstatus), "\n"))
	if err != nil {
		glog.Errorf("Could not parse df output: %s", err)
		return nil, err
	}
	glog.V(2).Infof("dfData = %v", dfData)

	usage := dfDataToUsageData(dfData)
	response := &volume.Status{
		Driver:     volume.DriverTypeBtrFS,
		UsageData:  usage,
		DriverData: map[string]string{"DataFile": rootDir},
	}
	return response, nil
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

// GetTenant implements volume.Driver.GetTenant
func (d *BtrfsDriver) GetTenant(volumeName string) (volume.Volume, error) {
	if !d.Exists(volumeName) {
		return nil, volume.ErrVolumeNotExists
	}
	return d.Get(getTenant(volumeName))
}

// Resize implements volume.Driver.Resize. For btrfs it's a noop.
func (d *BtrfsDriver) Resize(volumeName string, size uint64) error {
	return nil
}

// Get implements volume.Driver.Get
func (d *BtrfsDriver) Get(volumeName string) (volume.Volume, error) {
	volumePath := filepath.Join(d.root, volumeName)
	v := &BtrfsVolume{
		sudoer: d.sudoer,
		name:   volumeName,
		path:   volumePath,
		driver: d,
		tenant: getTenant(volumeName),
	}
	return v, nil
}

// List implements volume.Driver.List
func (d *BtrfsDriver) List() (result []string) {
	glog.Infof("Checking volumes at %s", d.root)
	raw, err := volume.RunBtrFSCmd(d.sudoer, "subvolume", "list", d.root)
	if err != nil {
		glog.Warningf("Could not list subvolumes at %s: %s (%s)", d.root, raw, err)
		return
	}
	rows := regexp.MustCompile(fmt.Sprintf("top level %s path (\\w+)", d.objectID)).FindAllSubmatch(raw, -1)
	for _, row := range rows {
		result = append(result, string(row[1]))
	}
	return
}

// Name implements volume.Volume.Name
func (v *BtrfsVolume) Name() string {
	return v.name
}

// Path implements volume.Volume.Path
func (v *BtrfsVolume) Path() string {
	return v.path
}

// Driver implements volume.Volume.Driver
func (v *BtrfsVolume) Driver() volume.Driver {
	return v.driver
}

// Tenant implements volume.Volume.Tenant
func (v *BtrfsVolume) Tenant() string {
	return v.tenant
}

// WriteMetadata writes the metadata info for a snapshot by writing to the base
// volume.
func (v *BtrfsVolume) WriteMetadata(label, name string) (io.WriteCloser, error) {
	filePath := filepath.Join(v.Path(), name)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil && !os.IsExist(err) {
		glog.Errorf("Could not create path for file %s: %s", name, err)
		return nil, err
	}

	return os.Create(filePath)
}

// ReadMetadata reads the metadata info from a snapshot
func (v *BtrfsVolume) ReadMetadata(label, name string) (io.ReadCloser, error) {
	filePath := filepath.Join(v.snapshotPath(label), name)
	return os.Open(filePath)
}

func (v *BtrfsVolume) getSnapshotPrefix() string {
	return v.Tenant() + "_"
}

// rawSnapshotLabel ensures that <label> has the tenant prefix for this volume
func (v *BtrfsVolume) rawSnapshotLabel(label string) string {
	prefix := v.getSnapshotPrefix()
	if !strings.HasPrefix(label, prefix) {
		return prefix + label
	}
	return label
}

// prettySnapshotLabel ensures that <label> does not have the tenant prefix for
// this volume
func (v *BtrfsVolume) prettySnapshotLabel(rawLabel string) string {
	return strings.TrimPrefix(rawLabel, v.getSnapshotPrefix())
}

// snapshotPath gets the path to the btrfs subvolume representing the snapshot <label>
func (v *BtrfsVolume) snapshotPath(label string) string {
	root := v.Driver().Root()
	rawLabel := v.rawSnapshotLabel(label)
	return filepath.Join(root, rawLabel)
}

// isSnapshot checks to see if <rawLabel> describes a snapshot (i.e., begins
// with the tenant prefix)
func (v *BtrfsVolume) isSnapshot(rawLabel string) bool {
	return strings.HasPrefix(rawLabel, v.getSnapshotPrefix())
}

// isInvalidSnapshot checks to see if <rawLabel> describes a snapshot (i.e., begins
// with the tenant prefix) but does NOT have a valid metadata file
func (v *BtrfsVolume) isInvalidSnapshot(rawLabel string) bool {
	if strings.HasPrefix(rawLabel, v.getSnapshotPrefix()) {
		reader, err := v.ReadMetadata(rawLabel, ".SNAPSHOTINFO")
		if err != nil {
			return true
		}
		reader.Close()
	}
	return false
}

// writeSnapshotInfo writes metadata about a snapshot
func (v *BtrfsVolume) writeSnapshotInfo(label string, info *volume.SnapshotInfo) error {
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
func (v *BtrfsVolume) SnapshotInfo(label string) (*volume.SnapshotInfo, error) {
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
func (v *BtrfsVolume) Snapshot(label, message string, tags []string) error {
	// make sure the label doesn't already exist
	path := v.snapshotPath(label)
	if ok, err := volume.IsDir(path); err != nil {
		return err
	} else if ok {
		return volume.ErrSnapshotExists
	}
	// check the tags for duplicates
	for _, tagName := range tags {
		if tagInfo, err := v.GetSnapshotWithTag(tagName); err != volume.ErrSnapshotDoesNotExist {
			if err != nil {
				glog.Errorf("Could not look up snapshot with tag %s: %s", tagName, err)
				return err
			} else {
				glog.Errorf("Tag '%s' is already in use by snapshot %s", tagName, tagInfo.Name)
				return volume.ErrTagAlreadyExists
			}
		}
	}
	v.Lock()
	defer v.Unlock()
	info := volume.SnapshotInfo{
		Name:     v.rawSnapshotLabel(label),
		TenantID: v.Tenant(),
		Label:    v.prettySnapshotLabel(label),
		Tags:     tags,
		Message:  message,
		Created:  time.Now(),
	}
	if err := v.writeSnapshotInfo(label, &info); err != nil {
		return err
	}
	_, err := volume.RunBtrFSCmd(v.sudoer, "subvolume", "snapshot", "-r", v.Path(), path)
	return err
}

// TagSnapshot implements volume.Volume.TagSnapshot
// This is not implemented with btrfs
func (v *BtrfsVolume) TagSnapshot(label string, tagName string) error {
	return ErrBtrfsNotSupported
}

// UntagSnapshot implements volume.Volume.RemoveSnapshotTag
// This is not implemented with btrfs
func (v *BtrfsVolume) UntagSnapshot(tagName string) (string, error) {
	return "", ErrBtrfsNotSupported
}

// GetSnapshotWithTag implements volume.Volume.GetSnapshotWithTag
func (v *BtrfsVolume) GetSnapshotWithTag(tagName string) (*volume.SnapshotInfo, error) {
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
func (v *BtrfsVolume) Snapshots() ([]string, error) {
	v.Lock()
	defer v.Unlock()

	glog.V(2).Infof("listing snapshots of volume:%v and v.name:%s ", v.path, v.name)
	output, err := volume.RunBtrFSCmd(v.sudoer, "subvolume", "list", "-s", v.path)
	if err != nil {
		glog.Errorf("Could not list subvolumes of %s: %s", v.path, err)
		return nil, err
	}

	var files []os.FileInfo
	for _, line := range strings.Split(string(output), "\n") {
		glog.V(0).Infof("line: %s", line)
		if parts := strings.Split(line, "path"); len(parts) == 2 {
			rawLabel := strings.TrimSpace(parts[1])
			rawLabel = strings.TrimPrefix(rawLabel, "volumes/")
			if v.isSnapshot(rawLabel) {
				label := v.prettySnapshotLabel(rawLabel)
				file, err := os.Stat(filepath.Join(v.Driver().Root(), rawLabel))
				if err != nil {
					glog.Errorf("Could not stat snapshot %s: %s", label, err)
					return nil, err
				}
				files = append(files, file)
				glog.V(2).Infof("found snapshot:%s", label)
			}
		}
	}
	labels := volume.FileInfoSlice(files).Labels()
	return labels, nil
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *BtrfsVolume) RemoveSnapshot(label string) error {
	if exists, err := v.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return volume.ErrSnapshotDoesNotExist
		}
	}

	v.Lock()
	defer v.Unlock()
	_, err := volume.RunBtrFSCmd(v.sudoer, "subvolume", "delete", v.snapshotPath(label))
	if err != nil {
		glog.Errorf("could not remove snapshot: %s", err)
		return volume.ErrRemovingSnapshot
	}
	return nil
}

// getEnvMinDuration returns the time.Duration env var meeting minimum and default duration
func getEnvMinDuration(envvar string, def, min int32) time.Duration {
	duration := def
	envval := os.Getenv(envvar)
	if len(strings.TrimSpace(envval)) == 0 {
		// ignore unset envvar
	} else if intVal, intErr := strconv.ParseInt(envval, 0, 32); intErr != nil {
		glog.Warningf("ignoring invalid %s of '%s': %s", envvar, envval, intErr)
		duration = min
	} else if int32(intVal) < min {
		glog.Warningf("ignoring invalid %s of '%s' < minimum:%v seconds", envvar, envval, min)
	} else {
		duration = int32(intVal)
	}

	return time.Duration(duration) * time.Second
}

// Rollback implements volume.Volume.Rollback
func (v *BtrfsVolume) Rollback(label string) error {
	if v.isInvalidSnapshot(label) {
		return volume.ErrInvalidSnapshot
	}

	if exists, err := v.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return volume.ErrSnapshotDoesNotExist
		}
	}

	v.Lock()
	defer v.Unlock()
	vd := filepath.Join(v.Driver().Root(), v.name)
	dirp, err := volume.IsDir(vd)
	if err != nil {
		return err
	}

	glog.Infof("starting rollback of snapshot %s", label)

	start := time.Now()
	if dirp {
		timeout := getEnvMinDuration("SERVICED_BTRFS_ROLLBACK_TIMEOUT", 300, 120)
		glog.Infof("rollback using env var SERVICED_BTRFS_ROLLBACK_TIMEOUT:%s", timeout)

		for {
			cmd := []string{"subvolume", "delete", vd}
			output, deleteError := volume.RunBtrFSCmd(v.sudoer, cmd...)
			if deleteError == nil {
				break
			}

			now := time.Now()
			if now.Sub(start) > timeout {
				glog.Errorf("rollback of snapshot %s failed - btrfs subvolume deletes took %s for cmd:%s", label, timeout, cmd)
				return deleteError
			} else if strings.Contains(string(output), "Device or resource busy") {
				waitTime := time.Duration(5 * time.Second)
				glog.Warningf("retrying rollback subvolume delete in %s - unable to run cmd:%s  output:%s  error:%s", waitTime, cmd, string(output), deleteError)
				time.Sleep(waitTime)
			} else {
				return deleteError
			}
		}
	}

	cmd := []string{"subvolume", "snapshot", v.snapshotPath(label), vd}
	_, err = volume.RunBtrFSCmd(v.sudoer, cmd...)
	if err != nil {
		glog.Errorf("rollback of snapshot %s failed for cmd:%s", label, cmd)
	} else {
		duration := time.Now().Sub(start)
		glog.Infof("rollback of snapshot %s took %s", label, duration)
	}
	return err
}

// Export implements volume.Volume.Export
func (v *BtrfsVolume) Export(label, parent string, writer io.Writer) error {
	if label = strings.TrimSpace(label); label == "" {
		glog.Errorf("%s: label cannot be empty", volume.DriverTypeBtrFS)
		return ErrBtrfsInvalidLabel
	} else if exists, err := v.snapshotExists(label); err != nil {
		return err
	} else if !exists {
		return volume.ErrSnapshotDoesNotExist
	}
	// TODO: add to tarfile and include metadata
	if err := runBtrfsSend(writer, v.sudoer, parent, v.snapshotPath(label)); err != nil {
		glog.Errorf("Could not export snapshot %s: %s", label, err)
		return err
	}
	return nil
}

// Import implements volume.Volume.Import
func (v *BtrfsVolume) Import(label string, reader io.Reader) error {
	if exists, err := v.snapshotExists(label); err != nil {
		return err
	} else if exists {
		return volume.ErrSnapshotExists
	}
	importdir := filepath.Join(v.path, fmt.Sprintf("import-%s", label))
	if _, err := volume.RunBtrFSCmd(v.sudoer, "subvolume", "create", importdir); err != nil {
		glog.Errorf("Could not create import path for snapshot %s: %s", label, err)
		return err
	}
	defer volume.RunBtrFSCmd(v.sudoer, "subvolume", "delete", importdir)
	if err := runBtrfsRecv(reader, v.sudoer, importdir); err != nil {
		glog.Errorf("Could not import snapshot %s: %s", label, err)
		return err
	}
	defer volume.RunBtrFSCmd(v.sudoer, "subvolume", "delete", filepath.Join(importdir, label))
	if _, err := volume.RunBtrFSCmd(v.sudoer, "subvolume", "snapshot", "-r", filepath.Join(importdir, label), v.Driver().Root()); err != nil {
		return err
	}
	return nil
}

// snapshotExists queries the snapshot existence for the given label
func (v *BtrfsVolume) snapshotExists(label string) (exists bool, err error) {
	rlabel := v.rawSnapshotLabel(label)
	plabel := v.prettySnapshotLabel(label)
	if snapshots, err := v.Snapshots(); err != nil {
		glog.Errorf("Could not get current snapshot list: %v", err)
		return false, ErrBtrfsListingSnapshots
	} else {
		for _, snapLabel := range snapshots {
			if rlabel == snapLabel || plabel == snapLabel {
				return true, nil
			}
		}
	}
	return false, nil
}

// runBtrfsSend writes a btrfs snapshot to a write handle
func runBtrfsSend(writer io.Writer, sudoer bool, parentpath, path string) error {
	cmdArgs := []string{"btrfs", "send", path}
	if parentpath = strings.TrimSpace(parentpath); parentpath != "" {
		cmdArgs = append(cmdArgs, "-p", parentpath)
	}
	if sudoer {
		cmdArgs = append([]string{"sudo", "-n"}, cmdArgs...)
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		glog.Errorf("Error while running command %+v: %s", cmdArgs, err)
		return volume.ErrBtrfsCommand
	}
	return nil
}

// runBtrfsRecv reads a btrfs snapshot to a read handle
func runBtrfsRecv(reader io.Reader, sudoer bool, path string) error {
	cmdArgs := []string{"btrfs", "receive", path}
	if sudoer {
		cmdArgs = append([]string{"sudo", "-n"}, cmdArgs...)
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = reader
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		glog.Errorf("Error while running command %+v: %s", cmdArgs, err)
		return volume.ErrBtrfsCommand
	}
	return nil
}

// Parse output of btrfs fi df command. Older btrfs does not support -b/--raw arg.
// output without -b:
/*
	System, DUP: total=8.00MiB, used=16.00KiB
	System, single: total=4.00MiB, used=0.00B
	Metadata, DUP: total=51.19MiB, used=112.00KiB
	Metadata, single: total=8.00MiB, used=0.00B
	GlobalReserve, single: total=16.00MiB, used=0.00B
*/
// output with -b:
/*
	System, DUP: total=8388608, used=16384
	System, single: total=4194304, used=0
	Metadata, DUP: total=53673984, used=114688
	Metadata, single: total=8388608, used=0
	GlobalReserve, single: total=16777216, used=0
*/
func parseDF(lines []string) ([]BtrfsDFData, error) {
	lines = removeBlankLines(lines)
	if len(lines) < 3 {
		return []BtrfsDFData{}, fmt.Errorf("insufficient output: %v", strings.Join(lines, "\n"))
	}
	df := []BtrfsDFData{}
	var err error
	for _, line := range lines {
		line = strings.TrimSpace(line)
		fields := strings.FieldsFunc(line, func(c rune) bool { return unicode.IsSpace(c) || strings.ContainsRune(",:", c) })
		glog.V(2).Infof("Fields from line %v: %v", line, fields)
		if len(fields) == 0 {
			glog.Info("Skipping blank line in input.")
			continue
		}
		if len(fields) != 4 {
			glog.Errorf("Wrong number of fields (%d, expected 4) in line %q", len(fields), line)
			return []BtrfsDFData{}, fmt.Errorf("Wrong number of fields (%d, expected 4) in line %q", len(fields), line)
		}
		switch fields[0] {
		case "Data", "System", "Metadata", "GlobalReserve":
			total := fields[2]
			var totalBytes, usedBytes uint64
			if strings.HasPrefix(total, "total=") {
				total = strings.TrimPrefix(total, "total=")
				if totalBytes, err = parseSize(total); err != nil {
					glog.Errorf("parseSize(%s) returned error: %s", total, err)
					return []BtrfsDFData{}, err
				}
			} else {
				glog.Errorf("total field not found in line %q", line)
				return []BtrfsDFData{}, fmt.Errorf("expected total field, not found in line %q", line)
			}
			used := fields[3]
			if strings.HasPrefix(used, "used=") {
				used = strings.TrimPrefix(used, "used=")
				if usedBytes, err = parseSize(used); err != nil {
					glog.Errorf("parseSize(%s) returned error: %s", used, err)
					return []BtrfsDFData{}, err
				}
			} else {
				glog.Errorf("used field not found in line %q", line)
				return []BtrfsDFData{}, fmt.Errorf("expected used field, not found in line %q", line)
			}

			df = append(df, BtrfsDFData{DataType: fields[0], Level: fields[1], Total: totalBytes, Used: usedBytes})
		default:
			glog.Errorf("Unrecognized field %q in line %q", fields[0], line)
			return []BtrfsDFData{}, fmt.Errorf("Unrecognized field %q in line %q", fields[0], line)
		}
	}
	return df, nil
}

func parseSize(size string) (uint64, error) {
	sizemod := strings.Trim(size, " ,:")
	sizeret, err := humanize.ParseBytes(sizemod)
	return sizeret, err
}

func dfDataToUsageData(dfData []BtrfsDFData) []volume.Usage {
	result := []volume.Usage{}
	for _, btrfsData := range dfData {
		label := btrfsData.DataType + " " + btrfsData.Level
		result = append(result, volume.Usage{Label: label, Type: "Total", Value: btrfsData.Total})
		result = append(result, volume.Usage{Label: label, Type: "Used", Value: btrfsData.Used})
	}
	return result
}

func removeBlankLines(in []string) []string {
	out := []string{}
	for _, value := range in {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
