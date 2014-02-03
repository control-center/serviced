/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014 all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"fmt"
	"github.com/mattbaird/elastigo/cluster"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/utils"
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
