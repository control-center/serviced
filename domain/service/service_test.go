// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package service

import (
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

func (s *S) TestAddVirtualHost(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					VHosts:      nil,
				},
			},
		},
	}

	var err error
	if err = svc.AddVirtualHost("empty_server", "name"); err == nil {
		t.Errorf("Expected error adding vhost")
	}

	if err = svc.AddVirtualHost("server", "name"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	//no duplicate hosts can be added... hostnames are case-insensitive
	if err = svc.AddVirtualHost("server", "NAME"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	if len(svc.Endpoints[0].VHosts) != 1 && (svc.Endpoints[0].VHosts)[0] != "name" {
		t.Errorf("Virtualhost incorrect, %+v should contain name", svc.Endpoints[0].VHosts)
	}
}

func (s *S) TestRemoveVirtualHost(t *C) {
	svc := Service{
		Endpoints: []ServiceEndpoint{
			ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Purpose:     "export",
					Application: "server",
					VHosts:      []string{"name0", "name1"},
				},
			},
		},
	}

	var err error
	if err = svc.RemoveVirtualHost("server", "name0"); err != nil {
		t.Errorf("Unexpected error adding vhost: %v", err)
	}

	if len(svc.Endpoints[0].VHosts) != 1 && svc.Endpoints[0].VHosts[0] != "name1" {
		t.Errorf("Virtualhost incorrect, %+v should contain one host", svc.Endpoints[0].VHosts)
	}

	if err = svc.RemoveVirtualHost("server", "name0"); err == nil {
		t.Errorf("Expected error removing vhost")
	}
}
