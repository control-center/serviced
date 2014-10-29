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

//Visit called with current Service being visited
type Visit func(svc *Service) error

//GetChildServices returns a list of services that are children to the parentID, return empty list if none found
type GetChildServices func(parentID string) ([]Service, error)

//GetService return a service, return error if not found
type GetService func(serviceID string) (Service, error)

//FindChildService finds a child service with a given name, error if not found
type FindChildService func(parentID, childName string) (Service, error)

//Walk traverses the service hierarchy and calls the supplied Visit function on each service
func Walk(serviceID string, visitFn Visit, getService GetService, getChildren GetChildServices) error {
	// get the children
	subServices, err := getChildren(serviceID)
	if err != nil {
		return err
	}

	// walk the children
	for _, svc := range subServices {
		if err := Walk(svc.ID, visitFn, getService, getChildren); err != nil {
			return err
		}
	}

	// update the service
	svc, err := getService(serviceID)
	if err != nil {
		return err
	}

	return visitFn(&svc)
}
