// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package volume

import (
	"github.com/zenoss/glog"

	"errors"
)

type Volume interface {
	New(baseDir, name string) (Volume, error)
	Name() string
	Dir() string
	Snapshot(label string) (err error)
	Snapshots() (labels []string, err error)
	RemoveSnapshot(label string) error
	Rollback(label string) (err error)
	BaseDir() string
}

var ErrNoSupportedDrivers error
var SupportedDrivers map[string]Volume

func init() {
	ErrNoSupportedDrivers = errors.New("no supported drivers found")

	SupportedDrivers = make(map[string]Volume)
	SupportedDrivers["btrfs"] = &BtrfsVolume{}
	SupportedDrivers["rsync"] = &RsyncVolume{}
}

func New(baseDir, volumeName string) (volume Volume, err error) {

	for name, driver := range SupportedDrivers {
		glog.V(4).Infof("Detecting if %s is supported on %s", name, baseDir)
		if volume, err = driver.New(baseDir, name); err == nil {
			glog.V(4).Infof("%s is supported on %s", name, baseDir)
			return volume, nil
		} else {
			glog.V(4).Infof("%s is NOT supported on %s", name, baseDir)
		}
	}
	if volume == nil {
		err = ErrNoSupportedDrivers
	}
	return volume, err
}
