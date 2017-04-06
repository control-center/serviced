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

var logstash *IService

func initLogstash() {
	var err error

	command := "exec /opt/logstash/bin/logstash agent " +
		"-f /usr/local/serviced/resources/logstash/logstash.conf --auto-reload"
	localFilePortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // logstash should always be open
		HostPort:       5042,
	}
	filebeatPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // filebeat should always be open
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
				filebeatPortBinding,
				webserverPortBinding},
			Volumes:    map[string]string{"servicedLogDir": "/var/log/serviced"},
			Links:      []string{"serviced-isvcs_elasticsearch-logstash:elasticsearch"},
			StartGroup: 1,
		})
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize Logstash internal service container")
	}
}
