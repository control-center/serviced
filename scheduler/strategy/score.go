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

package strategy

import "sort"

type ScoredHost struct {
	Host         Host
	Score        int
	NumInstances int
}

type scoredHostList []*ScoredHost

func (l scoredHostList) Len() int {
	return len(l)
}

func (l scoredHostList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l scoredHostList) Less(i, j int) bool {
	return l[i].Score < l[j].Score
}

// ScoreHosts returns two arrays of hosts. The first lists hosts that have
// enough resources to handle the service, sorted in order of combined free
// resources. The second lists hosts that do not have enough resources to
// handle the service, sorted in order of percentage memory used were the
// service deployed to the host.
func ScoreHosts(service ServiceConfig, hosts []Host) ([]*ScoredHost, []*ScoredHost) {

	undersubscribed := scoredHostList{}
	oversubscribed := scoredHostList{}

	for _, host := range hosts {

		scoredHost := &ScoredHost{Host: host}

		totalMem := host.TotalMemory()
		totalCpu := host.TotalCores()

		var (
			usedCpu  int
			usedMem  uint64
			cpuScore int
			memScore int
		)

		// Calculate used resources for the host
		for _, svc := range host.RunningServices() {
			usedCpu += svc.RequestedCorePercent()
			usedMem += svc.RequestedMemory()
			// Increment a counter of number of instances, for later strategies to use
			if svc.GetServiceID() == service.GetServiceID() {
				scoredHost.NumInstances += 1
			}
		}

		// Calculate CPU score as a percentage of used cores on the host with this service deployed
		if service.RequestedCorePercent() > 0 {
			cpuScore = (usedCpu + service.RequestedCorePercent()) * 100 / totalCpu
		}

		// Calculate memory score as a percentage of used memory on the host with this service deployed
		if service.RequestedMemory() > 0 {
			memScore = int((usedMem + service.RequestedMemory()) * 100 / totalMem)
		}

		if cpuScore <= 100 && memScore <= 100 {
			scoredHost.Score = cpuScore + memScore
			undersubscribed = append(undersubscribed, scoredHost)
		} else {
			scoredHost.Score = memScore
			oversubscribed = append(oversubscribed, scoredHost)

		}
	}

	sort.Sort(undersubscribed)
	sort.Sort(oversubscribed)

	return undersubscribed, oversubscribed
}
