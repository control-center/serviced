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

import (
	"sort"

	"github.com/zenoss/glog"
)

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

	glog.V(2).Infof("Scoring %d hosts for service %s", len(hosts), service.GetServiceID())
	glog.V(2).Infof("Service %s is requesting %s memory and %s percent CPU", service.GetServiceID(), service.RequestedMemoryBytes(), service.RequestedCorePercent())

	undersubscribed := scoredHostList{}
	oversubscribed := scoredHostList{}

	for _, host := range hosts {

		scoredHost := &ScoredHost{Host: host}

		totalMem := host.TotalMemory()
		totalCpu := host.TotalCores()

		glog.V(2).Infof("Host %s has %s memory and %s cores", host.HostID(), totalMem, totalCpu)

		var (
			usedCpu  int
			usedMem  uint64
			cpuScore int
			memScore int
		)

		// Calculate used resources for the host
		for _, svc := range host.RunningServices() {
			glog.V(2).Infof("Host %s is running service %s (%s/%s)", host.HostID(), svc.GetServiceID(), svc.RequestedCorePercent(), svc.RequestedMemoryBytes())
			usedCpu += svc.RequestedCorePercent()
			usedMem += svc.RequestedMemoryBytes()
			// Increment a counter of number of instances, for later strategies to use
			if svc.GetServiceID() == service.GetServiceID() {
				scoredHost.NumInstances += 1
			}
		}

		// Calculate CPU score as a percentage of used cores on the host with this service deployed
		if totalCpu > 0 {
			if service.RequestedCorePercent() > 0 {
				cpuScore = (usedCpu + service.RequestedCorePercent()) / totalCpu
			}
		} else {
			cpuScore = 100
		}

		// Calculate memory score as a percentage of used memory on the host with this service deployed
		if totalMem > 0 {
			if service.RequestedMemoryBytes() > 0 {
				memScore = int((usedMem + service.RequestedMemoryBytes()) * 100 / totalMem)
			}
		} else {
			memScore = 100
		}

		glog.V(2).Infof("Host %s CPU score: %s, memory score: %s", host.HostID(), cpuScore, memScore)
		if cpuScore <= 100 && memScore <= 100 {
			glog.V(2).Infof("Host %s can run service %s", host.HostID(), service.GetServiceID())
			scoredHost.Score = cpuScore + memScore
			undersubscribed = append(undersubscribed, scoredHost)
		} else {
			glog.V(2).Infof("Host %s would be oversubscribed with service %s", host.HostID(), service.GetServiceID())
			scoredHost.Score = memScore
			oversubscribed = append(oversubscribed, scoredHost)
		}
	}

	sort.Sort(undersubscribed)
	sort.Sort(oversubscribed)

	return undersubscribed, oversubscribed
}
