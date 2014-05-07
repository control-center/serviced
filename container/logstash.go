// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.
package container

import (
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"

	"encoding/json"
	"fmt"
	"github.com/zenoss/glog"
)

//createFields makes the map of tags for the logstash config including the type
func createFields(logConfig *servicedefinition.LogConfig) map[string]string {
	fields := make(map[string]string)
	fields["type"] = logConfig.Type
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
func writeLogstashAgentConfig(service *service.Service, resourcePath string) error {
	glog.Infof("Using logstash resourcePath: %s", resourcePath)

	// generate the json config.
	// TODO: Grab the structs from logstash-forwarder and marshal this instead of generating it
	logstashForwarderLogConf := `
		{
			"paths": [ "%s" ],
			"fields": %s
		}`
	logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, service.LogConfigs[0].Path, formatTagsForConfFile(createFields(&service.LogConfigs[0])))
	for _, logConfig := range service.LogConfigs[1:] {
		logstashForwarderLogConf = logstashForwarderLogConf + `,
				{
					"paths": [ "%s" ],
					"fields": %s
				}`
		logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, logConfig.Path, formatTagsForConfFile(createFields(&logConfig)))
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
		"172.17.42.1:5043",
		resourcePath+"/logstash-forwarder.crt",
		resourcePath+"/logstash-forwarder.key",
		resourcePath+"/logstash-forwarder.crt",
		logstashForwarderLogConf)

	config := servicedefinition.ConfigFile{
		Filename: "/etc/logstash-forwarder.conf",
		Content:  logstashForwarderShipperConf,
	}
	err := writeConfFile(config)
	if err != nil {
		return err
	}
	return nil
}
