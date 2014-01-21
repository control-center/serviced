package isvcs

var opentsdb ContainerDescription

func init() {
	opentsdb = ContainerDescription{
		Name:    "opentsdb",
		Repo:    "zctrl/isvcs",
		Tag:     "v1",
		Command: `/bin/bash -c "cd /opt/zenoss && supervisord -n -c /opt/zenoss/etc/supervisor.conf"`,
		Ports:   []int{4242, 8443, 8888, 9090, 60000, 60010, 60020, 60030},
		Volumes: []string{"/opt/zenoss/var/hbase"},
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
