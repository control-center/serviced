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
	"github.com/zenoss/glog"
)

// ServiceUse will use a new image for a given service - this will pull the image and tag it
func (c *Client) ServiceUse(serviceID string, imageID string, registry string, replaceImgs []string, noOp bool) (string, error) {
	svcUseRequest := &ServiceUseRequest{ServiceID: serviceID, ImageID: imageID, ReplaceImgs: replaceImgs, Registry: registry, NoOp: noOp}
	result := ""
	glog.Infof("Pulling %s, tagging to latest, and pushing to registry %s - this may take a while", imageID, registry)
	err := c.call("ServiceUse", svcUseRequest, &result)
	if err != nil {
		return "", err
	}
	return result, nil
}

// WaitService will wait for the specified services to reach the specified
// state, within the given timeout
func (c *Client) WaitService(serviceIDs []string, state service.DesiredState, timeout time.Duration, recursive bool) error {
	waitSvcRequest := &WaitServiceRequest{
		ServiceIDs: serviceIDs,
		State:      state,
		Timeout:    timeout,
		Recursive:  recursive,
	}
	throwaway := ""
	err := c.call("WaitService", waitSvcRequest, &throwaway)
	return err
}

// GetAllServiceDetails will return a list of all ServiceDetails
func (c *Client) GetAllServiceDetails(since time.Duration) ([]service.ServiceDetails, error) {
	svcs := []service.ServiceDetails{}
	err := c.call("GetAllServiceDetails", since, &svcs)
	return svcs, err
}

// GetServiceDetailsByTenantID will return a list of ServiceDetails for the specified tenant ID
func (c *Client) GetServiceDetailsByTenantID(tenantID string) ([]service.ServiceDetails, error) {
	svcs := []service.ServiceDetails{}
	err := c.call("GetServiceDetailsByTenantID", tenantID, &svcs)
	return svcs, err
}

// GetServiceDetails will return a ServiceDetails for the specified service
func (c *Client) GetServiceDetails(serviceID string) (*service.ServiceDetails, error) {
	svc := &service.ServiceDetails{}
	err := c.call("GetServiceDetails", serviceID, svc)
	return svc, err
}

// GetService returns a service with a particular service id.
func (c *Client) GetService(serviceID string) (*service.Service, error) {
	svc := &service.Service{}
	err := c.call("GetService", serviceID, svc)
	return svc, err
}

// GetEvaluatedService returns a service where an evaluation has been executed against all templated properties.
func (c *Client) GetEvaluatedService(serviceID string, instanceID int) (*service.Service, string, error) {
	request := EvaluateServiceRequest{
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}
	response := EvaluateServiceResponse{}
	err := c.call("GetEvaluatedService", request, &response)
	if err != nil {
		return nil, "", err
	}
	return &response.Service, response.TenantID, err
}

// GetTenantID returns the ID of the service's tenant (i.e. the root service's ID)
func (c *Client) GetTenantID(serviceID string) (string, error) {
	tenantID := ""
	err := c.call("GetTenantID", serviceID, &tenantID)
	return tenantID, err
}

// ResolveServicePath resolves a service path (e.g., "infrastructure/mariadb") to zero or more ServiceDetails.
func (c *Client) ResolveServicePath(path string) ([]service.ServiceDetails, error) {
	svcs := []service.ServiceDetails{}
	err := c.call("ResolveServicePath", path, &svcs)
	return svcs, err
}

// ClearEmergency clears the EmergencyShutdown flag on a service and all child services
// it returns the number of affected services
func (c *Client) ClearEmergency(serviceID string) (int, error) {
	affected := 0
	err := c.call("ClearEmergency", serviceID, &affected)
	return affected, err
}
