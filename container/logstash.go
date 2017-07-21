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
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
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

	// generate a prospector configuration for each service log file
	prospectorsConf := ``
	for _, logConfig := range service.LogConfigs {
		prospectorsConf = prospectorsConf + `
    - ignore_older: 10s
      close_inactive: 5m
      paths:
        - %s
      fields: %s`
		prospectorsConf = fmt.Sprintf(prospectorsConf, logConfig.Path,
			formatTagsForConfFile(createFields(hostID, hostIPs, svcPath, service, instanceID, &logConfig)))
	}

	resourcePath := filepath.Dir(logforwarderOptions.Path)
	templatePath := filepath.Join(resourcePath, "filebeat.conf.in")
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		glog.Errorf("Unable to read logforwarder configuration template %s: %s", templatePath, err)
		return err
	}
	glog.Infof("Using logforwarder template from %s", templatePath)

	// Perform all of the template substitution to create the filebeat configuration file from the ".in" template
	filebeatShipperConf := strings.Replace(string(templateContent), "${PROSPECTORS_SECTION}", prospectorsConf, 1)
	filebeatShipperConf = strings.Replace(filebeatShipperConf, "${HOSTS_SECTION}", "127.0.0.1:5043", 1)
	filebeatShipperConf = strings.Replace(filebeatShipperConf, "${CERT_SECTION}", resourcePath+"/filebeat.crt", 1)
	filebeatShipperConf = strings.Replace(filebeatShipperConf, "${CERT_KEY_SECTION}", resourcePath+"/filebeat.key", 1)
	filebeatShipperConf = strings.Replace(filebeatShipperConf, "${CERT_AUTH_SECTION}", resourcePath+"/filebeat.crt", 1)

	config := servicedefinition.ConfigFile{
		Filename: logforwarderOptions.ConfigFile,
		Content:  filebeatShipperConf,
	}

	if err := writeConfFile(config); err != nil {
		glog.Errorf("Unable to create logforwarder configuration file %s: %s", config.Filename, err)
		return err
	}

	glog.V(2).Infof("Created logforwarder configuration at %s", config.Filename)
	return nil
}
