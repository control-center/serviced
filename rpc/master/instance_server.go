// Copyright 2016 The Serviced Authors.
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

package master

import (
	"time"

	"github.com/control-center/serviced/domain/service"
)

// GetServiceInstances returns all instances of a service
func (s *Server) GetServiceInstances(serviceID string, res *[]service.Instance) (err error) {
	insts, err := s.f.GetServiceInstances(s.context(), time.Now().Add(-time.Hour), serviceID)
	if err != nil {
		return
	}
	*res = insts
	return
}

type ServiceInstanceRequest struct {
	ServiceID  string
	InstanceID int
}

type InstanceStateRequest struct {
	ServiceID  string
	InstanceID int
	CState     CurrentState
}

type SendStatesRequest struct {
	States map[string]CurrentState
}

func (s *Server) SetInstanceState(req InstanceStateRequest, unused *string) (err error) {
	c.f.SetInstanceCurrentState(req.ServiceID, req.InstanceID, req.CState)
	return nil
}

func (s *Server) BulkUpdate(req SendStatesRequest, unused *string) (err error) {
	c.f.BulkUpdate(req.States)
	return nil
}

// StopServiceInstance stops a single service instance
func (s *Server) StopServiceInstance(req ServiceInstanceRequest, unused *string) (err error) {
	// set the state change
	err = s.f.StopServiceInstance(s.context(), req.ServiceID, req.InstanceID)
	return
}

// LocateServiceInstance locates a single service instance
func (s *Server) LocateServiceInstance(req ServiceInstanceRequest, res *service.LocationInstance) (err error) {
	location, err := s.f.LocateServiceInstance(s.context(), req.ServiceID, req.InstanceID)
	if err != nil {
		return
	}
	*res = *location
	return
}

type DockerActionRequest struct {
	ServiceID  string
	InstanceID int
	Action     string
	Args       []string
}

// SendDockerAction submits an action to a docker container
func (s *Server) SendDockerAction(req DockerActionRequest, unused *string) (err error) {
	err = s.f.SendDockerAction(s.context(), req.ServiceID, req.InstanceID, req.Action, req.Args)
	return
}
