/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/utils"
	"os/user"
)

var Mgr *Manager

const (
	IMAGE_REPO = "zctrl/isvcs"
	IMAGE_TAG  = "v2"
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
