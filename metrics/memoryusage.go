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
	"strconv"
	"strings"
	"time"

	"github.com/zenoss/glog"
)

var (
	cache = NewMemoryUsageCache(time.Minute)
)

type ServiceInstance struct {
	ServiceID  string
	InstanceID int
}

// Main container for stats for serviced to consume
type MemoryUsageStats struct {
	HostID     string
	ServiceID  string
	InstanceID string
	Last       int64
	Max        int64
	Average    int64
}

// filterV2ResultsInstance will compare a result data's serviced ID and instance ID
// to a tag map of desired serviceIDs to instanceIDs. If the result data's service ID and
// instance ID are in the supplied mapping, then return True.
//
// TODO: This could be expensive on large data sets
func filterV2ResultsInstance(result V2ResultData, svcToInstances map[string][]string) bool {
	/*
		tags = serviceID -> [instanceIDs]
	*/
	servicedID := result.Tags["controlplane_service_id"]
	instances, ok := svcToInstances[servicedID]
	// curr result's service ID not in our tag list
	if !ok {
		return false
	}
	// see if result's instance ID is in the instance list for the service
	for _, instanceID := range instances {
		if instanceID == result.Tags["controlplane_instance_id"] {
			return true
		}
	}
	return false
}

func convertV2MemoryUsage(perfData map[string]*V2PerformanceData) []MemoryUsageStats {
	memStatsMap := make(map[string]*MemoryUsageStats) // serviceID.InstanceID
	for agg, perf := range perfData {
		for _, result := range perf.Series {
			key := result.Tags["controlplane_service_id"] + "." + result.Tags["controlplane_instance_id"]
			memStat, ok := memStatsMap[key]
			if !ok {
				memStat = &MemoryUsageStats{
					HostID:     result.Tags["controlplane_host_id"],
					ServiceID:  result.Tags["controlplane_service_id"],
					InstanceID: result.Tags["controlplane_instance_id"],
				}
				memStatsMap[key] = memStat
			}
			// fail if no datapoints in series for some reason
			if len(result.Datapoints) < 1 {
				continue
			}
			// fill in memStat
			val := int64(result.Datapoints[0].Value())
			switch agg {
			case "max":
				memStat.Max = val
			case "avg":
				memStat.Average = val
			case "last":
				memStat.Last = val
			}
		}
	}
	memStats := []MemoryUsageStats{}
	for _, memStat := range memStatsMap {
		memStats = append(memStats, *memStat)
	}
	return memStats
}

func convertMemoryUsage(data *PerformanceData) []MemoryUsageStats {
	mems := make([]MemoryUsageStats, len(data.Results))
	for i, result := range data.Results {
		mems[i] = MemoryUsageStats{}

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

		var last, sum, max, count float64
		for _, dp := range result.Datapoints {
			// lets skip NaN values
			if dp.Value.IsNaN {
				continue
			}

			if last = dp.Value.Value; last > max {
				max = last
			}
			sum += dp.Value.Value
			count++
		}
		mems[i].Last = int64(last)
		mems[i].Max = int64(max)
		mems[i].Average = int64(sum / count)
	}
	return mems
}

func (c *Client) GetHostMemoryStats(startDate time.Time, hostID string) (*MemoryUsageStats, error) {
	getter := func() ([]MemoryUsageStats, error) {
		glog.V(2).Infof("Requesting memory stats for host %s", hostID)
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

		return mems, nil
	}
	stats, err := cache.Get(hostID, getter)
	if err != nil {
		return nil, err
	}
	return &stats[0], nil
}

func (c *Client) GetServiceMemoryStats(startDate time.Time, serviceID string) (*MemoryUsageStats, error) {
	getter := func() ([]MemoryUsageStats, error) {
		glog.V(2).Infof("Requesting memory stats for service %s", serviceID)
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
			glog.V(2).Infof("Could not get performance data for service %s: %s", serviceID, err)
			return nil, err
		}

		mems := convertMemoryUsage(result)
		if len(mems) < 1 {
			err := fmt.Errorf("no data found")
			return nil, err
		}

		return mems, nil
	}
	stats, err := cache.Get(serviceID, getter)
	if err != nil {
		return nil, err
	}
	return &stats[0], nil
}

func (c *Client) GetInstanceMemoryStats(startDate time.Time, instances ...ServiceInstance) ([]MemoryUsageStats, error) {
	glog.V(2).Infof("Requesting memory stats for %d instances", len(instances))
	secsAgo := time.Now().Sub(startDate).Seconds()
	options := V2PerformanceOptions{
		Start: fmt.Sprintf("%ds-ago", int(secsAgo)),
		End:   "now",
	}

	// build a list of unique service IDs for the query
	serviceInstanceFilterMap := make(map[string][]string)
	servicesMap := make(map[string]struct{})
	serviceIdTags := []string{}
	for _, instance := range instances {
		if _, ok := servicesMap[instance.ServiceID]; !ok {
			servicesMap[instance.ServiceID] = struct{}{}
			serviceIdTags = append(serviceIdTags, instance.ServiceID)
		}
		// fill out filter map for later use
		tags, ok := serviceInstanceFilterMap[instance.ServiceID]
		if !ok {
			serviceInstanceFilterMap[instance.ServiceID] = []string{strconv.Itoa(instance.InstanceID)}
		} else {
			tags = append(tags, strconv.Itoa(instance.InstanceID))
			serviceInstanceFilterMap[instance.ServiceID] = tags
		}
	}

	query := V2MetricOptions{
		Metric: "cgroup.memory.totalrss",
		Tags: map[string][]string{
			"controlplane_service_id":  serviceIdTags,
			"controlplane_instance_id": []string{"*"},
		},
	}

	getter := func() ([]MemoryUsageStats, error) {
		// get max + avg
		perfDataMap := make(map[string]*V2PerformanceData)
		for _, agg := range []string{"max", "avg"} {
			query.Downsample = fmt.Sprintf("%ds-%s", int(secsAgo), agg)
			options.Metrics = []V2MetricOptions{query}
			options.Returnset = "exact"

			result, err := c.v2performanceQuery(options)
			if err != nil {
				glog.V(2).Infof("Could not get performance data for instances %+v: %s", instances, err)
				return nil, err
			}
			perfDataMap[agg] = result
		}

		// get curr
		query.Downsample = ""
		options.Metrics = []V2MetricOptions{query}
		options.Returnset = "last"
		result, err := c.v2performanceQuery(options)
		if err != nil {
			glog.V(2).Infof("Could not get performance data for instances %+v: %s", instances, err)
			return nil, err
		}
		perfDataMap["last"] = result

		// filter out our results by service ID + Instance ID
		for _, perfData := range perfDataMap {
			filteredSeries := []V2ResultData{}
			for _, result := range perfData.Series {
				if filterV2ResultsInstance(result, serviceInstanceFilterMap) {
					filteredSeries = append(filteredSeries, result)
				}
			}
			perfData.Series = filteredSeries
		}

		// normalize results to return
		return convertV2MemoryUsage(perfDataMap), nil
	}
	var keys []string
	for _, instance := range instances {
		keys = append(keys, fmt.Sprintf("%s.%d", instance.ServiceID, instance.InstanceID))
	}
	key := strings.Join(keys, "_")
	stats, err := cache.Get(key, getter)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
