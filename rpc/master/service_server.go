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

package master

import (
	"time"

	"github.com/control-center/serviced/domain/service"
)

type ServiceUseRequest struct {
	ServiceID   string
	ImageID     string
	ReplaceImgs []string
	Registry    string
	NoOp        bool
}

type WaitServiceRequest struct {
	ServiceIDs []string
	State      service.DesiredState
	Timeout    time.Duration
	Recursive  bool
}

type EvaluateServiceRequest struct {
	ServiceID  string
	InstanceID int
}

// Use a new image for a given service - this will pull the image and tag it
func (s *Server) ServiceUse(request *ServiceUseRequest, response *string) error {
	if err := s.f.ServiceUse(s.context(), request.ServiceID, request.ImageID, request.Registry, request.ReplaceImgs, request.NoOp); err != nil {
		return err
	}
	*response = ""
	return nil
}

// Wait on specified services to be in the given state
func (s *Server) WaitService(request *WaitServiceRequest, throwaway *string) error {
	err := s.f.WaitService(s.context(), request.State, request.Timeout, request.Recursive, request.ServiceIDs...)
	return err
}

// Get a specific service
func (s *Server) GetService(serviceID string, svc *service.Service) error {
	sv, err := s.f.GetService(s.context(), serviceID)
	if err != nil {
		return err
	}
	*svc = *sv
	return nil
}

// GetEvaluatedService returns a service where an evaluation has been executed against all templated properties.
func (s *Server) GetEvaluatedService(request EvaluateServiceRequest, svc *service.Service) error {
	result, err := s.f.GetEvaluatedService(s.context(), request.ServiceID, request.InstanceID)
	if err != nil {
		return err
	}
	*svc = *result
	return nil
}

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (s *Server) GetTenantID(serviceID string, tenantId *string) error {
	result, err := s.f.GetTenantID(s.context(), serviceID)
	if err != nil {
		return err
	}
	*tenantId = result
	return nil
}
