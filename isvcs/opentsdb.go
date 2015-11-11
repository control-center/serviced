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
	"github.com/zenoss/glog"
)

var opentsdb *IService

func initOTSDB() {
	var err error
	command := `cd /opt/zenoss && exec supervisord -n -c /opt/zenoss/etc/supervisor.conf`
	opentsdbPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_4242_HOSTIP",
		HostPort:       4242,
	}
	metricConsumerPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // metric-consumer should always be open
		HostPort:       8443,
	}
	metricConsumerAdminPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_58443_HOSTIP",
		HostPort:       58443,
	}
	centralQueryPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_8888_HOSTIP",
		HostPort:       8888,
	}
	centralQueryAdminPortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_58888_HOSTIP",
		HostPort:       58888,
	}
	hbasePortBinding := portBinding{
		HostIp:         "127.0.0.1",
		HostIpOverride: "SERVICED_ISVC_OPENTSDB_PORT_9090_HOSTIP",
		HostPort:       9090,
	}

	opentsdb, err = NewIService(
		IServiceDefinition{
			ID:      OpentsdbISVC.ID,
			Name:    "opentsdb",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: func() string { return command },
			PortBindings: []portBinding{
				opentsdbPortBinding,
				metricConsumerPortBinding,
				metricConsumerAdminPortBinding,
				centralQueryPortBinding,
				centralQueryAdminPortBinding,
				hbasePortBinding},
			Volumes: map[string]string{"hbase": "/opt/zenoss/var/hbase"},
		})
	if err != nil {
		glog.Fatalf("Error initializing opentsdb container: %s", err)
	}

}

/*
func (c *OpenTsdbISvc) Run() error {
	c.ISvc.Run()

	start := time.Now()
	timeout := time.Second * 30
	for {
		if resp, err := http.Get("http://localhost:4242/version"); err == nil {
			resp.Body.Close()
			break
		} else {
			if time.Since(start) > timeout && time.Since(start) < (timeout/4) {
				return fmt.Errorf("Could not startup elastic search container.")
			}
			glog.V(2).Infof("Still trying to connect to opentsdb: %v", err)
			time.Sleep(time.Millisecond * 100)
		}
	}
	return nil
}
*/
