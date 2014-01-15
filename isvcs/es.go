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
		NewISvc(
			"elasticsearch",
			"zctrl/es",
			"v1",
			[]int{9200},
			[]string{"/opt/elasticsearch-0.90.5/data"},
		),
	}
}

func (c *ElasticSearchISvc) Run() error {
	c.ISvc.Run()

	start := time.Now()
	timeout := time.Second * 30
	for {
		if _, err := http.Get("http://localhost:9200/"); err == nil {
			break
		} else {
			glog.V(2).Infof("Still trying to connect to elastic: %v", err)
		}
		if time.Since(start) > timeout && time.Since(start) < (timeout/4) {
			return fmt.Errorf("Could not startup elastic search container.")
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
