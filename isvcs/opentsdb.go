// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
	opentsdb, err = NewContainer(
		ContainerDescription{
			Name:    "opentsdb",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: `cd /opt/zenoss && exec supervisord -n -c /opt/zenoss/etc/supervisor.conf`,
			//only expose 8443 (the consumer port to the host)
			Ports:   []int{4242, 8443, 8888, 9090},
			Volumes: map[string]string{"hbase": "/opt/zenoss/var/hbase"},
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
