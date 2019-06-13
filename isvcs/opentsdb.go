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

var opentsdb *IService

func initOTSDB(bigtable bool) {
	var err error
	var portBindings []portBinding

	command := `cd /opt/zenoss && exec supervisord -n -c /opt/zenoss/etc/supervisor.conf`
	opentsdbPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_4242_HOSTIP",
		HostPort:       4242,
	}
	portBindings = append(portBindings, opentsdbPortBinding)
	metricConsumerPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // metric-consumer should always be open
		HostPort:       8443,
	}
	portBindings = append(portBindings, metricConsumerPortBinding)
	metricConsumerAdminPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_58443_HOSTIP",
		HostPort:       58443,
	}
	portBindings = append(portBindings, metricConsumerAdminPortBinding)
	centralQueryPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_8888_HOSTIP",
		HostPort:       8888,
	}
	portBindings = append(portBindings, centralQueryPortBinding)
	centralQueryAdminPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_58888_HOSTIP",
		HostPort:       58888,
	}
	portBindings = append(portBindings, centralQueryAdminPortBinding)
	hbasePortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_9090_HOSTIP",
		HostPort:       9090,
	}
	if !bigtable{
		portBindings = append(portBindings, hbasePortBinding)
	}

	volumes := map[string]string{}
	if !bigtable{
		volumes = map[string]string{"hbase": "/opt/zenoss/var/hbase"}
	}
	otsdbImage := IMAGE_REPO
	otsdbTag := IMAGE_TAG
    if bigtable {
        otsdbImage = OTSDB_BT_REPO
        otsdbTag = OTSDB_BT_TAG
    }
	opentsdb, err = NewIService(
		IServiceDefinition{
			ID:      OpentsdbISVC.ID,
			Name:    "opentsdb",
			Repo:    otsdbImage,
			Tag:     otsdbTag,
			Command: func() string { return command },
			PortBindings: portBindings,
			Volumes: volumes,
			CustomStats:  GetOpenTSDBCustomStats,
		})
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize opentsdb internal service container")
	}

}
