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

import "github.com/zenoss/glog"

var logstash *IService

func initLogstash() {
	var err error

	command := "exec /opt/logstash-2.2.2/bin/logstash agent -f /usr/local/serviced/resources/logstash/logstash.conf"
	localFilePortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // logstash should always be open
		HostPort:       5042,
	}
	lumberJackPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // lumberjack should always be open
		HostPort:       5043,
	}
	webserverPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_LOGSTASH_PORT_9292_HOSTIP",
		HostPort:       9292,
	}

	logstash, err = NewIService(
		IServiceDefinition{
			ID:      LogstashISVC.ID,
			Name:    "logstash",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: func() string { return command },
			PortBindings: []portBinding{
				localFilePortBinding,
				lumberJackPortBinding,
				webserverPortBinding},
			Volumes:    map[string]string{},
			Notify:     notifyLogstashConfigChange,
			Links:      []string{"serviced-isvcs_elasticsearch-logstash:elasticsearch"},
			StartGroup: 1,
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
