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
	"github.com/zenoss/glog"
)

var logstash *IService

func init() {
	var err error
	command := "exec /opt/logstash-1.4.2/bin/logstash agent -f /usr/local/serviced/resources/logstash/logstash.conf"
	logstash, err = NewIService(
		IServiceDefinition{
			Name:    "logstash",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: func() string { return command },
			Ports:   []uint16{5042, 5043, 9292},
			Volumes: map[string]string{},
			Notify:  notifyLogstashConfigChange,
		})
	if err != nil {
		glog.Fatalf("Error initializing logstash_master container: %s", err)
	}
}

func notifyLogstashConfigChange(svc *IService, value interface{}) error {

	if message, ok := value.(string); ok {
		if message == "restart logstash" {
			return svc.Restart()
		}
	}
	return nil
}
