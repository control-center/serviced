// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/zenoss/glog"
)

var logstash *Container

func init() {
	var err error
	logstash, err = NewContainer(
		ContainerDescription{
			Name:    "logstash",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: "java -jar /opt/logstash/logstash-1.3.2-flatjar.jar agent -f /usr/local/serviced/resources/logstash/logstash.conf -- web",
			Ports:   []int{5043, 9292},
			Volumes: map[string]string{},
			Notify:  notifyLogstashConfigChange,
		})
	if err != nil {
		glog.Fatal("Error initializing logstash_master container: %s", err)
	}
}

func notifyLogstashConfigChange(c *Container, value interface{}) error {

	if message, ok := value.(string); ok {
		if message == "restart logstash" {
			c.Stop()
			return c.Start()
		}
	}
	return nil
}
