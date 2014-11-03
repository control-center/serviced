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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/zenoss/glog"
)

var opentsdb *Container

func init() {
	var err error
	command := `cd /opt/zenoss && exec supervisord -n -c /opt/zenoss/etc/supervisor.conf`
	opentsdb, err = NewContainer(
		ContainerDescription{
			Name:    "opentsdb",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: func() string { return command },
			//only expose 8443 (the consumer port to the host)
			Ports:       []int{4242, 8443, 8888, 9090},
			Volumes:     map[string]string{"hbase": "/opt/zenoss/var/hbase"},
			HostNetwork: false,
		})
	if err != nil {
		glog.Fatal("Error initializing opentsdb container: %s", err)
	}

}

/*
func (c *OpenTsdbISvc) Run() error {
	c.ISvc.Run()

	start := time.Now()
	timeout := time.Second * 30
	for {
		if _, err := http.Get("http://localhost:4242/version"); err == nil {
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
