// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package isvcs

import (
	"github.com/zenoss/glog"
)

var celery *Container

func init() {
	var err error
	command := "supervisord -n -c /opt/celery/etc/supervisor.conf"
	celery, err = NewContainer(
		ContainerDescription{
			Name:    "celery",
			Repo:    IMAGE_REPO,
			Tag:     IMAGE_TAG,
			Command: func() string {return command},
			Ports:   []int{},
			Volumes: map[string]string{"celery": "/opt/celery/var"},
		})
	if err != nil {
		glog.Fatal("Error initializing celery container: %s", err)
	}
}
