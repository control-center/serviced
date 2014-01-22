package isvcs

import (
	"github.com/zenoss/glog"
)

var celery *Container

func init() {
	var err error
	celery, err = NewContainer(
		ContainerDescription{
			Name:    "celery",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: "supervisord -n -c /opt/celery/etc/supervisor.conf",
			Ports:   []int{},
			Volumes: map[string]string{"celery": "/opt/celery/var"},
		})
	if err != nil {
		glog.Fatal("Error initializing celery container: %s", err)
	}
}
