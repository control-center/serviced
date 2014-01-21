package isvcs

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mattbaird/elastigo/cluster"
	"github.com/zenoss/glog"
	"net/http"
	"os"
	"time"
)

type ElasticSearchISvc struct {
	ISvc
}

var ElasticSearchContainer ElasticSearchISvc

func init() {
	ElasticSearchContainer = ElasticSearchISvc{
		NewISvc(
			"elasticsearch",
			"zctrl/isvcs",
			"v1",
			"/opt/elasticsearch-0.90.9/bin/elasticsearch -f",
			[]int{9200},
			[]string{"/opt/elasticsearch-0.90.9/data"},
		),
	}
}

func (c *ElasticSearchISvc) Run() error {
	c.ISvc.Run()

	start := time.Now()
	timeout := time.Second * 30

	schemaFile := localDir("resources/controlplane.json")

	for {
		if healthResponse, err := cluster.Health(true); err == nil && (healthResponse.Status == "green" || healthResponse.Status == "yellow") {
			if buffer, err := os.Open(schemaFile); err != nil {
				glog.Fatalf("problem reading %s", err)
				return err
			} else {
				postResp, postErr := http.Post("http://localhost:9200/controlplane", "application/json", buffer)
				if postErr != nil {
					glog.Infof("Post schema failed: Err=%s, StatusCode=%d", postErr, postResp.StatusCode)
					return postErr
				}
				if postResp.StatusCode != 200 {
					body := new(bytes.Buffer)
					body.ReadFrom(postResp.Body)
					bodyString := body.String()
					if bodyString != "{\"error\":\"IndexAlreadyExistsException[[controlplane] already exists]\",\"status\":400}" {
						glog.Infof("Post schema failed: statuscode=%d, body=%s", postResp.StatusCode, bodyString)
						return errors.New(bodyString)
					}
				}
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
