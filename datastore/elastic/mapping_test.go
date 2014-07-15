// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elastic_test

import (
	"github.com/zenoss/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"encoding/json"
	"io/ioutil"
	"testing"
)

// This plumbs gocheck into testing
func TestelasticMappings(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&mt{})

type mt struct {
}

func (s *mt) TestJSON(c *C) {
	bytes, err := ioutil.ReadFile("testmapping.json")
	c.Assert(err, IsNil)
	var mapping elastic.Mapping

	err = json.Unmarshal(bytes, &mapping)
	c.Assert(err, IsNil)

	c.Assert(mapping.Name, Equals, "testentity")

	c.Assert(len(mapping.Entries), Equals, 1)

	c.Assert(mapping.Entries["properties"], NotNil)
	var props map[string]interface{}
	props = mapping.Entries["properties"].(map[string]interface{})
	c.Assert(len(props), Equals, 2)
	c.Assert(props["ID"], NotNil)
	c.Assert(props["Name"], NotNil)

}

func (s *mt) TestBadJSON(c *C) {
	bytes, err := ioutil.ReadFile("badmapping.json")
	var mapping elastic.Mapping

	err = json.Unmarshal(bytes, &mapping)
	c.Assert(err, NotNil)
}

func (s *mt) TestMarshal(c *C) {
	//read the good file into built-in types
	goodBytes, err := ioutil.ReadFile("testmapping.json")
	c.Assert(err, IsNil)

	var goodMapping map[string]interface{}
	err = json.Unmarshal(goodBytes, &goodMapping)
	c.Assert(err, IsNil)

	//unmarshal good bytes into a Mapping
	var mapping elastic.Mapping
	err = json.Unmarshal(goodBytes, &mapping)
	c.Assert(err, IsNil)

	//write mapping to bytes
	bytes, err := json.Marshal(mapping)
	c.Assert(err, IsNil)

	//unmarshal mapping bytes and compare to unmarshalled good bytes
	var testMapping map[string]interface{}
	err = json.Unmarshal(bytes, &testMapping)
	c.Assert(err, IsNil)
	c.Assert(testMapping, DeepEquals, goodMapping)
}
