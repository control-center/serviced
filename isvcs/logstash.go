package isvcs

import (
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"net/http"
	"time"
)

type LogstashISvc struct {
	ISvc
}

var LogstashContainer LogstashISvc

func init() {
	LogstashContainer = LogstashISvc{
		NewISvc(
			"logstash_master",
			"zctrl/logstash_master",
			"v1",
			[]int{5043, 9292},
			[]string{},
		),
	}
}

func (c *LogstashISvc) StartService(templates map[string]*dao.ServiceTemplate) error {
	err := WriteConfigurationFile(templates)

	if err != nil {
		return err
	}

	// start up the service
	c.ISvc.Run()

	start := time.Now()
	timeout := time.Second * 30
	for {
		if _, err := http.Get("http://localhost:9292/"); err == nil {
			break
		} else {
			glog.V(2).Infof("Still trying to connect to logstash: %v", err)
		}
		if time.Since(start) > timeout {
			glog.Errorf("Timeout starting up logstash container")
			return fmt.Errorf("Could not startup logstash container.")
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
