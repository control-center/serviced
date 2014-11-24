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
	"encoding/json"
	"fmt"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

const (
	logstashContainerConfig = "/etc/logstash-forwarder.conf"
)

//createFields makes the map of tags for the logstash config including the type
func createFields(service *service.Service, instanceID string, logConfig *servicedefinition.LogConfig) map[string]string {
	fields := make(map[string]string)
	fields["type"] = logConfig.Type
	fields["service"] = service.ID
	fields["instance"] = instanceID
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
	result, err := json.Marshal(tags)
	if err != nil {
		glog.Warningf("Unable to unmarhsal %s because of %s", tags, err)
		return ""
	}
	return string(result)
}

// writeLogstashAgentConfig creates the logstash forwarder config file
func writeLogstashAgentConfig(confPath string, service *service.Service, instanceID string, resourcePath string) error {
	glog.Infof("Using logstash resourcePath: %s", resourcePath)

	// generate the json config.
	// TODO: Grab the structs from logstash-forwarder and marshal this instead of generating it
	logstashForwarderLogConf := `
		{
			"paths": [ "%s" ],
			"fields": %s
		}`
	logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, service.LogConfigs[0].Path, formatTagsForConfFile(createFields(service, instanceID, &service.LogConfigs[0])))
	for _, logConfig := range service.LogConfigs[1:] {
		logstashForwarderLogConf = logstashForwarderLogConf + `,
				{
					"paths": [ "%s" ],
					"fields": %s
				}`
		logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, logConfig.Path, formatTagsForConfFile(createFields(service, instanceID, &logConfig)))
	}

	logstashForwarderShipperConf := `
			{
				"network": {
					"servers": [ "%s" ],
					"ssl certificate": "%s",
					"ssl key": "%s",
					"ssl ca": "%s",
					"timeout": 15
				},
				"files": [
					%s
				]
			}`
	logstashForwarderShipperConf = fmt.Sprintf(logstashForwarderShipperConf,
		//		"172.17.42.1:5043",
		"127.0.0.1:5043",
		resourcePath+"/logstash-forwarder.crt",
		resourcePath+"/logstash-forwarder.key",
		resourcePath+"/logstash-forwarder.crt",
		logstashForwarderLogConf)

	config := servicedefinition.ConfigFile{
		Filename: confPath,
		Content:  logstashForwarderShipperConf,
	}
	err := writeConfFile(config)
	if err != nil {
		return err
	}
	return nil
}
