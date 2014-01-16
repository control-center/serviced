package isvcs

import "fmt"
import "time"
import "net/http"
import "github.com/zenoss/glog"

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
			Ports:      []int{4242},
		},
	}
}

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
