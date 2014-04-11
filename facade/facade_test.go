// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	. "gopkg.in/check.v1"
	"testing"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&FacadeTest{})

type FacadeTest struct {
	elastic.ElasticTest
	ctx    datastore.Context
	facade *Facade
}

func (tt *FacadeTest) SetUpSuite(c *C) {
	//set up index and mappings before setting up elastic
	tt.Index = "controlplane"
	tt.Mappings = map[string]string{
		"host":         "../domain/host/host_mapping.json",
		"resourcepool": "../domain/pool/pool_mapping.json",
	}
	tt.ElasticTest.SetUpSuite(c)
	datastore.Register(tt.Driver())
	tt.ctx = datastore.Get()
	tt.facade = New()
}
