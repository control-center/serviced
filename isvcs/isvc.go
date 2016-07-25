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

package isvcs

import (
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"

	"time"
)

var Mgr *Manager

const (
	IMAGE_REPO    = "zenoss/serviced-isvcs"
	IMAGE_TAG     = "v44"
	ZK_IMAGE_REPO = "zenoss/isvcs-zookeeper"
	ZK_IMAGE_TAG  = "v4"
)

func Init(esStartupTimeoutInSeconds int, dockerLogDriver string, dockerLogConfig map[string]string, dockerAPI docker.Docker) {
	elasticsearch_serviced.StartupTimeout = time.Duration(esStartupTimeoutInSeconds) * time.Second
	elasticsearch_logstash.StartupTimeout = time.Duration(esStartupTimeoutInSeconds) * time.Second

	Mgr = NewManager(utils.LocalDir("images"), utils.TempDir("var/isvcs"), dockerLogDriver, dockerLogConfig)

	elasticsearch_serviced.docker = dockerAPI
	if err := Mgr.Register(elasticsearch_serviced); err != nil {
		glog.Fatalf("%s", err)
	}
	elasticsearch_logstash.docker = dockerAPI
	if err := Mgr.Register(elasticsearch_logstash); err != nil {
		glog.Fatalf("%s", err)
	}
	zookeeper.docker = dockerAPI
	if err := Mgr.Register(zookeeper); err != nil {
		glog.Fatalf("%s", err)
	}
	logstash.docker = dockerAPI
	if err := Mgr.Register(logstash); err != nil {
		glog.Fatalf("%s", err)
	}
	opentsdb.docker = dockerAPI
	if err := Mgr.Register(opentsdb); err != nil {
		glog.Fatalf("%s", err)
	}
	celery.docker = dockerAPI
	if err := Mgr.Register(celery); err != nil {
		glog.Fatalf("%s", err)
	}
	dockerRegistry.docker = dockerAPI
	if err := Mgr.Register(dockerRegistry); err != nil {
		glog.Fatalf("%s", err)
	}
}

func InitServices(isvcNames []string, dockerLogDriver string, dockerLogConfig map[string]string, dockerAPI docker.Docker) {
	Mgr = NewManager(utils.LocalDir("images"), utils.TempDir("var/isvcs"), dockerLogDriver, dockerLogConfig)
	for _, isvcName := range isvcNames {
		switch isvcName {
		case "zookeeper":
			zookeeper.docker = dockerAPI
			if err := Mgr.Register(zookeeper); err != nil {
				glog.Fatalf("%s", err)
			}
		}
	}
}
