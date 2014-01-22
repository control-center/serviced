/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014 all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
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
			Reload:  reload,
		})
	if err != nil {
		glog.Fatal("Error initializing logstash_master container: %s", err)
	}
}

func reload(c *Container, value interface{}) error {

	if templates, ok := value.(map[string]*dao.ServiceTemplate); ok {
		if err := WriteConfigurationFile(templates); err != nil {
			return err
		}
		c.Stop()
		return c.Start()
	}
	return nil
}
