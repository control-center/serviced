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

// +build unit

package facade_test

import (
	"github.com/control-center/serviced/facade"
	. "gopkg.in/check.v1"
	"time"
)

func (ft *FacadeUnitTest) Test_NewPendingDeployment(c *C) {
	deploymentID, templateID, poolID := "a", "b", "c"
	pd := facade.NewPendingDeployment(deploymentID, templateID, poolID)
	info := pd.GetInfo()
	c.Assert(info["TemplateID"], Equals, templateID)
	c.Assert(info["PoolID"], Equals, poolID)
	c.Assert(info["DeploymentID"], Equals, deploymentID)
	c.Assert(info["status"], Equals, "Starting")
	c.Assert(info["templateName"], Equals, "")
}

func (ft *FacadeUnitTest) Test_PendingDeployment_GetStatus(c *C) {
	deploymentID, templateID, poolID := "a", "b", "c"
	pd := facade.NewPendingDeployment(deploymentID, templateID, poolID)
	status, notify := pd.GetStatus()
	c.Assert(status, Equals, "Starting")

	newStatus := "I am Henry The Eighth I am"
	pd.UpdateStatus(newStatus)
	status, _ = pd.GetStatus()
	c.Assert(status, Equals, newStatus)

	select {
	case _, ok := <-notify:
		// channel was closed
		c.Assert(ok, Equals, false)
	case <-time.After(time.Second):
		c.Fail()
	}
}

func (ft *FacadeUnitTest) Test_PendingDeployment_SetTemplateName(c *C) {
	deploymentID, templateID, poolID := "a", "b", "c"
	templateName := "foobar"
	pd := facade.NewPendingDeployment(deploymentID, templateID, poolID)
	pd.SetTemplateName(templateName)
	info := pd.GetInfo()
	c.Assert(info["templateName"], Equals, templateName)
}

func (ft *FacadeUnitTest) Test_PendingDeploymentMgr_New(c *C) {
	pdm := facade.NewPendingDeploymentMgr()
	deploymentID, templateID, poolID := "a", "b", "c"
	pd, _ := pdm.NewPendingDeployment(deploymentID, templateID, poolID)
	info := pd.GetInfo()
	c.Assert(info["TemplateID"], Equals, templateID)
	c.Assert(info["PoolID"], Equals, poolID)
	c.Assert(info["DeploymentID"], Equals, deploymentID)
}

func (ft *FacadeUnitTest) Test_PendingDeploymentMgr_Conflict(c *C) {
	pdm := facade.NewPendingDeploymentMgr()
	deploymentID, templateID, poolID := "a", "b", "c"
	pd, err := pdm.NewPendingDeployment(deploymentID, templateID, poolID)
	c.Assert(pd, NotNil)
	c.Assert(err, IsNil)
	pd, err = pdm.NewPendingDeployment(deploymentID, templateID, poolID)
	c.Assert(pd, IsNil)
	c.Assert(err, Equals, facade.ErrPendingDeploymentConflict)
}

func (ft *FacadeUnitTest) Test_PendingDeploymentMgr_Get(c *C) {
	pdm := facade.NewPendingDeploymentMgr()
	deploymentID, templateID, poolID := "a", "b", "c"
	pd, _ := pdm.NewPendingDeployment(deploymentID, templateID, poolID)
	pd1 := pdm.GetPendingDeployment(deploymentID)
	c.Assert(pd, Equals, pd1)
}
