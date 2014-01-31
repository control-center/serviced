// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package volume

import (
	"fmt"
)

var (
	drivers = make(map[string]Driver)
)

type Driver interface {
	MkVolume(volumeName, root string) (*Volume, error)
	Name(volumeName string) string
	Dir(volumeName string) string
	Snapshot(volumeName, label string) (err error)
	Snapshots(volumeName string) (labels []string, err error)
	RemoveSnapshot(volumeName, label string) error
	Rollback(volumeName, label string) (err error)
	RootDir(volumeName string) string
}

type Volume struct {
	driver Driver
	name   string
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

	vol, err := driver.MkVolume(volumeName, rootDir)
	if err != nil {
		return nil, err
	}

	return vol, nil
}

func (v *Volume) Name() string {
	return v.driver.Name(v.name)
}

func (v *Volume) Dir() string {
	return v.driver.Dir(v.name)
}

func (v *Volume) Snapshot(label string) error {
	return v.driver.Snapshot(v.name, label)
}

func (v *Volume) Snapshots() ([]string, error) {
	return v.driver.Snapshots(v.name)
}

func (v *Volume) RemoveSnapshot(label string) error {
	return v.driver.RemoveSnapshot(v.name, label)
}

func (v *Volume) Rollback(label string) error {
	return v.driver.Rollback(v.name, label)
}

func (v *Volume) RootDir() string {
	return v.driver.RootDir(v.name)
}
