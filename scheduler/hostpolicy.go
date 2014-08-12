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
	"errors"

	"github.com/zenoss/glog"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
)

// ServiceHostPolicy wraps a service and provides several policy
// implementations for choosing hosts on which to run instances of that
// service.
type ServiceHostPolicy struct {
	svc   *service.Service
	hinfo HostInfo
}

// ServiceHostPolicy returns a new ServiceHostPolicy.
func NewServiceHostPolicy(s *service.Service, cp dao.ControlPlane) *ServiceHostPolicy {
	return &ServiceHostPolicy{s, &DAOHostInfo{cp}}
}

func (sp *ServiceHostPolicy) SelectHost(hosts []*host.Host) (*host.Host, error) {
	switch sp.svc.HostPolicy {
	case servicedefinition.PreferSeparate:
		glog.V(2).Infof("Using PREFER_SEPARATE host policy")
		return sp.preferSeparateHosts(hosts)
	case servicedefinition.RequireSeparate:
		glog.V(2).Infof("Using REQUIRE_SEPARATE host policy")
		return sp.requireSeparateHosts(hosts)
	default:
		glog.V(2).Infof("Using LEAST_COMMITTED host policy")
		return sp.leastCommittedHost(hosts)
	}
}

func (sp *ServiceHostPolicy) firstFreeHost(svc *service.Service, hosts []*host.Host) *host.Host {
hosts:
	for _, h := range hosts {
		rss := sp.hinfo.ServicesOnHost(h)
		for _, rs := range rss {
			if rs.ServiceID == svc.ID {
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
