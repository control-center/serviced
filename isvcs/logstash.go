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
		ISvc{
			Name:       "logstash_master",
			Repository: "zctrl/logstash_master",
			Tag:        "v1",
			Ports:      []int{5043, 9292},
		},
	}
}


func (c *LogstashISvc) StartService(templates map[string]*dao.ServiceTemplate) error {
	err := WriteConfigurationFile(templates)

	if err != nil {
		return err
	}

	// start up the service
	err = c.ISvc.Run()
	if err != nil {
		return err
	}

	start := time.Now()
	timeout := time.Second * 30
	for {
		_, err = http.Get("http://localhost:9292/")
		if err == nil {
			break
		}
		running, err := c.Running()
		if !running {
			glog.Errorf("Logstash container stopped: %s", err)
			return err
		}
		if time.Since(start) > timeout {
			glog.Errorf("Timeout starting up logstash container")
			return fmt.Errorf("Could not startup logstash container.")
		}
		glog.V(2).Infof("Still trying to connect to logstash: %v", err)
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}
