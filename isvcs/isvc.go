// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/utils"

	"fmt"
	"os/user"
)

var Mgr *Manager

const (
	IMAGE_REPO = "zctrl/isvcs"
	IMAGE_TAG  = "v4"
)

func Init() {
	var volumesDir string
	if user, err := user.Current(); err == nil {
		volumesDir = fmt.Sprintf("/tmp/serviced-%s/var/isvcs", user.Username)
	} else {
		volumesDir = "/tmp/serviced/var/isvcs"
	}

	Mgr = NewManager("unix:///var/run/docker.sock", imagesDir(), volumesDir)

	if err := Mgr.Register(elasticsearch); err != nil {
		glog.Fatalf("%s", err)
	}
	if err := Mgr.Register(zookeeper); err != nil {
		glog.Fatalf("%s", err)
	}
	if err := Mgr.Register(logstash); err != nil {
		glog.Fatalf("%s", err)
	}
	if err := Mgr.Register(opentsdb); err != nil {
		glog.Fatalf("%s", err)
	}
	if err := Mgr.Register(celery); err != nil {
		glog.Fatalf("%s", err)
	}
}

func imagesDir() string {
	return utils.LocalDir("images")
}
