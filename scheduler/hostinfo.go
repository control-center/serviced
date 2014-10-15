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

package scheduler

import (
	"container/heap"
	"errors"
	"sync"

	"github.com/zenoss/glog"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
)

// HostInfo provides methods for getting host information from the dao or
// otherwise. It's a separate interface for the sake of testing.
type HostInfo interface {
	AvailableRAM(*host.Host, chan *hostitem, <-chan bool)
	PrioritizeByMemory([]*host.Host) ([]*host.Host, error)
	ServicesOnHost(*host.Host) []dao.RunningService
}

type DAOHostInfo struct {
	dao dao.ControlPlane
}

func (hi *DAOHostInfo) ServicesOnHost(h *host.Host) []dao.RunningService {
	rss := []dao.RunningService{}
	if err := hi.dao.GetRunningServicesForHost(h.ID, &rss); err != nil {
		glog.Errorf("cannot retrieve running services for host: %s (%v)", h.ID, err)
	}
	return rss
}

// AvailableRAM computes the amount of RAM available on a given host by
// subtracting the sum of the RAM commitments of each of its running services
// from its total memory.
func (hi *DAOHostInfo) AvailableRAM(host *host.Host, result chan *hostitem, done <-chan bool) {
	rss := []dao.RunningService{}
	if err := hi.dao.GetRunningServicesForHost(host.ID, &rss); err != nil {
		glog.Errorf("cannot retrieve running services for host: %s (%v)", host.ID, err)
		return // this host won't be scheduled
	}

	var cr uint64

	for i := range rss {
		s := service.Service{}
		if err := hi.dao.GetService(rss[i].ServiceID, &s); err != nil {
			glog.Errorf("cannot retrieve service information for running service (%v)", err)
			return // this host won't be scheduled
		}

		cr += s.RAMCommitment
	}

	result <- &hostitem{host, host.Memory - cr, -1}
}

func (hi *DAOHostInfo) PrioritizeByMemory(hosts []*host.Host) ([]*host.Host, error) {
	var wg sync.WaitGroup

	result := make([]*host.Host, 0)
	done := make(chan bool)
	defer close(done)

	hic := make(chan *hostitem)

	// fan-out available RAM computation for each host
	for _, h := range hosts {
		wg.Add(1)
		go func(host *host.Host) {
			hi.AvailableRAM(host, hic, done)
			wg.Done()
		}(h)
	}

	// close the hostitem channel when all the calculation is finished
	go func() {
		wg.Wait()
		close(hic)
	}()

	pq := &PriorityQueue{}
	heap.Init(pq)

	// fan-in all the available RAM computations
	for hi := range hic {
		heap.Push(pq, hi)
	}

	if pq.Len() < 1 {
		return nil, errors.New("Unable to find a host to schedule")
	}

	for pq.Len() > 0 {
		result = append(result, heap.Pop(pq).(*hostitem).host)
	}
	return result, nil
}
