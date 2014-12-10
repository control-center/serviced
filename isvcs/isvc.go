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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"

	"fmt"
	"os/user"
)

var Mgr *Manager

const (
	IMAGE_REPO = "zenoss/serviced-isvcs"
	IMAGE_TAG  = "v24"
)

func Init() {
	var volumesDir string
	if user, err := user.Current(); err == nil {
		volumesDir = fmt.Sprintf("/tmp/serviced-%s/var/isvcs", user.Username)
	} else {
		volumesDir = "/tmp/serviced/var/isvcs"
	}

	Mgr = NewManager(imagesDir(), volumesDir)

	if err := Mgr.Register(elasticsearch_serviced); err != nil {
		glog.Fatalf("%s", err)
	}
	if err := Mgr.Register(elasticsearch_logstash); err != nil {
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
	if err := Mgr.Register(dockerRegistry); err != nil {
		glog.Fatalf("%s", err)
	}
}

func imagesDir() string {
	return utils.LocalDir("images")
}
