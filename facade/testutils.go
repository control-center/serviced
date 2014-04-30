// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
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

	ft.ElasticTest.SetUpSuite(c)
	datastore.Register(ft.Driver())
	ft.CTX = datastore.Get()
	ft.Facade = New()
}
