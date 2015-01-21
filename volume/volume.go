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
	"os"
	"sort"

	"github.com/zenoss/glog"

	"fmt"
)

var (
	drivers = make(map[string]Driver)
)

type FileInfoSlice []os.FileInfo

func (p FileInfoSlice) Len() int           { return len(p) }
func (p FileInfoSlice) Less(i, j int) bool { return p[i].ModTime().Before(p[j].ModTime()) }
func (p FileInfoSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p FileInfoSlice) Labels() []string {
	sort.Sort(p)
	labels := make([]string, p.Len())
	for i, label := range p {
		labels[i] = label.Name()
	}
	return labels
}

type Driver interface {
	Mount(volumeName, root string) (Volume, error)
	List(root string) []string
}

type Volume interface {
	Name() string
	Path() string
	SnapshotPath(label string) string
	Snapshot(label string) (err error)
	Snapshots() ([]string, error)
	RemoveSnapshot(label string) error
	Rollback(label string) error
	Unmount() error
	Export(label, parent, filename string) error
	Import(label, filename string) error
}

func Register(name string, driver Driver) {
	if driver == nil {
		panic("volume: Register driver is nil")
	}

	if _, dup := drivers[name]; dup {
		panic("volume: Register called twice for driver: " + name)
	}

	drivers[name] = driver
}

func Registered(name string) (Driver, bool) {
	driver, registered := drivers[name]
	return driver, registered
}

func Mount(driverName, volumeName, rootDir string) (Volume, error) {
	driver, ok := Registered(driverName)
	if ok == false {
		return nil, fmt.Errorf("No such driver: %s", driverName)
	}

	volume, err := driver.Mount(volumeName, rootDir)
	if err != nil {
		glog.Errorf("Error mounting :%s", err)
		return nil, err
	}

	return volume, nil
}
