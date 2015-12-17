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

// +build integration

package facade

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"

	dfsmocks "github.com/control-center/serviced/dfs/mocks"
	zzkmocks "github.com/control-center/serviced/facade/mocks"
	"github.com/stretchr/testify/mock"
	gocheck "gopkg.in/check.v1"
)

var (
	testDataRootDir        = "/tmp/dfsregistrytest"
	remoteRegistryEndpoint = "1.2.3.4:5000"
)

var dfsRegistrySvcDefs = [...]service.Service{
	service.Service{
		ID:           "dfsrgysvc1",
		Name:         "TestDfsRegistry1",
		DeploymentID: "testdfsdeployment",
		ImageID:      "zenoss/testimage1",
		PoolID:       "pool_id",
		Launch:       "auto",
	},
	service.Service{
		ID:           "dfsrgysvc2",
		Name:         "TestDfsRegistry2",
		DeploymentID: "testdfsdeployment",
		ImageID:      "zenoss/testimage2",
		PoolID:       "pool_id",
		Launch:       "auto",
	},
}

//-----------------------------------------------------------------------
// FacadeDfsRegistryTest and other support functions

var _ = gocheck.Suite(&FacadeDfsRegistryTest{})

type FacadeDfsRegistryTest struct {
	elastic.ElasticTest
	CTX    datastore.Context
	Facade *Facade
	zzk    *zzkmocks.ZZK
	dfs    *dfsmocks.DFS

	registryVersionInfo registryVersionInfo
	v1RegistryRootDir   string
}

func (fdrt *FacadeDfsRegistryTest) SetUpSuite(c *gocheck.C) {
	//set up index and mappings before setting up elastic
	fdrt.Index = "controlplane"
	if fdrt.Mappings == nil {
		fdrt.Mappings = make([]elastic.Mapping, 0)
	}
	fdrt.Mappings = append(fdrt.Mappings, host.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, pool.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, service.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, servicetemplate.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, addressassignment.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, serviceconfigfile.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, user.MAPPING)
	fdrt.Mappings = append(fdrt.Mappings, registry.MAPPING)

	fdrt.ElasticTest.SetUpSuite(c)
	datastore.Register(fdrt.Driver())
	fdrt.CTX = datastore.Get()

	fdrt.Facade = New()

	fdrt.registryVersionInfo = registryVersionInfos[1] // Currently, only old registry is v1
	fdrt.cleanUpOldRegistry(c)                         // Just to be sure
}

func (fdrt *FacadeDfsRegistryTest) SetUpTest(c *gocheck.C) {
	fdrt.ElasticTest.SetUpTest(c)
	fdrt.zzk = &zzkmocks.ZZK{}
	fdrt.Facade.SetZZK(fdrt.zzk)
	fdrt.dfs = &dfsmocks.DFS{}
	fdrt.Facade.SetDFS(fdrt.dfs)
	fdrt.setupMockZZK()

	fdrt.Facade.SetIsvcsPath(testDataRootDir)

	fdrt.addServices(c)
	fdrt.setupOldRegistry(c)
}

func (fdrt *FacadeDfsRegistryTest) TearDownTest(c *gocheck.C) {
	fdrt.cleanUpOldRegistry(c)
}

func (fdrt *FacadeDfsRegistryTest) TearDownSuite(c *gocheck.C) {
	if exists := fdrt.oldRegistryImageExists(c); exists {
		cmd := exec.Command("docker", "rmi", "--force", fdrt.registryVersionInfo.imageId)
		if err := cmd.Run(); err != nil {
			c.Errorf("Could not delete image %s: %s", fdrt.registryVersionInfo.imageId, err)
		}
	}
}

