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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
)

// instantiate the package logger
var plog = logging.PackageLogger()

// TTL manages time-to-live data
type TTL interface {
	// Name identifies the TTL instance
	Name() string

	// Purge purges expired data from source
	Purge(time.Duration) (time.Duration, error)
}

// RunTTL purges expired data based upon the time interval
func RunTTL(ttl TTL, cancel <-chan interface{}, min, max time.Duration) {

	logger := plog.WithFields(log.Fields{
		"name": ttl.Name(),
		"min":  int(min.Minutes()),
		"max":  int(max.Minutes()),
	})
	logger.Debug("Start TTL routine")
	original_min := min
	for {
		var repeatSooner bool
		wait, err := ttl.Purge(max)
		if err != nil {
			wait = min
			repeatSooner = true
			logger.WithField("wait", int(wait.Minutes())).WithError(err).
				Warning("Could not purge; trying again after wait period")
		} else {
			min = original_min
			repeatSooner = false
		}

		logger.WithField("wait", int(wait.Minutes())).Debug("Waiting for next purge cycle")
		select {
		case <-time.After(wait):
			// Use exponential backoff if repeating check after a failure
			if repeatSooner {
				min = 2 * min
				if min > max {
					min = max
				}
			}
		case <-cancel:
			return
		}
	}
}
