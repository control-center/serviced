// Copyright 2015 The Serviced Authors.
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

package docker

import (
	"time"

	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

// DockerTTL is the ttl manager for stale docker containers.
type DockerTTL struct{}

// RunTTL starts the ttl to reap stale docker containers.
func RunTTL(cancel <-chan interface{}, min, max time.Duration) {
	utils.RunTTL(DockerTTL{}, cancel, min, max)
}

// Purge cleans up old docker containers and returns the time to live til the
// next purge.
// Implements utils.TTL
func (ttl DockerTTL) Purge(age time.Duration) (time.Duration, error) {
	expire := time.Now().Add(-age)
	ctrs, err := Containers()
	if err != nil {
		glog.Errorf("Could not look up containers: %s", err)
		return 0, err
	}
	for _, ctr := range ctrs {
		if finishTime := ctr.State.FinishedAt; finishTime.Unix() <= 0 || ctr.IsRunning() {
			// container is still running or hasn't started; skip
			continue
		} else if timeToLive := finishTime.Sub(expire); timeToLive <= 0 {
			// container has exceeded its expiration date
			if err := ctr.Delete(true); err != nil {
				glog.Errorf("Could not delete container %s (%s): %s", ctr.Name, ctr.ID, err)
				return 0, err
			}
		} else if timeToLive < age {
			// set the new time to live based on the age of the oldest
			// non-expired snapshot.
			age = timeToLive
		}
	}

	return age, nil
}