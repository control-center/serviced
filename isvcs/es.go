package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"net/http"
	"time"
)

type ElasticSearchISvc struct {
	ISvc
}

var ElasticSearchContainer ElasticSearchISvc

func init() {
	ElasticSearchContainer = ElasticSearchISvc{
		ISvc{
			Name:       "elasticsearch",
			Repository: "zctrl/es",
			Tag:        "v1",
			Ports:      []int{9200},
		},
	}
}

func (c *ElasticSearchISvc) Run() error {
	err := c.ISvc.Run()
	if err != nil {
		return err
	}

	start := time.Now()
	timeout := time.Second * 30
	for {
		_, err = http.Get("http://localhost:9200/")
		if err == nil {
			break
		}
		if time.Since(start) > timeout && time.Since(start) < (timeout / 4) {
			return fmt.Errorf("Could not startup elastic search container.")
		}
		glog.V(2).Infof("Still trying to connect to elastic: %v", err)
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
