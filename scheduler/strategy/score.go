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

type scoredHost struct {
	Host  Host
	Score int
}

type scoredHostList []*scoredHost

func (l scoredHostList) Len() int {
	return len(l)
}

func (l scoredHostList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l scoredHostList) Less(i, j int) bool {
	return l[i].Score < l[j].Score
}

func ScoreHosts(service ServiceConfig, hosts []Host) ([]Host, error) {

	enough_free := scoredHostList{}
	not_enough_free := scoredHostList{}

	for _, host := range hosts {

		totalMem := host.TotalMemory()
		totalCpu := host.TotalCores()

		var (
			usedCpu  int
			usedMem  uint64
			cpuScore int = 100
			memScore int = 100
		)

		// Calculate used resources for the host
		for _, svc := range host.RunningServices() {
			usedCpu += svc.RequestedCores()
			usedMem += svc.RequestedMemory()
		}

		// Calculate CPU score as a percentage of used cores on the host with this service deployed
		if service.RequestedCores() > 0 {
			cpuScore = (usedCpu + service.RequestedCores()) * 100 / totalCpu
		}

		// Calculate memory score as a percentage of used memory on the host with this service deployed
		if service.RequestedMemory() > 0 {
			memScore = int((usedMem + service.RequestedMemory()) * 100 / totalMem)
		}

		if cpuScore <= 100 && memScore <= 100 {
			shost := &scoredHost{Host: host, Score: cpuScore + memScore}
			enough_free = append(enough_free, shost)
		} else {
			shost := &scoredHost{Host: host, Score: memScore}
			not_enough_free = append(not_enough_free, shost)
		}

	}

	sort.Sort(enough_free)
	sort.Sort(not_enough_free)

	var sorted []Host

	for _, s := range enough_free {
		sorted = append(sorted, s.Host)
	}
	for _, s := range not_enough_free {
		sorted = append(sorted, s.Host)
	}
	return sorted, nil
}
