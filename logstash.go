/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/
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

func formatTagsForConfFile(tags *map[string]string) string {
	if len(*tags) == 0 {
		return ""
	}
	result, err := json.Marshal(*tags)
	if err != nil {
		glog.Warningf("Unable to unmarhsal %s because of %s", *tags, err)
		return ""
	}
	return string(result)
}

// writeLogstashAgentConfig creates the logstash forwarder config file and places it in a temp directory
// the filename of the newly created file is returned
func writeLogstashAgentConfig(service *dao.Service) (string, error) {

	service.LogConfigs[0].Tags["type"] = service.LogConfigs[0].Type
	logstashForwarderLogConf := `
		{
			"paths": [ "%s" ],
			"fields": %s
		}`
	logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, service.LogConfigs[0].Path, formatTagsForConfFile(&service.LogConfigs[0].Tags))
	for _, logConfig := range service.LogConfigs[1:] {
		logConfig.Tags["type"] = logConfig.Type
		logstashForwarderLogConf = logstashForwarderLogConf + `,
				{
					"paths": [ "%s" ],
					"fields": %s
				}`
		logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, logConfig.Path, formatTagsForConfFile(&logConfig.Tags))
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
		// TODO: Don't hard code this and in fact the whole thing won't work at all
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
