// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
