package scheduler

import (
	"container/heap"
	"errors"
	"sync"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
)

// HostInfo provides methods for getting host information from the dao or
// otherwise. It's a separate interface for the sake of testing.
type HostInfo interface {
	AvailableRAM(*host.Host, chan *hostitem, <-chan bool)
	PrioritizeByMemory([]*host.Host) ([]*host.Host, error)
	FirstFreeHost(*dao.Service, []*host.Host) *host.Host
}

type DAOHostInfo struct {
	dao dao.ControlPlane
}

func (hi *DAOHostInfo) FirstFreeHost(svc *dao.Service, hosts []*host.Host) *host.Host {
	for _, host := range hosts {
		rss := []*dao.RunningService{}
		if err := hi.dao.GetRunningServicesForHost(host.ID, &rss); err != nil {
			glog.Errorf("cannot retrieve running services for host: %s (%v)", host.ID, err)
		}
		for _, rs := range rss {
			if rs.ServiceId != svc.Id {
				return host
			}
		}
	}
	return nil
}

// AvailableRAM computes the amount of RAM available on a given host by
// subtracting the sum of the RAM commitments of each of its running services
// from its total memory.
func (hi *DAOHostInfo) AvailableRAM(host *host.Host, result chan *hostitem, done <-chan bool) {
	rss := []*dao.RunningService{}
	if err := hi.dao.GetRunningServicesForHost(host.ID, &rss); err != nil {
		glog.Errorf("cannot retrieve running services for host: %s (%v)", host.ID, err)
		return // this host won't be scheduled
	}

	var cr uint64

	for i := range rss {
		s := dao.Service{}
		if err := hi.dao.GetService(rss[i].ServiceId, &s); err != nil {
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
