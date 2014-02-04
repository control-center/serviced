// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/mattbaird/elastigo/cluster"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/utils"

	"fmt"
	"net/http"
	"os"
	"time"
)

var elasticsearch *Container

func init() {
	var err error
	elasticsearch, err = NewContainer(
		ContainerDescription{
			Name:        "elasticsearch",
			Repo:        IMAGE_REPO,
			Tag:         IMAGE_TAG,
			Command:     `/opt/elasticsearch-0.90.9/bin/elasticsearch -f`,
			Ports:       []int{9200},
			Volumes:     map[string]string{"data": "/opt/elasticsearch-0.90.9/data"},
			HealthCheck: elasticsearchHealthCheck,
		},
	)
	if err != nil {
		glog.Fatal("Error initializing zookeeper container: %s", err)
	}
}

// elasticsearchHealthCheck() determines if elasticsearch is healthy
func elasticsearchHealthCheck() error {

	start := time.Now()
	lastError := time.Now()
	minUptime := time.Second * 2
	timeout := time.Second * 30

	schemaFile := utils.LocalDir("resources/controlplane.json")

	for {
		if healthResponse, err := cluster.Health(true); err == nil && (healthResponse.Status == "green" || healthResponse.Status == "yellow") {
			if buffer, err := os.Open(schemaFile); err != nil {
				glog.Fatalf("problem reading %s", err)
				return err
			} else {
				http.Post("http://localhost:9200/controlplane", "application/json", buffer)
				buffer.Close()
			}
		} else {
			lastError = time.Now()
			glog.V(2).Infof("Still trying to connect to elastic: %v: %s", err, healthResponse)
		}
		if time.Since(lastError) > minUptime {
			break
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("Could not startup elastic search container.")
		}
		time.Sleep(time.Millisecond * 1000)
	}
	glog.Info("elasticsearch container started, browser at http://localhost:9200/_plugin/head/")
	return nil
}
