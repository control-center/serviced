/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
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

func localDir(p string) string {
	homeDir := os.Getenv("SERVICED_HOME")
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = path.Dir(filename)
	}
	return path.Join(homeDir, p)
}

func imagesDir() string {
	return localDir("images")
}

func resourcesDir() string {
	path, err := filepath.EvalSymlinks(localDir("resources"))
	if err != nil {
		glog.Fatalf("Could not evaluate %s, not following symlinks: %s", localDir("resources"), err)
	}
	return path
}
