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

package container

import (
	"bytes"
	"fmt"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
	"path/filepath"
)

//createFields makes the map of tags for the logstash config including the type
func createFields(hostID string, hostIPs string, svcPath string, service *service.Service, instanceID string, logConfig *servicedefinition.LogConfig) map[string]string {
	fields := make(map[string]string)
	fields["type"] = logConfig.Type
	fields["service"] = service.ID
	fields["instance"] = instanceID
	fields["hostips"] = hostIPs
	fields["poolid"] = service.PoolID
	fields["servicepath"] = svcPath

	// CC-2234: Note that logstash is hardcoded to inject a field named 'host' into to every message, but when run from within
	// a docker container, the value is actually the container id, not the name of the docker host. So this tag is
	// named a little differently to distinguish it from the tag named 'host'
	fields["ccWorkerID"] = hostID

	for _, tag := range logConfig.LogTags {
		fields[tag.Name] = tag.Value
	}
	return fields
}

//formatTagsForConfFile takes the set of tags for a LogConfig and return json representing the tags
func formatTagsForConfFile(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	buffer.WriteString("{")
	for k, v := range tags {
		buffer.WriteString(k + ": " + v + ", ")
	}
	buffer.WriteString("}")
	return buffer.String()
}

// writeLogstashAgentConfig creates the logstash forwarder config file
func writeLogstashAgentConfig(hostID string, hostIPs string, svcPath string, service *service.Service,
	instanceID string, logforwarderOptions LogforwarderOptions) error {
	resourcePath := filepath.Dir(logforwarderOptions.Path)
	glog.Infof("Using logstash resourcePath: %s", resourcePath)

	// generate the json config.
	filebeatLogConf := ``
	for _, logConfig := range service.LogConfigs {
		filebeatLogConf = filebeatLogConf + `
    - ignore_older: 10s
      close_older: 5m
      paths:
        - %s
      fields: %s`

		filebeatLogConf = fmt.Sprintf(filebeatLogConf, logConfig.Path, formatTagsForConfFile(createFields(hostID, hostIPs, svcPath, service, instanceID, &logConfig)))
	}

	filebeatShipperConf :=
`filebeat:
  idle_timeout: 5s
  prospectors: %s

output:
  logstash:
    enabled: true
    hosts:
      - %s
    tls:
      insecure: true
      certificate: %s
      certificate_key: %s
      certificate_authorities:
        - %s
    timeout: 15

logging:
  level: warning
`

	filebeatShipperConf = fmt.Sprintf(filebeatShipperConf,
		filebeatLogConf,
		"127.0.0.1:5043",
		resourcePath+"/filebeat.crt",
		resourcePath+"/filebeat.key",
		resourcePath+"/filebeat.crt",
	)

	config := servicedefinition.ConfigFile{
		Filename: logforwarderOptions.ConfigFile,
		Content:  filebeatShipperConf,
	}
	err := writeConfFile(config)
	if err != nil {
		return err
	}
	return nil
}
