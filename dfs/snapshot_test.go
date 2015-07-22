// Copyright 2015 The Serviced Authors.
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

package dfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/control-center/serviced/commons/docker"
	dockertest "github.com/control-center/serviced/commons/docker/test"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	facadetest "github.com/control-center/serviced/facade/test"
	"github.com/control-center/serviced/volume"
	volumetest "github.com/control-center/serviced/volume/test"
	dockerclient "github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"

	"github.com/stretchr/testify/mock"
)

// snapshotTest test type for setting up mocks and other resources needed by these tests
type snapshotTest struct {
	dfs *DistributedFilesystem

	//  A mock implementatino of FacadeInterface
	mockFacade *facadetest.MockFacade

	// Used to mock the response from dfs.mountVolume()
	mountVolumeResponse mockMountVolumeResponse

	// The path to the temporary directory used by some test cases
	tmpDir string

	// Function used by commons/docker to get an instance of docker.ClientInterface
	dockerClientGetter docker.DockerClientGetter
}

const (
	testTenantID = "testTenantID"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&snapshotTest{})

func (st *snapshotTest) SetUpTest(c *C) {
	st.mockFacade = &facadetest.MockFacade{}
	// st.dfs.facade = st.mockFacade
	st.dfs = &DistributedFilesystem{
		fsType:           "rsync",
		varpath:          "/tmp",
		dockerHost:       "localhost",
		dockerPort:       5000,
		facade:           st.mockFacade,
		timeout:          time.Minute * 5,
		lock:             nil,
		datastoreGet:     st.mock_datastoreGet,
		getServiceVolume: st.mock_getServiceVolume,
	}
}

func (st *snapshotTest) TearDownTest(c *C) {
	// don't allow per-test-case values to be reused across test cases
	st.dfs = nil
	st.mockFacade = nil
	st.mountVolumeResponse.volume = nil
	st.mountVolumeResponse.err = nil
	st.dockerClientGetter = nil
	if st.tmpDir != "" {
		os.RemoveAll(st.tmpDir)
		st.tmpDir = ""
	}
}

func (st *snapshotTest) TestSnapshot_Snapshot_GetServiceFails(c *C) {
	errorStub := errors.New("errorStub: GetService() failed")
	st.mockFacade.
		On("GetService", st.mock_datastoreGet(), testTenantID).
		Return(nil, errorStub)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_Snapshot_ServiceNotFound(c *C) {
	st.mockFacade.
		On("GetService", st.mock_datastoreGet(), testTenantID).
		Return(nil, nil)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, NotNil)
}

