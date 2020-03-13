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

// +build integration

package service

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	. "gopkg.in/check.v1"

	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain/servicedefinition"
)

var _ = Suite(&S{
	Test: elastic.Test{
		Index:    "controlplane",
		Mappings: []elastic.Mapping{MAPPING},
	},
})

type S struct {
	elastic.Test
	ctx   datastore.Context
	store Store
}

func (s *S) SetupSuite(c *C) {
	// Show Debug logs if something fails.
	plog.SetLevel(logrus.DebugLevel, true)
}

func (s *S) SetUpTest(c *C) {
	s.Test.SetUpTest(c)
	datastore.Register(s.Driver())
	s.ctx = datastore.GetContext()
	s.store = NewStore()
}

func (s *S) Test_ServiceCRUD(t *C) {
	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}

	confFile := servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"}
	svc.OriginalConfigs = map[string]servicedefinition.ConfigFile{"testname": confFile}

	svc2, err := s.store.Get(s.ctx, svc.ID)
	t.Assert(err, NotNil)
	if !datastore.IsErrNoSuchEntity(err) {
		t.Fatalf("unexpected error type: %v", err)
	}

	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	//Test update
	svc.Description = "new description"
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svc2, err = s.store.Get(s.ctx, svc.ID)
	t.Assert(err, IsNil)

	t.Assert(svc2.Description, Equals, svc.Description)
	t.Assert(len(svc2.ConfigFiles), Equals, len(svc.OriginalConfigs))
	t.Assert(svc2.ConfigFiles["testname"], Equals, svc.OriginalConfigs["testname"])

	//test delete
	err = s.store.Delete(s.ctx, svc.ID)
	t.Assert(err, IsNil)

	svc2, err = s.store.Get(s.ctx, svc.ID)
	t.Assert(err, NotNil)
	if !datastore.IsErrNoSuchEntity(err) {
		t.Fatalf("unexpected error type: %v", err)
	}

}

func (s *S) Test_FindChildService(t *C) {
	svcIn := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto", ParentServiceID: "parent_svc_id", DeploymentID: "deployment_id"}
	err := s.store.Put(s.ctx, svcIn)
	t.Assert(err, IsNil)

	svcOut, err := s.store.FindChildService(s.ctx, "deployment_id", "parent_svc_id", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, NotNil)

	svcOut, err = s.store.FindChildService(s.ctx, "not_deployment", "parent_svc_id", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcOut, err = s.store.FindChildService(s.ctx, "deployment_id", "parent_svc_id", "not_svc")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcOut, err = s.store.FindChildService(s.ctx, "deployment_id", "not_parent", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcOut, err = s.store.FindChildService(s.ctx, "deployment_id", "not_parent", "not_svc")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)
}

func (s *S) Test_FindTenantByDeploymentID(t *C) {
	svcIn := &Service{ID: "svc_test_id1", PoolID: "testPool", Name: "svc_name", Launch: "auto", ParentServiceID: "parent_svc_id", DeploymentID: "deployment"}
	err := s.store.Put(s.ctx, svcIn)
	t.Assert(err, IsNil)

	// Case 1: no service exists with deployment ID
	svcOut, err := s.store.FindTenantByDeploymentID(s.ctx, "dummy_deployment", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	// Case 2: service exists with deployment ID, but is not tenant
	svcOut, err = s.store.FindTenantByDeploymentID(s.ctx, "deployment", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	svcIn = &Service{ID: "svc_test_id2", PoolID: "testPool", Name: "svc_name", Launch: "auto", ParentServiceID: "", DeploymentID: "deployment2"}
	err = s.store.Put(s.ctx, svcIn)
	t.Assert(err, IsNil)

	// Case 3: service is tenant, but does not have deployment ID
	svcOut, err = s.store.FindTenantByDeploymentID(s.ctx, "deployment", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut, IsNil)

	// Case 4: success
	svcOut, err = s.store.FindTenantByDeploymentID(s.ctx, "deployment2", "svc_name")
	t.Assert(err, IsNil)
	t.Assert(svcOut.ID, Equals, svcIn.ID)
}

func (s *S) Test_GetServices(t *C) {
	svcs, err := s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 0)

	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)

	svc.ParentServiceID = svc.ID
	svc.ID = "Test_GetHosts2"
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 2)

	svcs, err = s.store.GetServicesByPool(s.ctx, "testPool")
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 2)

	svc.ID = "Test_GetHosts3"
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetChildServices(s.ctx, "svc_test_id")
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 2)

}

func (s *S) Test_GetUpdatedServices(t *C) {
	svcs, err := s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 0)

	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto", UpdatedAt: time.Now().Add(-time.Duration(10) * time.Second)}
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
}

func (s *S) Test_VersionConflicts(t *C) {
	svc := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	err := s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	svcs, err := s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
	t.Assert(svcs[0].DatabaseVersion, Equals, 1)

	svc2 := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	svc2.DatabaseVersion = 1
	err = s.store.Put(s.ctx, svc2)
	t.Assert(err, IsNil)

	svcs, err = s.store.GetServices(s.ctx)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
	t.Assert(svcs[0].DatabaseVersion, Equals, 2)

	svc3 := &Service{ID: "svc_test_id", PoolID: "testPool", Name: "svc_name", Launch: "auto"}
	svc3.DatabaseVersion = 1
	err = s.store.Put(s.ctx, svc3)
	t.Assert(err, Not(IsNil))
}

