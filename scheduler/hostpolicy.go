package scheduler

import (
	"errors"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
)

// ServiceHostPolicy wraps a service and provides several policy
// implementations for choosing hosts on which to run instances of that
// service.
type ServiceHostPolicy struct {
	svc   *dao.Service
	hinfo HostInfo
}

// ServiceHostPolicy returns a new ServiceHostPolicy.
func NewServiceHostPolicy(s *dao.Service, cp dao.ControlPlane) *ServiceHostPolicy {
	return &ServiceHostPolicy{s, &DAOHostInfo{cp}}
}

func (sp *ServiceHostPolicy) SelectHost(hosts []*host.Host) (*host.Host, error) {
	switch sp.svc.HostPolicy {
	case dao.PREFER_SEPARATE:
		glog.V(2).Infof("Using PREFER_SEPARATE host policy")
		return sp.preferSeparateHosts(hosts)
	case dao.REQUIRE_SEPARATE:
		glog.V(2).Infof("Using REQUIRE_SEPARATE host policy")
		return sp.requireSeparateHosts(hosts)
	default:
		glog.V(2).Infof("Using LEAST_COMMITTED host policy")
		return sp.leastCommittedHost(hosts)
	}
}

func (sp *ServiceHostPolicy) firstFreeHost(svc *dao.Service, hosts []*host.Host) *host.Host {
hosts:
	for _, h := range hosts {
		rss := sp.hinfo.ServicesOnHost(h)
		for _, rs := range rss {
			if rs.ServiceId == svc.Id {
				// This host already has an instance of this service. Move on.
				continue hosts
			}
		}
		return h
	}
	return nil
}

// leastCommittedHost chooses the host with the least RAM committed to running
// containers.
func (sp *ServiceHostPolicy) leastCommittedHost(hosts []*host.Host) (*host.Host, error) {
	var (
		prioritized []*host.Host
		err         error
	)
	if prioritized, err = sp.hinfo.PrioritizeByMemory(hosts); err != nil {
		return nil, err
	}
	return prioritized[0], nil
}

// preferSeparateHosts chooses the least committed host that isn't already
// running an instance of the service. If all hosts are running an instance of
// the service already, it returns the least committed host.
func (sp *ServiceHostPolicy) preferSeparateHosts(hosts []*host.Host) (*host.Host, error) {
	var (
		prioritized []*host.Host
		err         error
	)
	if prioritized, err = sp.hinfo.PrioritizeByMemory(hosts); err != nil {
		return nil, err
	}
	// First pass: find one that isn't running an instance of the service
	if h := sp.firstFreeHost(sp.svc, prioritized); h != nil {
		return h, nil
	}
	// Second pass: just find an available host
	for _, h := range prioritized {
		return h, nil
	}
	return nil, errors.New("Unable to find a host to schedule")
}

// requireSeparateHosts chooses the least committed host that isn't already
// running an instance of the service. If all hosts are running an instance of
// the service already, it returns an error.
func (sp *ServiceHostPolicy) requireSeparateHosts(hosts []*host.Host) (*host.Host, error) {
	var (
		prioritized []*host.Host
		err         error
	)
	if prioritized, err = sp.hinfo.PrioritizeByMemory(hosts); err != nil {
		return nil, err
	}
	// First pass: find one that isn't running an instance of the service
	if h := sp.firstFreeHost(sp.svc, prioritized); h != nil {
		return h, nil
	}
	// No second pass
	return nil, errors.New("Unable to find a host to schedule")
}
