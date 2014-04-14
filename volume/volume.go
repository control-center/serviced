// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package volume

import (
	"github.com/zenoss/glog"

	"fmt"
)

var (
	drivers = make(map[string]Driver)
)

type Driver interface {
	Mount(volumeName, root string) (Conn, error)
	List(root string) []string
}

type Conn interface {
	Name() string
	Path() string
	SnapshotPath(label string) string
	Snapshot(label string) (err error)
	Snapshots() ([]string, error)
	RemoveSnapshot(label string) error
	Rollback(label string) error
	Unmount() error
}

type Volume struct {
	Conn
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

func Mount(driverName, volumeName, rootDir string) (*Volume, error) {
	driver, ok := Registered(driverName)
	if ok == false {
		return nil, fmt.Errorf("No such driver: %s", driverName)
	}

	conn, err := driver.Mount(volumeName, rootDir)
	if err != nil {
		glog.Errorf("Error mounting :%s", err)
		return nil, err
	}

	return &Volume{conn}, nil
}