func (fdrt *FacadeDfsRegistryTest) setupMockZZK() {
	fdrt.zzk.On("AddResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	fdrt.zzk.On("UpdateResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	fdrt.zzk.On("RemoveResourcePool", mock.AnythingOfType("string")).Return(nil)
	fdrt.zzk.On("AddVirtualIP", mock.AnythingOfType("*pool.VirtualIP")).Return(nil)
	fdrt.zzk.On("RemoveVirtualIP", mock.AnythingOfType("*pool.VirtualIP")).Return(nil)
	fdrt.zzk.On("AddHost", mock.AnythingOfType("*host.Host")).Return(nil)
	fdrt.zzk.On("UpdateHost", mock.AnythingOfType("*host.Host")).Return(nil)
	fdrt.zzk.On("RemoveHost", mock.AnythingOfType("*host.Host")).Return(nil)
	fdrt.zzk.On("UpdateService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	fdrt.zzk.On("RemoveService", mock.AnythingOfType("*service.Service")).Return(nil)
	fdrt.zzk.On("SetRegistryImage", mock.AnythingOfType("*registry.Image")).Return(nil)
	fdrt.zzk.On("DeleteRegistryImage", mock.AnythingOfType("string")).Return(nil)
	fdrt.zzk.On("DeleteRegistryLibrary", mock.AnythingOfType("string")).Return(nil)
	fdrt.zzk.On("LockServices", mock.AnythingOfType("[]service.Service")).Return(nil)
	fdrt.zzk.On("UnlockServices", mock.AnythingOfType("[]service.Service")).Return(nil)
}

func (fdrt *FacadeDfsRegistryTest) getOldRegistryContainerName() string {
	return fmt.Sprintf(oldLocalRegistryContainerNameBase, fdrt.registryVersionInfo.version)
}

func (fdrt *FacadeDfsRegistryTest) getRegistryPath(c *gocheck.C) string {
	return filepath.Join(testDataRootDir, registryRootSubdir, fdrt.registryVersionInfo.rootDir)
}

func (fdrt *FacadeDfsRegistryTest) setupOldRegistry(c *gocheck.C) {
	fdrt.v1RegistryRootDir = fdrt.getRegistryPath(c)
	os.MkdirAll(fdrt.v1RegistryRootDir, 0755)
}

func (fdrt *FacadeDfsRegistryTest) oldRegistryContainerExists(c *gocheck.C) (found bool) {
	return execGrep(c, "docker ps --all", fdrt.getOldRegistryContainerName())
}

func (fdrt *FacadeDfsRegistryTest) oldRegistryImageExists(c *gocheck.C) (found bool) {
	return execGrep(c, "docker images", fdrt.registryVersionInfo.imageId)
}

func execGrep(c *gocheck.C, command string, stringToFind string) (found bool) {
	found = false

	commandAndArgs := strings.Split(command, " ")
	cmd := exec.Command(commandAndArgs[0], commandAndArgs[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.Errorf("Could not read output from go exec.Command: %s", err)
		return
	}
	if err := cmd.Start(); err != nil {
		c.Errorf("Could not start go exec.Command('docker ps --all'): %s", err)
		return
	}
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), stringToFind) {
			found = true
			break
		}
	}

	return
}

func (fdrt *FacadeDfsRegistryTest) cleanUpOldRegistry(c *gocheck.C) {
	if exists := fdrt.oldRegistryContainerExists(c); exists {
		cmd := exec.Command("docker", "rm", "--force", fdrt.getOldRegistryContainerName())
		if err := cmd.Run(); err != nil {
			c.Fatalf("Cleanup failed: Could not delete container %s: %s", fdrt.getOldRegistryContainerName(), err)
		}
	}

	os.RemoveAll(fdrt.v1RegistryRootDir)
}

func (fdrt *FacadeDfsRegistryTest) verifyOldRegistryContainerExists(c *gocheck.C, exists bool) {
	found := fdrt.oldRegistryContainerExists(c)
	if exists != found {
		c.Errorf("Expected old registry container exists == %t, but found == %t", exists, found)
	}
}

func (fdrt *FacadeDfsRegistryTest) addServices(c *gocheck.C) {
	for _, svc := range dfsRegistrySvcDefs {
		if err := fdrt.Facade.AddService(fdrt.CTX, svc); err != nil {
			c.Fatalf("Setup failed: Could not add service %s: %s", svc.ID, err)
			return
		}
	}
}

// Must be called before calling Facade.UpgradeRegistry()
func (fdrt *FacadeDfsRegistryTest) verifyAllImagesUpgraded(c *gocheck.C, endpoint string, force bool) {
	fdrt.dfs.On("UpgradeRegistry", mock.AnythingOfType("[]service.Service"), dfsRegistrySvcDefs[0].ID, endpoint, force).Return(nil).Run(func(args mock.Arguments) {
		svcs := args.Get(0).([]service.Service)
		c.Assert(len(svcs), gocheck.Equals, 1)
		c.Assert(svcs[0].ID, gocheck.Equals, dfsRegistrySvcDefs[0].ID)
		c.Assert(svcs[0].Name, gocheck.Equals, dfsRegistrySvcDefs[0].Name)
		c.Assert(svcs[0].DeploymentID, gocheck.Equals, dfsRegistrySvcDefs[0].DeploymentID)
		c.Assert(svcs[0].ImageID, gocheck.Equals, dfsRegistrySvcDefs[0].ImageID)
		c.Assert(svcs[0].PoolID, gocheck.Equals, dfsRegistrySvcDefs[0].PoolID)
	})
	fdrt.dfs.On("UpgradeRegistry", mock.AnythingOfType("[]service.Service"), dfsRegistrySvcDefs[1].ID, endpoint, force).Return(nil).Run(func(args mock.Arguments) {
		svcs := args.Get(0).([]service.Service)
		c.Assert(len(svcs), gocheck.Equals, 1)
		c.Assert(svcs[0].ID, gocheck.Equals, dfsRegistrySvcDefs[1].ID)
		c.Assert(svcs[0].Name, gocheck.Equals, dfsRegistrySvcDefs[1].Name)
		c.Assert(svcs[0].DeploymentID, gocheck.Equals, dfsRegistrySvcDefs[1].DeploymentID)
		c.Assert(svcs[0].ImageID, gocheck.Equals, dfsRegistrySvcDefs[1].ImageID)
		c.Assert(svcs[0].PoolID, gocheck.Equals, dfsRegistrySvcDefs[1].PoolID)
	})
}

