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

package utils

import (
	"time"

	"github.com/zenoss/glog"
)

// TTL manages time-to-live data
type TTL interface {
	// Purge purges expired data from source
	Purge(time.Duration) (time.Duration, error)
}

// RunTTL purges expired data based upon the time interval
func RunTTL(ttl TTL, cancel <-chan interface{}, min, max time.Duration) {
	for {
		wait, err := ttl.Purge(max)
		if err != nil {
			glog.Warningf("Could not purge: %s", err)
			wait = min
		}

		glog.V(1).Info("Next purge in %s", wait)
		select {
		case <-time.After(wait):
		case <-cancel:
			return
		}
	}
}