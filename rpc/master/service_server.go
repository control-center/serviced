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

type EvaluateServiceResponse struct {
	Service  service.Service
	TenantID string
}

type ServiceDetailsByTenantIDRequest struct {
	TenantID string
	Since    time.Duration
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

// GetAllServiceDetails will return a list of all ServiceDetails
func (s *Server) GetAllServiceDetails(since time.Duration, response *[]service.ServiceDetails) error {
	svcs, err := s.f.GetAllServiceDetails(s.context(), since)
	if err != nil {
		return err
	}
	*response = svcs
	return nil
}

// GetServiceDetails will return a ServiceDetails for the specified service
func (s *Server) GetServiceDetails(serviceID string, response *service.ServiceDetails) error {
	svc, err := s.f.GetServiceDetails(s.context(), serviceID)
	if err != nil {
		return err
	}
	*response = *svc
	return nil
}

// GetServiceDetailsByTenantID will return a list of ServiceDetails for the specified tenant ID
func (s *Server) GetServiceDetailsByTenantID(tenantID string, response *[]service.ServiceDetails) error {
	svcs, err := s.f.GetServiceDetailsByTenantID(s.context(), tenantID)
	if err != nil {
		return err
	}
	*response = svcs
	return nil
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
func (s *Server) GetEvaluatedService(request EvaluateServiceRequest, response *EvaluateServiceResponse) error {
	svc, err := s.f.GetEvaluatedService(s.context(), request.ServiceID, request.InstanceID)
	if err != nil {
		return err
	}

	tenantID, err := s.f.GetTenantID(s.context(), request.ServiceID)
	if err != nil {
		return err
	}
	response.Service = *svc
	response.TenantID = tenantID
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

// ResolveServicePath resolves a service path (e.g., "infrastructure/mariadb") to zero or more ServiceDetails.
func (s *Server) ResolveServicePath(path string, response *[]service.ServiceDetails) error {
	svcs, err := s.f.ResolveServicePath(s.context(), path)
	if err != nil {
		return err
	}
	*response = svcs
	return nil
}

// ClearEmergency clears the EmergencyShutdown flag on a service and all child services
// it returns the number of affected services
func (s *Server) ClearEmergency(serviceID string, count *int) error {
	c, err := s.f.ClearEmergencyStopFlag(s.context(), serviceID)
	if err != nil {
		return err
	}
	*count = c
	return nil
}
