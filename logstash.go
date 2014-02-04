// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.
package serviced

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/utils"
	"strings"
)

const (

	// this is the directory where the logstash forwarder config file is from within the container
	LOGSTASH_CONTAINER_CONFIG = "/etc/logstash-forwarder.conf"
	// this is the directory where we keep the executable and cert files for logstash forwarder
	LOGSTASH_CONTAINER_DIRECTORY = "/usr/local/serviced/resources/logstash"
)

//createFields makes the map of tags for the logstash config including the type
func createFields(logConfig *dao.LogConfig) map[string]string {
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

// writeLogstashAgentConfig creates the logstash forwarder config file and places it in a temp directory
// the filename of the newly created file is returned
func writeLogstashAgentConfig(service *dao.Service) (string, error) {

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
		LOGSTASH_CONTAINER_DIRECTORY+"/logstash-forwarder.crt",
		LOGSTASH_CONTAINER_DIRECTORY+"/logstash-forwarder.key",
		LOGSTASH_CONTAINER_DIRECTORY+"/logstash-forwarder.crt",
		logstashForwarderLogConf)
	filename := service.Name + "_logstash_forwarder_conf"
	prefix := fmt.Sprintf("cp_%s_%s_", service.Id, strings.Replace(filename, "/", "__", -1))
	f, err := writeConfFile(prefix, service.Id, filename, logstashForwarderShipperConf)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

//getLogstashBindMounts This sets up the logstash config and returns the bind mounts
func getLogstashBindMounts(configFileName string) string {

	// the local file system path
	logstashPath := utils.ResourcesDir() + "/logstash"

	return fmt.Sprintf("-v %s:%s -v %s:%s",
		logstashPath,
		LOGSTASH_CONTAINER_DIRECTORY,
		configFileName,
		LOGSTASH_CONTAINER_CONFIG)
}
