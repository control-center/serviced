// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

//Visit called with current Service being visited
type Visit func(svc *Service) error

//GetChildServices returns a list of services that are children to the parentID, return empty list if none found
type GetChildServices func(parentID string) ([]*Service, error)

//GetService return a service, return error if not found
type GetService func(serviceID string) (Service, error)

//Walk traverses the service hierarchy and calls the supplied Visit function on each service
func Walk(serviceID string, visitFn Visit, getService GetService, getChildren GetChildServices) error {
	//get the original service
	svc, err := getService(serviceID)
	if err != nil {
		return err
	}

	// do what you requested to do while visiting this node
	err = visitFn(&svc)
	if err != nil {
		return err
	}

	subServices, err := getChildren(serviceID)
	if err != nil {
		return err
	}
	for _, svc := range subServices {
		err = Walk(svc.ID, visitFn, getService, getChildren)
		if err != nil {
			return err
		}
	}

	return nil
}