// Sets up the initial state of the Cache tests, adding svc_test_id.
func setInitialCacheState(s *S, t *C) *Service {
	// Verify there are no updated services initially.
	t.Log("Initial GetUpdatedservices() call")
	svcs, err := s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 0)

	// Store svc_name, last updated 10h ago.
	t.Log("Store svc_test_id with updatedAt 10h ago")
	svc := &Service{
		ID:           "svc_test_id",
		Name:         "svc_name",
		PoolID:       "testPool",
		DesiredState: int(SVCStop),
		Launch:       "auto",
		UpdatedAt:    time.Now().Add(-time.Duration(10) * time.Hour),
	}
	err = s.store.Put(s.ctx, svc)
	t.Assert(err, IsNil)

	// Validate that the DesiredState from elastic is SVCStop.
	t.Log("Validate DesiredState from elastic is SVCStop(0)")
	svcElastic, err := s.store.Get(s.ctx, svc.ID)
	t.Assert(err, IsNil)
	t.Assert(svcElastic.DesiredState, Equals, int(SVCStop))

	return svc
}

func (s *S) Test_GetUpdatedServiceStates(t *C) {
	// Setup the cache test.
	svc := setInitialCacheState(s, t)

	// Validate that we do not get this service when querying services updated in the last hour.
	t.Log("Validate no service returned with since=1h ago")
	svcs, err := s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 0)

	// Update the DesiredState
	t.Log("Updating DesiredState")
	s.store.UpdateDesiredState(s.ctx, svc.ID, int(SVCRun))

	// Validate that we get this service when querying services updated in the last hour.
	t.Log("Verify GetUpdatedServices() returns updated svc_test_id")
	svcs, err = s.store.GetUpdatedServices(s.ctx, time.Duration(1)*time.Hour)
	t.Assert(err, IsNil)
	t.Assert(len(svcs), Equals, 1)
	svcElastic := &svcs[0]
	t.Log("Verify the updated service has the cached state SVCRun(1)")
	t.Assert(svcElastic.DesiredState, Equals, int(SVCRun))
}

func (s *S) Test_GetWithCachedState(t *C) {
	// Setup the cache test.
	svc := setInitialCacheState(s, t)

	// Update the DesiredState
	t.Log("Updating DesiredState")
	s.store.UpdateDesiredState(s.ctx, svc.ID, int(SVCRun))

	// Validate that if we query for this service that we'll get the
	// updated desired state.
	t.Log("Verify Get() returns a service with cached state SVCRun(1)")
	svc, err := s.store.Get(s.ctx, svc.ID)
	t.Assert(err, IsNil)
	t.Assert(svc, NotNil)
	t.Assert(svc.DesiredState, Equals, int(SVCRun))
}

func (s *S) Test_GetServiceDetailsWithSince(c *C) {
	svca := &Service{
		ID:              "svcaid",
		PoolID:          "testPool",
		Name:            "svc_a",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id",
		UpdatedAt:       time.Now().Add(-time.Minute),
	}
	svcb := &Service{
		ID:              "svcbid",
		PoolID:          "testPool",
		Name:            "svc_b",
		Launch:          "auto",
		ParentServiceID: "svc_a",
		DeploymentID:    "deployment_id",
		UpdatedAt:       time.Now().Add(2 * -time.Second),
	}
	c.Assert(s.store.Put(s.ctx, svca), IsNil)
	c.Assert(s.store.Put(s.ctx, svcb), IsNil)

	// Query for all time, get both
	details, err := s.store.Search(s.ctx, Query{Since: time.Duration(0)})
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 2)

	// Query for last hour, get both
	details, err = s.store.Search(s.ctx, Query{Since: time.Hour})
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 2)

	// Query for last 30 seconds, get one
	details, err = s.store.Search(s.ctx, Query{Since: 30 * time.Second})
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 1)

	// Query for last second, get none
	details, err = s.store.Search(s.ctx, Query{Since: time.Second})
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 0)
}

