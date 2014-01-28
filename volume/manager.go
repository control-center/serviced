/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*******************************************************************************/

package volume

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/volume/btrfs"

	"errors"
)

type Volume interface {
	Dir() string
	Snapshot() (label string, err error)
	Snapshots() (labels []string, err error)
	RemoveSnapshot(label string) error
	Rollback(label string) (err error)
	BaseDir()
}

type Manager struct {
	driver Volume
}

var ErrNoSupportedDrivers error

func init() {
	ErrNoSupportedDrivers = errors.New("no supported drivers found")
}

// NewManager() returns a new Manager object. If no driver is specified, the
// driver will be auto-detected. If any error is encountered, it is returned
// and newManager is set to nil.
func NewManager(driver string) (newManager *Manager, err error) {
	glog.V(4).Info("Creating new Manager object")

	var volume Volume
	if btrfs.Supported() {
		driver = btrfs.Volume
	}

	if driver == nil {
		return nil, ErrNoSupportedDrivers
	}

	mgr := Manager{
		driver: volume,
	}
	return mgr, nil
}
