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

var celery *IService

func initCelery() {
	var err error
	command := "exec supervisord -n -c /opt/celery/etc/supervisor.conf"
	celery, err = NewIService(
		IServiceDefinition{
			ID:           CeleryISVC.ID,
			Name:         "celery",
			Repo:         IMAGE_REPO,
			Tag:          IMAGE_TAG,
			Command:      func() string { return command },
			PortBindings: []portBinding{},
			Volumes:      map[string]string{"celery": "/opt/celery/var"},
		})
	if err != nil {
		glog.Fatalf("Error initializing celery container: %s", err)
	}
}
