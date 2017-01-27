// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/logging"
)

var (
	plog = logging.PackageLogger()
)

type ProgressCounter struct {
	Total                 uint64
	Message               string
	lastUpdate            time.Time
	updateIntervalSeconds int
}

func (pc *ProgressCounter) Write(data []byte) (int, error) {

	pc.Total += uint64(len(data))

	timeSinceUpdate := int(time.Since(pc.lastUpdate).Seconds())
	if timeSinceUpdate > pc.updateIntervalSeconds {
		plog.Info(fmt.Sprintf(pc.Message, pc.Total))
		pc.lastUpdate = time.Now()
	}
	return len(data), nil
}

func NewProgressCounter(interval int, message string) *ProgressCounter {
	if len(message) == 0 {
		message = "Processed %v Total Bytes."
	}

	return &ProgressCounter{
		lastUpdate:            time.Now(),
		updateIntervalSeconds: interval,
		Message:               message,
	}
}