func (st *snapshotTest) TestSnapshot_Snapshot_GetServicesFails(c *C) {
	st.setupSimpleGetService()
	errorStub := errors.New("errorStub: GetServices() failed")
	st.mockFacade.
		On("GetServices", st.mock_datastoreGet(), dao.ServiceRequest{TenantID: testTenantID}).
		Return(nil, errorStub)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_Snapshot_ServicePauseFails(c *C) {
	st.setupSimpleGetService()
	stubServices := []service.Service{
		service.Service{ID: "servceID1", DesiredState: int(service.SVCRun)},
	}
	st.mockFacade.
		On("GetServices", st.mock_datastoreGet(), dao.ServiceRequest{TenantID: testTenantID}).
		Return(stubServices, nil)

	// Mock the defer call
	st.mockFacade.
		On("ScheduleService", st.mock_datastoreGet(), stubServices[0].ID, false, stubServices[0].DesiredState).
		Return(0, nil)

	// Mock an error trying to pause the service
	errorStub := errors.New("errorStub: ScheduleService(SVCPause) failed")
	st.mockFacade.
		On("ScheduleService", st.mock_datastoreGet(), stubServices[0].ID, false, service.SVCPause).
		Return(0, errorStub)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_Snapshot_WaitForPauseFails(c *C) {
	waitForPauseError := errors.New("errorStub: WaitService() failed")
	st.setupWaitForServicesToBePaused(waitForPauseError)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, waitForPauseError)
}

func (st *snapshotTest) TestSnapshot_Snapshot_VolumeNotFound(c *C) {
	st.setupWaitForServicesToBePaused(nil)
	errorStub := errors.New("errorStub: GetVolume() failed")
	st.mountVolumeResponse.volume = &volumetest.MockVolume{}
	st.mountVolumeResponse.err = errorStub

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_Snapshot_SnapshotFailed(c *C) {
	svcs := st.setupWaitForServicesToBePaused(nil)
	mockVol := st.setupMockSnapshotVolume(c, svcs[0].ID)

	errorStub := errors.New("errorStub: Snapshot() failed")
	mockVol.On("Snapshot", mock.AnythingOfTypeArgument("string")).Return(errorStub)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_Snapshot_TagFailed(c *C) {
	svcs := st.setupWaitForServicesToBePaused(nil)
	mockVol := st.setupMockSnapshotVolume(c, svcs[0].ID)
	mockVol.On("Snapshot", mock.AnythingOfTypeArgument("string")).Return(nil)

	mockClient := st.setupMockDockerClient()
	errorStub := errors.New("errorStub: Tag() failed")
	mockClient.On("ListImages", dockerclient.ListImagesOptions{All: false}).Return(nil, errorStub)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, "description")

	c.Assert(snapshotLabel, Equals, "")
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_Snapshot_WithDescription(c *C) {
	nonEmptyDescription := "description"

	st.testSnapshot(c, nonEmptyDescription)
}

func (st *snapshotTest) TestSnapshot_Snapshot_WithoutDescription(c *C) {
	emptyDescription := ""

	st.testSnapshot(c, emptyDescription)
}

func (st *snapshotTest) testSnapshot(c *C, description string) {
	svcs := st.setupWaitForServicesToBePaused(nil)
	mockVol := st.setupMockSnapshotVolume(c, svcs[0].ID)
	mockVol.On("Snapshot", mock.AnythingOfTypeArgument("string")).Return(nil)

	mockClient := st.setupMockDockerClient()
	mockClient.On("ListImages", dockerclient.ListImagesOptions{All: false}).Return(nil, nil)

	snapshotLabel, err := st.dfs.Snapshot(testTenantID, description)

	partialTagID := time.Now().UTC().Format("20060102")
	partialLabel := fmt.Sprintf("%s_%s.*", testTenantID, partialTagID)
	c.Assert(snapshotLabel, Matches, partialLabel)
	c.Assert(err, IsNil)

	servicesFile := filepath.Join(mockVol.Path(), serviceJSON)
	st.assertServicesJSON(c, svcs, servicesFile)

	metadataFile := filepath.Join(mockVol.Path(), snapshotMeta)
	st.assertSnapshotMetadata(c, description, metadataFile)
}

func (st *snapshotTest) TestSnapshot_ListSnapshots_GetServiceFails(c *C) {
	errorStub := errors.New("errorStub: GetService() unavailable")
	st.mockFacade.
		On("GetService", st.mock_datastoreGet(), testTenantID).
		Return(nil, errorStub)

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	c.Assert(snapshots, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_ListSnapshots_ServiceNotFound(c *C) {
	st.mockFacade.
		On("GetService", st.mock_datastoreGet(), testTenantID).
		Return(nil, nil)

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	c.Assert(snapshots, IsNil)
	c.Assert(err, NotNil)
}

func (st *snapshotTest) TestSnapshot_ListSnapshots_VolumeNotFound(c *C) {
	st.setupSimpleGetService()
	errorStub := errors.New("errorStub: GetVolume() failed")
	st.mountVolumeResponse.volume = &volumetest.MockVolume{}
	st.mountVolumeResponse.err = errorStub

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	c.Assert(snapshots, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_ListSnapshots_GetVolumeSnapshotsFail(c *C) {
	st.setupSimpleGetService()
	errorStub := errors.New("errorStub: Snapshots() failed")
	mockVol := &volumetest.MockVolume{}
	mockVol.On("Snapshots").Return(nil, errorStub)
	st.mountVolumeResponse.volume = mockVol
	st.mountVolumeResponse.err = nil

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	c.Assert(snapshots, IsNil)
	c.Assert(err, Equals, errorStub)
}

func (st *snapshotTest) TestSnapshot_ListSnapshots_EmptyResult(c *C) {
	st.setupSimpleGetService()
	mockVol := &volumetest.MockVolume{}
	mockVol.On("Snapshots").Return(nil, nil)
	st.mountVolumeResponse.volume = mockVol
	st.mountVolumeResponse.err = nil

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	c.Assert(snapshots, HasLen, 0)
	c.Assert(err, IsNil)
}

func (st *snapshotTest) TestSnapshot_ListSnapshots_WithDescriptions(c *C) {
	snapshotIDs := []string{
		"snapshot1",
		"snapshot2",
	}
	descriptions := []string{
		"description1",
		"description2",
	}
	st.setupListSnapshots(c, snapshotIDs, descriptions)

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	st.assertListSnapshots(c, snapshotIDs, descriptions, snapshots, err)
}

// Snapshots without descriptions represent snapshots created prior to CC-577
func (st *snapshotTest) TestSnapshot_ListSnapshots_WithoutDescriptions(c *C) {
	snapshotIDs := []string{
		"snapshot1",
		"snapshot2",
	}
	st.setupListSnapshots(c, snapshotIDs, nil)

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	st.assertListSnapshots(c, snapshotIDs, nil, snapshots, err)
}

// Snapshots with empty descriptions represent an edge case (unexpected content in snapshot.json)
func (st *snapshotTest) TestSnapshot_ListSnapshots_WithEmptyDescriptions(c *C) {
	snapshotIDs := []string{
		"snapshot1",
		"snapshot2",
	}
	descriptions := []string{
		"",
		"",
	}
	st.setupListSnapshots(c, snapshotIDs, descriptions)

	snapshots, err := st.dfs.ListSnapshots(testTenantID)

	st.assertListSnapshots(c, snapshotIDs, descriptions, snapshots, err)
}

// Mock for datastore.Get()
func (st *snapshotTest) mock_datastoreGet() datastore.Context {
	return nil // we don't need a datastore.Context to unit-test snapshots
}

// Mock for volume.Mount()
func (st *snapshotTest) mock_getServiceVolume(fsType, serviceID, baseDir string) (volume.Volume, error) {
	return st.mountVolumeResponse.volume, st.mountVolumeResponse.err
}

// A set of values used to mock the response from dfs.mountVolume()
type mockMountVolumeResponse struct {
	volume *volumetest.MockVolume
	err    error
}

// Sets up a simple mock for dfs.GetService() which can used for a variety of test cases
func (st *snapshotTest) setupSimpleGetService() {
	service := &service.Service{ID: "test service id"}
	st.mockFacade.
		On("GetService", st.mock_datastoreGet(), testTenantID).
		Return(service, nil)
}

func (st *snapshotTest) setupWaitForServicesToBePaused(errorStub error) []service.Service {
	st.setupSimpleGetService()
	stubServices := []service.Service{
		service.Service{ID: "servceID1", DesiredState: int(service.SVCPause)},
	}
	st.mockFacade.
		On("GetServices", st.mock_datastoreGet(), dao.ServiceRequest{TenantID: testTenantID}).
		Return(stubServices, nil)

	st.mockFacade.
		On("WaitService", st.mock_datastoreGet(), service.SVCPause, st.dfs.timeout, []string{stubServices[0].ID}).
		Return(errorStub)
	return stubServices
}

func (st *snapshotTest) setupMockSnapshotVolume(c *C, serviceID string) *volumetest.MockVolume {
	snapshotDir := st.makeSnapshotDir(c, serviceID)

	mockVol := &volumetest.MockVolume{}
	mockVol.On("Path").Return(snapshotDir)
	st.mountVolumeResponse.volume = mockVol
	st.mountVolumeResponse.err = nil
	return mockVol
}

func (st *snapshotTest) setupMockDockerClient() *dockertest.MockDockerClient {
	mockClient := &dockertest.MockDockerClient{}
	st.dockerClientGetter = func() (docker.ClientInterface, error) {
		return mockClient, nil
	}
	docker.SetDockerClientGetter(st.dockerClientGetter)
	return mockClient
}

// Get a temporary directory for files created by this unit-test.
// NOTE: the caller is responsible for deleting the directory
func (st *snapshotTest) getTmpDir(c *C) string {
	tmpDir, err := ioutil.TempDir("", "test-serviced-dfs-snapshot")
	if err != nil {
		c.Fatalf("Failed to create temporary directory: %s", err)
	}
	st.tmpDir = tmpDir
	return tmpDir
}

func (st *snapshotTest) makeSnapshotDir(c *C, serviceID string) string {
	// use only 1 tmpDir per test case
	if st.tmpDir == "" {
		st.getTmpDir(c)
	}

	snapshotDir := filepath.Join(st.tmpDir, serviceID)
	if err := os.Mkdir(snapshotDir, 0700); err != nil {
		c.Fatalf("Failed creating directory %s: %s", snapshotDir, err)
	}
	return snapshotDir
}

func (st *snapshotTest) assertServicesJSON(c *C, services []service.Service, servicesFile string) {
	data, e := ioutil.ReadFile(servicesFile)
	if e != nil {
		c.Fatalf("Failed to read services JSON file %s: %s", servicesFile, e)
	}

	svcsJSON, e2 := json.Marshal(services)
	if e2 != nil {
		c.Fatalf("Failed to marshall services into JSON: %s", e2)
	}
	c.Assert(strings.TrimSpace(string(data)), Equals, string(svcsJSON))
}

func (st *snapshotTest) assertSnapshotMetadata(c *C, description, metadataFile string) {
	data, e := ioutil.ReadFile(metadataFile)
	if e != nil {
		c.Fatalf("Failed to read metadata file %s: %s", metadataFile, e)
	}

	metadata := SnapshotMetadata{Description: description}
	metadataJSON, e2 := json.Marshal(metadata)
	if e2 != nil {
		c.Fatalf("Failed to marshall services into JSON: %s", e2)
	}
	c.Assert(strings.TrimSpace(string(data)), Equals, string(metadataJSON))
}

func (st *snapshotTest) setupListSnapshots(c *C, snapshotIDs, descriptions []string) {
	st.setupSimpleGetService()

	mockVol := &volumetest.MockVolume{}
	mockVol.On("Snapshots").Return(snapshotIDs, nil)
	st.mountVolumeResponse.volume = mockVol
	st.mountVolumeResponse.err = nil

	// Make separate test directories for each snapshot
	for i, id := range snapshotIDs {
		snapshotDir := st.makeSnapshotDir(c, id)
		mockVol.On("SnapshotMetadataPath", id).Return(snapshotDir)

		if descriptions != nil {
			st.writeDescription(c, snapshotDir, descriptions[i])
		}
	}
}

func (st *snapshotTest) assertListSnapshots(
	c *C,
	expectedSnapshotIDs, expectedDescriptions []string,
	snapshots []dao.SnapshotInfo,
	err error) {

	c.Assert(err, IsNil)
	c.Assert(len(snapshots), Equals, len(expectedSnapshotIDs))
	for i, snapshot := range snapshots {
		c.Assert(expectedSnapshotIDs[i], Equals, snapshot.SnapshotID)
		if expectedDescriptions != nil {
			c.Assert(snapshot.Description, Equals, expectedDescriptions[i])
		}
	}
}

// Write the description to the a metadata file in the snapshot directory.
func (st *snapshotTest) writeDescription(c *C, snapshotDir string, description string) {
	jsonString := fmt.Sprintf("{ \"description\": %q}\n", description)
	jsonFile := filepath.Join(snapshotDir, snapshotMeta)
	if err := ioutil.WriteFile(jsonFile, []byte(jsonString), 0600); err != nil {
		c.Fatalf("Failed writing file %s: %s", jsonFile, err)
	}
}
