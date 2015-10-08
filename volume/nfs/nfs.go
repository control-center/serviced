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

package nfs

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/control-center/serviced/volume"
)

var (
	ErrNotSupported = errors.New("not supported by nfs driver")
)

type NFSDriver struct {
	root string
}

type NFSVolume struct {
	name   string
	path   string
	tenant string
	driver *NFSDriver
}

func init() {
	volume.Register(volume.DriverTypeNFS, Init)
}

func Init(root string, _ []string) (volume.Driver, error) {
	// TODO: Initialize the NFS mount here instead of in the HostAgent code
	if fi, err := os.Stat(root); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, volume.ErrNotADirectory
	}
	driver := &NFSDriver{
		root: root,
	}
	return driver, nil
}

// TODO: Put this somewhere shareable
func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

// Root implements volume.Driver.Root
func (d *NFSDriver) Root() string {
	return d.root
}

// DriverType implements volume.Driver.DriverType
func (d *NFSDriver) DriverType() volume.DriverType {
	return volume.DriverTypeNFS
}

// Get implements volume.Driver.Get
func (d *NFSDriver) Get(volumeName string) (volume.Volume, error) {
	volumePath := filepath.Join(d.root, volumeName)
	volume := &NFSVolume{
		name:   volumeName,
		path:   volumePath,
		driver: d,
		tenant: getTenant(volumeName),
	}
	return volume, nil
}

// List implements volume.Driver.List
func (d *NFSDriver) List() (result []string) {
	fis, err := ioutil.ReadDir(d.root)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if fi.IsDir() {
			result = append(result, fi.Name())
		}
	}
	return
}

// Exists implements volume.Driver.Exists
func (d *NFSDriver) Exists(volumeName string) bool {
	var (
		fi  os.FileInfo
		err error
	)
	if fi, err = os.Stat(filepath.Join(d.root, volumeName)); err != nil {
		return false
	}
	return fi.IsDir()
}

// Cleanup implements volume.Driver.Cleanup
func (d *NFSDriver) Cleanup() error {
	// TODO: Release the NFS root
	return nil
}

// Create implements volume.Driver.Create
func (d *NFSDriver) Create(volumeName string) (volume.Volume, error) {
	return nil, ErrNotSupported
}

// Remove implements volume.Driver.Remove
func (d *NFSDriver) Remove(volumeName string) error {
	return ErrNotSupported
}

func (d *NFSDriver) Status() (*volume.Status, error) {
	return nil, ErrNotSupported
}

// Release implements volume.Driver.Release
func (d *NFSDriver) Release(volumeName string) error {
	return ErrNotSupported
}

// Name implements volume.Volume.Name
func (v *NFSVolume) Name() string {
	return v.name
}

// Path implements volume.Volume.Path
func (v *NFSVolume) Path() string {
	return v.path
}

// Driver implements volume.Volume.Driver
func (v *NFSVolume) Driver() volume.Driver {
	return v.driver
}

// Tenant implements volume.Volume.Tenant
func (v *NFSVolume) Tenant() string {
	return v.tenant
}

// WriteMetadata implements volume.Volume.WriteMetadata
func (v *NFSVolume) WriteMetadata(label, name string) (io.WriteCloser, error) {
	return nil, ErrNotSupported
}

// ReadMetadata implements volume.Volume.ReadMetadata
func (v *NFSVolume) ReadMetadata(label, name string) (io.ReadCloser, error) {
	return nil, ErrNotSupported
}

// Snapshot implements volume.Volume.Snapshot
func (v *NFSVolume) Snapshot(label, message string, tags []string) (err error) {
	return ErrNotSupported
}

// SnapshotInfo implements volume.Volume.SnapshotInfo
func (v *NFSVolume) SnapshotInfo(label string) (*volume.SnapshotInfo, error) {
	return nil, ErrNotSupported
}

// Snapshots implements volume.Volume.Snapshots
func (v *NFSVolume) Snapshots() ([]string, error) {
	return nil, ErrNotSupported
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *NFSVolume) RemoveSnapshot(label string) error {
	return ErrNotSupported
}

// Rollback implements volume.Volume.Rollback
func (v *NFSVolume) Rollback(label string) (err error) {
	return ErrNotSupported
}

// Export implements volume.Volume.Export
func (v *NFSVolume) Export(label, parent string, writer io.Writer) error {
	return ErrNotSupported
}

// Import implements volume.Volume.Import
func (v *NFSVolume) Import(label string, reader io.Reader) error {
	return ErrNotSupported
}
