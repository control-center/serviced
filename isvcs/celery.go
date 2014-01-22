package isvcs

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/zenoss/glog"
	"time"
)

type CeleryISvc struct {
	ISvc
}

var CeleryContainer CeleryISvc

func init() {
	CeleryContainer = CeleryISvc{
		NewISvc(
			"celery",
			"zctrl/isvcs",
			"v2",
			"supervisord -n -c /opt/celery/etc/supervisor.conf",
			[]int{6379},
			[]string{"/opt/celery/var"},
		),
	}
}

func (c *CeleryISvc) Run() error {
	c.ISvc.Run()

	start := time.Now()
	timeout := time.Second * 30

	for {
		if _, err := redis.Dial("tcp", ":6379"); err == nil {
			break
		} else {
			if time.Since(start) > timeout && time.Since(start) < (timeout/4) {
				return fmt.Errorf("Could not startup elastic search container.")
			}
			glog.V(2).Info("Still trying to connect to Celery broker")
		}
		time.Sleep(time.Millisecond * 1000)
	}
	return nil
}