func (s *S) Test_GetServiceDetailsByIDOrName(c *C) {
	svca := &Service{
		ID:              "svcaid",
		PoolID:          "testPool",
		Name:            "svc_a",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id",
	}
	svcb := &Service{
		ID:              "svcbid",
		PoolID:          "testPool",
		Name:            "svc_b",
		Launch:          "auto",
		ParentServiceID: "svc_a",
		DeploymentID:    "deployment_id",
	}
	svcc := &Service{
		ID:              "svccid",
		PoolID:          "testPool",
		Name:            "svc_c",
		Launch:          "auto",
		ParentServiceID: "svc_b",
		DeploymentID:    "deployment_id",
	}
	svcd := &Service{
		ID:              "svcdid",
		PoolID:          "testPool",
		Name:            "svc_d",
		Launch:          "auto",
		ParentServiceID: "svc_b",
		DeploymentID:    "deployment_id",
	}
	svcd2 := &Service{
		ID:              "svcd2id",
		PoolID:          "testPool",
		Name:            "svc_d_2",
		Launch:          "auto",
		ParentServiceID: "svc_b",
		DeploymentID:    "deployment_id",
	}
	svce := &Service{
		ID:              "svceid",
		PoolID:          "testPool",
		Name:            "svc-e",
		Launch:          "auto",
		ParentServiceID: "svc_b",
		DeploymentID:    "deployment_id",
	}
	svcdontmatch := &Service{
		ID:              "svc_a",
		PoolID:          "testPool",
		Name:            "dontmatch",
		Launch:          "auto",
		ParentServiceID: "svc_b",
		DeploymentID:    "deployment_id",
	}
	c.Assert(s.store.Put(s.ctx, svca), IsNil)
	c.Assert(s.store.Put(s.ctx, svcb), IsNil)
	c.Assert(s.store.Put(s.ctx, svcc), IsNil)
	c.Assert(s.store.Put(s.ctx, svcd), IsNil)
	c.Assert(s.store.Put(s.ctx, svcd2), IsNil)
	c.Assert(s.store.Put(s.ctx, svce), IsNil)
	c.Assert(s.store.Put(s.ctx, svcdontmatch), IsNil)

	// Get by exact ID should succeed
	details, err := s.store.GetServiceDetailsByIDOrName(s.ctx, "svcaid", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 1)
	c.Assert(details[0].ID, Equals, "svcaid")

	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "svc-e", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 1)
	c.Assert(details[0].ID, Equals, "svceid")

	// Get where substring of query matches a svc ID should fail
	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "svcaidnope", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 0)

	// Get where query matches substring of a svc ID should fail
	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "svca", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 0)

	// Get where query is a substring of many service names
	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "svc_", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 5)

	// Get where query matches both an ID and a service name
	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "svc_a", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 2)

	// Get substring vs strict ends with
	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "_d", false)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 2)

	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "_d", true)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 1)

	details, err = s.store.GetServiceDetailsByIDOrName(s.ctx, "svc_d", true)
	c.Assert(err, IsNil)
	c.Assert(details, HasLen, 1)
}

func (s *S) Test_GetAllExportedEndpoints(c *C) {
	svca := &Service{
		ID:              "svcaid",
		PoolID:          "testPool",
		Name:            "svc_a",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id",
		Endpoints: []ServiceEndpoint{
			{
				Application: "application1",
				Name:        "application1",
				Purpose:     "export",
				Protocol:    "tcp",
				PortNumber:  12345,
			}, {
				Application: "application2",
				Name:        "application2",
				Purpose:     "import",
				Protocol:    "udp",
				PortNumber:  23456,
			},
		},
	}

	svcb := &Service{
		ID:              "svcbid",
		PoolID:          "testPool",
		Name:            "svc_b",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id",
		Endpoints: []ServiceEndpoint{
			{
				Name:        "application1",
				Application: "application1",
				Purpose:     "import_all",
				Protocol:    "tcp",
				PortNumber:  44321,
			},
		},
	}

	c.Assert(s.store.Put(s.ctx, svca), IsNil)
	c.Assert(s.store.Put(s.ctx, svcb), IsNil)

	// Query to get services with exported endpoints only
	eps, err := s.store.GetAllExportedEndpoints(s.ctx)
	c.Assert(err, IsNil)
	c.Assert(eps, HasLen, 1)
	c.Assert(eps[0], DeepEquals, ExportedEndpoint{
		ServiceID:   svca.ID,
		ServiceName: svca.Name,
		Application: "application1",
		Protocol:    "tcp",
	})
}

func (s *S) Test_GetImageUseCount(c *C) {
	imageID := "image-1-id"
	svca := &Service{
		ID:              "svcaid",
		PoolID:          "testPool",
		Name:            "svc_a",
		Launch:          "auto",
		ParentServiceID: "",
		DeploymentID:    "deployment_id",
		UpdatedAt:       time.Now().Add(-time.Minute),
		ImageID:         imageID,
	}
	svcb := &Service{
		ID:              "svcbid",
		PoolID:          "testPool",
		Name:            "svc_b",
		Launch:          "auto",
		ParentServiceID: "svc_a",
		DeploymentID:    "deployment_id",
		UpdatedAt:       time.Now().Add(2 * -time.Second),
		ImageID:         imageID,
	}
	c.Assert(s.store.Put(s.ctx, svca), IsNil)
	c.Assert(s.store.Put(s.ctx, svcb), IsNil)

	count, err := s.store.GetServiceCountByImage(s.ctx, imageID)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 2)

	err = s.store.Delete(s.ctx, svca.ID)
	c.Assert(err, IsNil)
	count, err = s.store.GetServiceCountByImage(s.ctx, imageID)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 1)

	err = s.store.Delete(s.ctx, svcb.ID)
	c.Assert(err, IsNil)
	count, err = s.store.GetServiceCountByImage(s.ctx, imageID)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 0)
}
