// Copyright 2016 The Serviced Authors.
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

// +build integration

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

func (s *IntegrationTest) TestServiceConfig_CRUD(c *C) {
	svcA := service.Service{
		ID:           "servicea",
		Name:         "serviceA",
		DeploymentID: "deploymentid",
		PoolID:       "poolid",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
	}

	c.Assert(s.Facade.AddService(s.CTX, svcA), IsNil)

	// get configs for service that does not exist
	files, err := s.Facade.GetServiceConfigs(s.CTX, "badservice")
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
	c.Assert(files, IsNil)

	// get configs for a service that does exist (=0)
	files, err = s.Facade.GetServiceConfigs(s.CTX, "servicea")
	c.Assert(err, IsNil)
	c.Assert(files, HasLen, 0)

	// create a config
	conf := servicedefinition.ConfigFile{
		Filename:    "conf.txt",
		Owner:       "root",
		Permissions: "rwx",
		Content:     "some content",
	}

	err = s.Facade.AddServiceConfig(s.CTX, "servicea", conf)
	c.Assert(err, IsNil)

	// get the created config (=1)
	files, err = s.Facade.GetServiceConfigs(s.CTX, "servicea")
	c.Assert(err, IsNil)
	c.Assert(files, HasLen, 1)
	c.Assert(files[0].Filename, Equals, "conf.txt")

	// get the config
	f, err := s.Facade.GetServiceConfig(s.CTX, files[0].ID)
	c.Assert(err, IsNil)
	c.Assert(*f, DeepEquals, conf)

	// create another config
	conf2 := servicedefinition.ConfigFile{
		Filename:    "conf2.txt",
		Owner:       "newowner",
		Permissions: "waq",
		Content:     "other content",
	}

	err = s.Facade.AddServiceConfig(s.CTX, "servicea", conf2)
	c.Assert(err, IsNil)

	// get the created config (>1)
	files, err = s.Facade.GetServiceConfigs(s.CTX, "servicea")
	c.Assert(err, IsNil)
	c.Assert(files, HasLen, 2)

	confids := make(map[string]string)
	confmap := make(map[string]servicedefinition.ConfigFile)
	for _, file := range files {
		f, err := s.Facade.GetServiceConfig(s.CTX, file.ID)
		c.Assert(err, IsNil)
		confmap[file.Filename] = *f
		confids[file.Filename] = file.ID
	}
	c.Assert(confmap["conf.txt"], DeepEquals, conf)
	c.Assert(confmap["conf2.txt"], DeepEquals, conf2)

	// update the conf
	conf.Owner = "zenoss"
	err = s.Facade.UpdateServiceConfig(s.CTX, confids["conf.txt"], conf)
	c.Assert(err, IsNil)

	f, err = s.Facade.GetServiceConfig(s.CTX, confids["conf.txt"])
	c.Assert(err, IsNil)
	c.Assert(*f, DeepEquals, conf)

	f, err = s.Facade.GetServiceConfig(s.CTX, confids["conf2.txt"])
	c.Assert(err, IsNil)
	c.Assert(*f, DeepEquals, conf2)

	// delete the conf
	err = s.Facade.DeleteServiceConfig(s.CTX, confids["conf2.txt"])
	c.Assert(err, IsNil)

	f, err = s.Facade.GetServiceConfig(s.CTX, confids["conf.txt"])
	c.Assert(err, IsNil)
	c.Assert(*f, DeepEquals, conf)

	f, err = s.Facade.GetServiceConfig(s.CTX, confids["conf2.txt"])
	c.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)
}
