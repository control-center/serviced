package isvcs

import (
	"github.com/zenoss/glog"
)

var opentsdb *Container

func init() {
	var err error
	opentsdb, err = NewContainer(
		ContainerDescription{
			Name:    "opentsdb",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: `cd /opt/zenoss && supervisord -n -c /opt/zenoss/etc/supervisor.conf`,
			Ports:   []int{4242, 8443, 8888, 9090, 60000, 60010, 60020, 60030},
			Volumes: map[string]string{"hbase": "/opt/zenoss/var/hbase"},
		})
	if err != nil {
		glog.Fatal("Error initializing opentsdb container: %s", err)
	}

}

/*
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
*/