func (fdrt *FacadeDfsRegistryTest) verifyNoImagesUpgraded(c *gocheck.C) {
	fdrt.dfs.AssertNotCalled(c, "UpgradeRegistry", mock.AnythingOfType("[]service.Service"), mock.AnythingOfType("string"), mock.AnythingOfType("string"))
}

func (fdrt *FacadeDfsRegistryTest) markOldRegistryUpgraded(c *gocheck.C) {
	markerFile := filepath.Join(fdrt.getRegistryPath(c), upgradedMarkerFile)
	if err := ioutil.WriteFile(markerFile, []byte{}, 0644); err != nil {
		c.Fatalf("Could not create 'upgraded' marker file %s: %s", markerFile, err)
	}
}

func getOldRegistryEndpoint() string {
	return fmt.Sprintf("localhost:%s", oldLocalRegistryPort)
}

//-----------------------------------------------------------------------
// Test Cases

func (fdrt *FacadeDfsRegistryTest) TestUpgradeRegistry_UpgradeLocal(c *gocheck.C) {
	fdrt.verifyAllImagesUpgraded(c, getOldRegistryEndpoint(), false)

	err := fdrt.Facade.UpgradeRegistry(fdrt.CTX, "", false)

	c.Assert(err, gocheck.IsNil)
	fdrt.verifyOldRegistryContainerExists(c, true)
}

func (fdrt *FacadeDfsRegistryTest) TestUpgradeRegistry_SkipUpgradedLocal(c *gocheck.C) {
	fdrt.markOldRegistryUpgraded(c)

	err := fdrt.Facade.UpgradeRegistry(fdrt.CTX, "", false)

	c.Assert(err, gocheck.IsNil)
	fdrt.verifyNoImagesUpgraded(c)
	fdrt.verifyOldRegistryContainerExists(c, false)
}

func (fdrt *FacadeDfsRegistryTest) TestUpgradeRegistry_ForceLocal(c *gocheck.C) {
	fdrt.markOldRegistryUpgraded(c)

	fdrt.verifyAllImagesUpgraded(c, getOldRegistryEndpoint(), true)

	err := fdrt.Facade.UpgradeRegistry(fdrt.CTX, "", true)

	c.Assert(err, gocheck.IsNil)
	fdrt.verifyOldRegistryContainerExists(c, true)
}

func (fdrt *FacadeDfsRegistryTest) TestUpgradeRegistry_ForceLocalButNoRegistry(c *gocheck.C) {
	fdrt.cleanUpOldRegistry(c)

	err := fdrt.Facade.UpgradeRegistry(fdrt.CTX, "", true)

	c.Assert(err, gocheck.IsNil) // Should report there was no registry to upgrade
	fdrt.verifyNoImagesUpgraded(c)
	fdrt.verifyOldRegistryContainerExists(c, false)
}

func (fdrt *FacadeDfsRegistryTest) TestUpgradeRegistry_UpgradeRemote(c *gocheck.C) {
	fdrt.verifyAllImagesUpgraded(c, remoteRegistryEndpoint, false)

	err := fdrt.Facade.UpgradeRegistry(fdrt.CTX, remoteRegistryEndpoint, false)

	c.Assert(err, gocheck.IsNil)
	fdrt.verifyOldRegistryContainerExists(c, false)
}

func (fdrt *FacadeDfsRegistryTest) TestUpgradeRegistry_UpgradeRemoteForce(c *gocheck.C) {
	fdrt.verifyAllImagesUpgraded(c, remoteRegistryEndpoint, true)

	err := fdrt.Facade.UpgradeRegistry(fdrt.CTX, remoteRegistryEndpoint, true)

	c.Assert(err, gocheck.IsNil)
	fdrt.verifyOldRegistryContainerExists(c, false)
}
