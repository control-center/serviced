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

package metrics

import (
	"fmt"
	"time"

	"github.com/zenoss/glog"
)

type ServiceInstance struct {
	ServiceID  string
	InstanceID int
}

type MemoryUsageStats struct {
	StartDate  time.Time
	EndDate    time.Time
	HostID     string
	ServiceID  string
	InstanceID string
	Last       int64
	Max        int64
	Average    int64
}

func convertMemoryUsage(data *PerformanceData) []MemoryUsageStats {
	mems := make([]MemoryUsageStats, len(data.Results))
	for i, result := range data.Results {
		mems[i] = MemoryUsageStats{
			StartDate: time.Unix(data.StartTimeActual, 0),
			EndDate:   time.Unix(data.EndTimeActual, 0),
		}

		for tag, value := range result.Tags {
			switch tag {
			case "controlplane_host_id":
				mems[i].HostID = value[0]
			case "controlplane_service_id":
				mems[i].ServiceID = value[0]
			case "controlplane_instance_id":
				mems[i].InstanceID = value[0]
			}
		}

		var last, sum, max float64
		for _, dp := range result.Datapoints {
			if last = dp.Value; last > max {
				max = last
			}
			sum += dp.Value
		}
		mems[i].Last = int64(last)
		mems[i].Max = int64(max)
		mems[i].Average = int64(sum / float64(len(result.Datapoints)))
	}
	return mems
}

func (c *Client) GetHostMemoryStats(startDate time.Time, hostID string) (*MemoryUsageStats, error) {
	options := PerformanceOptions{
		Start:     startDate.Format(timeFormat),
		End:       "now",
		Returnset: "exact",
		Tags: map[string][]string{
			"controlplane_host_id": []string{hostID},
		},
		Metrics: []MetricOptions{
			{
				Metric:     "cgroup.memory.totalrss",
				Name:       hostID,
				Aggregator: "sum",
			},
		},
	}

	result, err := c.performanceQuery(options)
	if err != nil {
		glog.Errorf("Could not get performance data for host %s: %s", hostID, err)
		return nil, err
	}

	mems := convertMemoryUsage(result)
	if len(mems) < 1 {
		err := fmt.Errorf("no data found")
		return nil, err
	}

	return &mems[0], nil
}

func (c *Client) GetServiceMemoryStats(startDate time.Time, serviceID string) (*MemoryUsageStats, error) {
	options := PerformanceOptions{
		Start:     startDate.Format(timeFormat),
		End:       "now",
		Returnset: "exact",
		Tags: map[string][]string{
			"controlplane_service_id": []string{serviceID},
		},
		Metrics: []MetricOptions{
			{
				Metric:     "cgroup.memory.totalrss",
				Name:       serviceID,
				Aggregator: "max",
			},
		},
	}

	result, err := c.performanceQuery(options)
	if err != nil {
		glog.Errorf("Could not get performance data for service %s: %s", serviceID, err)
		return nil, err
	}

	mems := convertMemoryUsage(result)
	if len(mems) < 1 {
		err := fmt.Errorf("no data found")
		return nil, err
	}

	return &mems[0], nil
}

func (c *Client) GetInstanceMemoryStats(startDate time.Time, instances ...ServiceInstance) ([]MemoryUsageStats, error) {
	options := PerformanceOptions{
		Start:     startDate.Format(timeFormat),
		End:       "now",
		Returnset: "exact",
	}

	metrics := make([]MetricOptions, len(instances))
	for i, instance := range instances {
		metrics[i] = MetricOptions{
			Metric: "cgroup.memory.totalrss",
			Name:   fmt.Sprintf("%s.%d", instance.ServiceID, instance.InstanceID),
			Tags: map[string][]string{
				"controlplane_service_id":  []string{instance.ServiceID},
				"controlplane_instance_id": []string{fmt.Sprintf("%d", instance.InstanceID)},
			},
		}
	}
	options.Metrics = metrics

	result, err := c.performanceQuery(options)
	if err != nil {
		glog.Errorf("Could not get performance data for instances %+v: %s", instances, err)
		return nil, err
	}

	return convertMemoryUsage(result), nil
}