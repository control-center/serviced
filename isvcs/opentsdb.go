package isvcs

import (
	"fmt"
	"github.com/zenoss/glog"
	"net/http"
	"time"
)

type OpenTsdbISvc struct {
	ISvc
}

var OpenTsdbContainer OpenTsdbISvc

func init() {
	OpenTsdbContainer = OpenTsdbISvc{
		ISvc{
			Name:       "opentsdb",
			Repository: "zctrl/opentsdb",
			Tag:        "v1",
			Ports:      []int{4242, 8443, 9090, 60000, 60010, 60020, 60030},
		},
	}
}

func (c *OpenTsdbISvc) Run() error {
	err := c.ISvc.Run()
	if err != nil {
		return err
	}

	start := time.Now()
	timeout := time.Second * 30
	for {
		_, err = http.Get("http://localhost:4242/version")
		if err == nil {
			break
		}
		if time.Since(start) > timeout && time.Since(start) < (timeout/4) {
			return fmt.Errorf("Could not startup elastic search container.")
		}
		glog.V(2).Infof("Still trying to connect to opentsdb: %v", err)
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
