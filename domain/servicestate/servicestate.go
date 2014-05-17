// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicestate

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/domain/service"

	"fmt"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/utils"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// An instantiation of a Service.
type ServiceState struct {
	Id          string
	ServiceID   string
	HostId      string
	DockerId    string
	PrivateIP   string
	Scheduled   time.Time
	Terminated  time.Time
	Started     time.Time
	PortMapping map[string][]domain.HostIpAndPort // protocol -> container port (internal) -> host port (external)
	Endpoints   []service.ServiceEndpoint
	HostIp      string
	InstanceID  int
}

//A new service instance (ServiceState)
func BuildFromService(service *service.Service, hostId string) (serviceState *ServiceState, err error) {
	serviceState = &ServiceState{}
	serviceState.Id, err = utils.NewUUID36()
	if err == nil {
		serviceState.ServiceID = service.Id
		serviceState.HostId = hostId
		serviceState.Scheduled = time.Now()
		serviceState.Endpoints = service.Endpoints
	}
	return serviceState, err
}

// Retrieve service container port info.
func (ss *ServiceState) GetHostEndpointInfo(applicationRegex *regexp.Regexp) (hostPort, containerPort uint16, protocol string, match bool) {
	for _, ep := range ss.Endpoints {
		if ep.Purpose == "export" {
			if applicationRegex.MatchString(ep.Application) {
				portS := fmt.Sprintf("%d/%s", ep.PortNumber, strings.ToLower(ep.Protocol))

				external := ss.PortMapping[portS]
				if len(external) == 0 {
					glog.Warningf("Found match for %s:%s, but no portmapping is available", applicationRegex, portS)
					break
				}

				extPort, err := strconv.ParseUint(external[0].HostPort, 10, 16)
				if err != nil {
					glog.Errorf("Portmap parsing failed for %s:%s %v", applicationRegex, portS, err)
					break
				}
				return uint16(extPort), ep.PortNumber, ep.Protocol, true
			}
		}
	}

	return 0, 0, "", false
}
