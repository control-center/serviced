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

var kibana *IService

func initKibana() {
	var err error
	command := "exec /opt/kibana/bin/kibana --allow-root"

	webserverPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_KIBANA_PORT_5601_HOSTIP",
		HostPort:       5601,
	}

	kibana, err = NewIService(
		IServiceDefinition{
			ID:           KibanaISVC.ID,
			Name:         "kibana",
			Repo:         IMAGE_REPO,
			Tag:          IMAGE_TAG,
			Command:      func() string { return command },
			PortBindings: []portBinding{webserverPortBinding},
			Volumes:      map[string]string{},
			Links:        []string{"serviced-isvcs_elasticsearch-logstash:elasticsearch"},
			StartGroup:   1,
		})
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize Kibana internal service container")
	}
}
