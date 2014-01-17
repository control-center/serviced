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
		NewISvc(
			"opentsdb",
			"zctrl/isvcs",
			"v1",
			"/bin/bash -c \"cd /opt/zenoss && supervisord -n -c /opt/zenoss/etc/supervisor.conf\"",
			[]int{4242, 8443, 9090, 60000, 60010, 60020, 60030},
			[]string{"/opt/zenoss/var/hbase"},
		),
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
