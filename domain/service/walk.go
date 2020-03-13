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

package service

// VisitFn called with current Service being visited
type VisitFn func(svc *Service) error

// GetChildServicesFn returns a list of services that are children to the parentID, return empty list if none found
type GetChildServicesFn func(parentID string) ([]Service, error)

// GetServiceFn return a service, return error if not found
type GetServiceFn func(serviceID string) (Service, error)

// FindChildServiceFn finds a child service with a given name, error if not found
type FindChildServiceFn func(parentID, childName string) (Service, error)

//Walk traverses the service hierarchy and calls the supplied Visit function on each service
func Walk(serviceID string, visit VisitFn, getService GetServiceFn, getChildren GetChildServicesFn) error {
	// get the children
	subServices, err := getChildren(serviceID)
	if err != nil {
		return err
	}

	// walk the children
	for _, svc := range subServices {
		if err := Walk(svc.ID, visit, getService, getChildren); err != nil {
			return err
		}
	}

	// update the service
	svc, err := getService(serviceID)
	if err != nil {
		return err
	}

	return visit(&svc)
}
