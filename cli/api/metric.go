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

package api

import (
	"fmt"
	"time"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/stats"
	"github.com/zenoss/glog"
)

//
func (a *api) PostMetric(metricName string, metricValue string) (string, error) {
	url := fmt.Sprintf("http://%s/api/metrics/store", options.HostStats)
	timeStamp := time.Now().Unix()
	hostId, err := utils.HostID()
	if err != nil {
		glog.Errorf("Error getting host id, error: %s", err)
		return "", err
	}

	samples := make([]stats.Sample, 1)
	samples[0] = stats.Sample{
		Metric:    metricName,
		Value:     metricValue,
		Timestamp: timeStamp,
		Tags:      map[string]string{"controlplane_host_id": hostId},
	}

	if err := stats.Post(url, samples); err != nil {
		glog.Errorf("could not post stats: %s", err)
		return "", err
	}
	return "Posted metric", nil
}
