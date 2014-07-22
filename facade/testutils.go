// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	gocheck "gopkg.in/check.v1"
)

//FacadeTest used for running tests where a facade type is needed.
type FacadeTest struct {
	elastic.ElasticTest
	CTX    datastore.Context
	Facade *Facade
}

//SetUpSuite sets up test suite
func (ft *FacadeTest) SetUpSuite(c *gocheck.C) {
	//set up index and mappings before setting up elastic
	ft.Index = "controlplane"
	if ft.Mappings == nil {
		ft.Mappings = make([]elastic.Mapping, 0)
	}
	ft.Mappings = append(ft.Mappings, host.MAPPING)
	ft.Mappings = append(ft.Mappings, pool.MAPPING)
	ft.Mappings = append(ft.Mappings, service.MAPPING)
	ft.Mappings = append(ft.Mappings, servicetemplate.MAPPING)
	ft.Mappings = append(ft.Mappings, addressassignment.MAPPING)
	ft.Mappings = append(ft.Mappings, serviceconfigfile.MAPPING)
	ft.Mappings = append(ft.Mappings, user.MAPPING)

	ft.ElasticTest.SetUpSuite(c)
	datastore.Register(ft.Driver())
	ft.CTX = datastore.Get()

	ft.Facade = New("localhost:5000")

	//mock out ZK calls to no ops
	zkAPI = func(f *Facade) zkfuncs { return &zkMock{} }
}

type zkMock struct {
}

func (z *zkMock) updateService(svc *service.Service) error {
	return nil
}

func (z *zkMock) removeService(svc *service.Service) error {
	return nil
}

func (z *zkMock) getSvcStates(poolID string, serviceStates *[]*servicestate.ServiceState, serviceIds ...string) error {
	return nil
}

func (z *zkMock) RegisterHost(h *host.Host) error {
	return nil
}

func (z *zkMock) UnregisterHost(h *host.Host) error {
	return nil
}

func (z *zkMock) AddVirtualIP(vip *pool.VirtualIP) error {
	return nil
}

func (z *zkMock) RemoveVirtualIP(vip *pool.VirtualIP) error {
	return nil
}

func (z *zkMock) AddResourcePool(poolID string) error {
	return nil
}

func (z *zkMock) RemoveResourcePool(poolID string) error {
	return nil
}
