/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"

	"fmt"
	"time"
)

var zookeeper *Container

func init() {

	var err error

	zookeeper, err = NewContainer(
		ContainerDescription{
			Name:        "zookeeper",
			Repo:        IMAGE_REPO,
			Tag:         IMAGE_TAG,
			Command:     "/opt/zookeeper-3.4.5/bin/zkServer.sh start-foreground",
			Ports:       []int{2181, 12181},
			Volumes:     map[string]string{"data": "/tmp"},
			HealthCheck: zkHealthCheck,
		})
	if err != nil {
		glog.Fatal("Error initializing zookeeper container: %s", err)
	}
}

// a health check for zookeeper
func zkHealthCheck() error {

	start := time.Now()
	lastError := time.Now()
	minUptime := time.Second * 2
	timeout := time.Second * 30
	zookeepers := []string{"127.0.0.1:2181"}

	for {
		if conn, _, err := zk.Connect(zookeepers, time.Second*10); err == nil {
			conn.Close()
		} else {
			conn.Close()
			glog.V(1).Infof("Could not connect to zookeeper: %s", err)
			lastError = time.Now()
		}
		// make sure that service has been good for at least minUptime
		if time.Since(lastError) > minUptime {
			break
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("Zookeeper did not respond.")
		}
		time.Sleep(time.Millisecond * 1000)
	}
	glog.Info("zookeeper container started, browser at http://localhost:12181/exhibitor/v1/ui/index.html")
	return nil
}
