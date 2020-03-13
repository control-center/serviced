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
)

var (
	cache = NewMemoryUsageCache(time.Minute)
)

// ServiceInstance identifies an instance of a running service.
type ServiceInstance struct {
	ServiceID  string
	InstanceID int
}

// MemoryUsageStats main container for stats for serviced to consume
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
func filterV2ResultsInstance(result v2ResultData, svcToInstances map[string][]string) bool {
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

func convertV2MemoryUsage(perfData map[string]*v2PerformanceData) []MemoryUsageStats {
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

// GetHostMemoryStats returns host memory usage stats
func (c *clientImpl) GetHostMemoryStats(startDate time.Time, hostID string) (*MemoryUsageStats, error) {
	logger := log.WithField("hostid", hostID)
	getter := func() ([]MemoryUsageStats, error) {
		logger.Debug("Requesting memory stats for host")
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

// GetServiceMemoryStats returns service memory usage stats
func (c *clientImpl) GetServiceMemoryStats(startDate time.Time, serviceID string) (*MemoryUsageStats, error) {
	logger := log.WithField("serviceid", serviceID)
	getter := func() ([]MemoryUsageStats, error) {
		logger.Debug("Requesting memory stats for service")
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

// GetInstanceMemoryStats returns service instance memory usage stats
func (c *clientImpl) GetInstanceMemoryStats(startDate time.Time, instances ...ServiceInstance) ([]MemoryUsageStats, error) {
	logger := log.WithField("instancecount", len(instances))
	logger.Debug("Requesting memory stats service instances")
	secsAgo := time.Now().Sub(startDate).Seconds()
	options := v2PerformanceOptions{
		Start: fmt.Sprintf("%ds-ago", int(secsAgo)),
		End:   "now",
	}

	// build a list of unique service IDs for the query
	serviceInstanceFilterMap := make(map[string][]string)
	servicesMap := make(map[string]struct{})
	serviceIDTags := []string{}
	for _, instance := range instances {
		if _, ok := servicesMap[instance.ServiceID]; !ok {
			servicesMap[instance.ServiceID] = struct{}{}
			serviceIDTags = append(serviceIDTags, instance.ServiceID)
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

	query := v2MetricOptions{
		Metric: "cgroup.memory.totalrss",
		Tags: map[string][]string{
			"controlplane_service_id":  serviceIDTags,
			"controlplane_instance_id": []string{"*"},
		},
	}

	getter := func() ([]MemoryUsageStats, error) {
		// get max + avg
		perfDataMap := make(map[string]*v2PerformanceData)
		for _, agg := range []string{"max", "avg"} {
			query.Downsample = fmt.Sprintf("%ds-%s", int(secsAgo), agg)
			options.Metrics = []v2MetricOptions{query}
			options.Returnset = "exact"

			result, err := c.v2performanceQuery(options)
			if err != nil {
				return nil, err
			}
			perfDataMap[agg] = result
		}

		// get curr
		query.Downsample = ""
		options.Metrics = []v2MetricOptions{query}
		options.Returnset = "last"
		options.Start = "10m-ago" //Reduce the time frame to search for last value to reduce memory usage in CentralQuery
		result, err := c.v2performanceQuery(options)
		if err != nil {
			return nil, err
		}
		perfDataMap["last"] = result

		// filter out our results by service ID + Instance ID
		for _, perfData := range perfDataMap {
			filteredSeries := []v2ResultData{}
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
