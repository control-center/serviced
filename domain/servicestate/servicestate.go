// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicestate

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/zenoss/glog"

	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/utils"
)

// Function map for evaluating PortTemplate fields
var funcmap = template.FuncMap{
	"plus": func(a, b int) int {
		return a + b
	},
}

// An instantiation of a Service.
type ServiceState struct {
	ID          string
	ServiceID   string
	HostID      string
	DockerID    string
	PrivateIP   string
	Scheduled   time.Time
	Terminated  time.Time
	Started     time.Time
	PortMapping map[string][]domain.HostIPAndPort // protocol -> container port (internal) -> host port (external)
	// remove list?  PortMapping:map[6379/tcp:[{HostIP:0.0.0.0 HostPort:49195}]]
	//  i.e. redis:  PortMapping:map[6379/tcp: {HostIP:0.0.0.0 HostPort:49195} ]
	Endpoints  []service.ServiceEndpoint
	HostIP     string
	InstanceID int
}

func (ss *ServiceState) evalPortTemplate(portTemplate string) (int, error) {
	t := template.Must(template.New("PortTemplate").Funcs(funcmap).Parse(portTemplate))
	b := bytes.Buffer{}
	if err := t.Execute(&b, ss); err != nil {
		return 0, err
	}
	i, err := strconv.Atoi(b.String())
	if err != nil {
		return 0, err
	}
	return i, nil
}

//A new service instance (ServiceState)
func BuildFromService(service *service.Service, hostId string) (serviceState *ServiceState, err error) {
	serviceState = &ServiceState{}
	serviceState.ID, err = utils.NewUUID36()
	if err == nil {
		serviceState.ServiceID = service.ID
		serviceState.HostID = hostId
		serviceState.Scheduled = time.Now()
		serviceState.Endpoints = service.Endpoints
		for j, ep := range serviceState.Endpoints {
			if ep.PortTemplate != "" {
				port, err := serviceState.evalPortTemplate(ep.PortTemplate)
				if err != nil {
					return nil, err
				}
				ep.PortNumber = uint16(port)
				serviceState.Endpoints[j] = ep
			}
		}
	}
	return serviceState, err
}

// Retrieve service container port info.
func (ss *ServiceState) GetHostEndpointInfo(applicationRegex *regexp.Regexp) (hostPort, containerPort uint16, protocol string, match bool) {
	for _, ep := range ss.Endpoints {

		if ep.Purpose == "export" {
			if applicationRegex.MatchString(ep.Application) {
				if ep.PortTemplate != "" {
					port, err := ss.evalPortTemplate(ep.PortTemplate)
					if err != nil {
						glog.Errorf("%+v", err)
						break
					}
					ep.PortNumber = uint16(port)
				}
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
