package isvcs

import (
	"github.com/zenoss/glog"
)

var logstash *Container

func init() {
	var err error
	logstash, err = NewContainer(
		ContainerDescription{
			Name:    "logstash",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: "java -jar /opt/logstash/logstash-1.3.2-flatjar.jar agent -f /usr/local/serviced/resources/logstash/logstash.conf -- web",
			Ports:   []int{5043, 9292},
			Volumes: map[string]string{},
		})
	if err != nil {
		glog.Fatal("Error initializing logstash_master container: %s", err)
	}
}

/*
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
*/
