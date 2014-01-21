package isvcs

import (
	"github.com/mattbaird/elastigo/cluster"
	"github.com/zenoss/glog"

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

func elasticsearchHealthCheck() error {

	start := time.Now()
	timeout := time.Second * 30

	schemaFile := localDir("resources/controlplane.json")

	for {
		if healthResponse, err := cluster.Health(true); err == nil && (healthResponse.Status == "green" || healthResponse.Status == "yellow") {
			if buffer, err := os.Open(schemaFile); err != nil {
				glog.Fatalf("problem reading %s", err)
			} else {
				http.Post("http://localhost:9200/controlplane", "application/json", buffer)
			}
			break
		} else {
			glog.V(2).Infof("Still trying to connect to elastic: %v: %s", err, healthResponse)
		}
		if time.Since(start) > timeout && time.Since(start) < (timeout/4) {
			return fmt.Errorf("Could not startup elastic search container.")
		}
		time.Sleep(time.Millisecond * 1000)
	}
	return nil
}
