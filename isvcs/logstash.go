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

import (
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/proxy"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/control-center/serviced/utils"
)

var logstash *IService

func initLogstash() {
	var err error

	confFile := filepath.Join(utils.LOGSTASH_CONTAINER_DIRECTORY, "logstash.conf")
	command := "export JAVA_HOME=/usr/lib/jvm/jre-11; exec /opt/logstash/bin/logstash -f " + confFile + " --config.reload.automatic"
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

	// Generalize serviced cert and pem files to logstash service according to CC-4136
	logstashPath := filepath.Join(utils.ResourcesDir(), "logstash")
	filebeatCrtPath := filepath.Join(logstashPath, "filebeat.crt")
	filebeatPemPath := filepath.Join(logstashPath, "filebeat.pem")
	globalOptions := config.GetOptions()

	if certPEM, keyPEM, err := proxy.GetKeyPairs(globalOptions.CertPEMFile, globalOptions.KeyPEMFile); err != nil {
		log.WithError(err).Errorf("Unable to retrieve TLS configuration")
	} else {
		if err = ioutil.WriteFile(filebeatCrtPath, certPEM, os.FileMode(0644)); err != nil {
			log.WithError(err).Errorf("Could not write out crt file: %s", filebeatCrtPath)
		}
		if err = ioutil.WriteFile(logstashPath+"/filebeat.pem", keyPEM, os.FileMode(0644)); err != nil {
			log.WithError(err).Errorf("Could not write out pem file: %s", filebeatPemPath)
		}
		log.Info("Re-written filebeat crt and pem files for logstash configuration")
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
			Volumes:    map[string]string{UseServicedLogDir: utils.LOGSTASH_LOCAL_SERVICED_LOG_DIR},
			Links:      []string{"serviced-isvcs_elasticsearch-logstash:elasticsearch"},
			StartGroup: 1,
		})
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize Logstash internal service container")
	}
}
