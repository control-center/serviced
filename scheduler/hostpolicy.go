package scheduler

import (
	"errors"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
)

// serviceHostPolicy wraps a service and provides several policy
// implementations for choosing hosts on which to run instances of that
// service.
type serviceHostPolicy struct {
	svc *dao.Service
	hi  HostInfo
}

// ServiceHostPolicy returns a new serviceHostPolicy.
func ServiceHostPolicy(s *dao.Service, cp dao.ControlPlane) *serviceHostPolicy {
	return &serviceHostPolicy{s, &DAOHostInfo{cp}}
}

func (sp *serviceHostPolicy) SelectHost(hosts []*host.Host) (*host.Host, error) {
	switch sp.svc.HostPolicy {
	case dao.PREFER_SEPARATE:
		glog.V(2).Infof("Using PREFER_SEPARATE host policy")
		return sp.preferSeparateHosts(hosts)
	case dao.REQUIRE_SEPARATE:
		glog.V(2).Infof("Using REQUIRE_SEPARATE host policy")
		return sp.requireSeparateHosts(hosts)
	default:
		glog.V(2).Infof("Using LEAST_COMMITED host policy")
		return sp.leastCommittedHost(hosts)
	}
}

// leastCommittedHost chooses the host with the least RAM committed to running
// containers.
func (sp *serviceHostPolicy) leastCommittedHost(hosts []*host.Host) (*host.Host, error) {
	var (
		prioritized []*host.Host
		err         error
	)
	if prioritized, err = sp.hi.PrioritizeByMemory(hosts); err != nil {
		return nil, err
	}
	return prioritized[0], nil
}

// preferSeparateHosts chooses the least committed host that isn't already
// running an instance of the service. If all hosts are running an instance of
// the service already, it returns the least committed host.
func (sp *serviceHostPolicy) preferSeparateHosts(hosts []*host.Host) (*host.Host, error) {
	var (
		prioritized []*host.Host
		err         error
	)
	if prioritized, err = sp.hi.PrioritizeByMemory(hosts); err != nil {
		return nil, err
	}
	// First pass: find one that isn't running an instance of the service
	if h := sp.hi.FirstFreeHost(sp.svc, prioritized); h != nil {
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
func (sp *serviceHostPolicy) requireSeparateHosts(hosts []*host.Host) (*host.Host, error) {
	var (
		prioritized []*host.Host
		err         error
	)
	if prioritized, err = sp.hi.PrioritizeByMemory(hosts); err != nil {
		return nil, err
	}
	// First pass: find one that isn't running an instance of the service
	if h := sp.hi.FirstFreeHost(sp.svc, prioritized); h != nil {
		return h, nil
	}
	// No second pass
	return nil, errors.New("Unable to find a host to schedule")
}
