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

package facade

import (
	"errors"
	"sync"

	"github.com/control-center/serviced/utils"
)

var (
	ErrPendingDeploymentConflict = errors.New("template deployment id conflict")
)

// PendingDeployment represents a template deployment which is has been initiated
// but has not yet completed.  It is synchronized, allowing access from multiple
// goroutines.  In particular clients can register for asynchronous notification of
// changes to its "status".
type PendingDeployment struct {
	status       utils.ValueChangePublisher
	templateID   string
	poolID       string
	deploymentID string
	templateName string
	mutex        sync.RWMutex
}

// NewPendingDeployment returns a new PendingDeployment object.
func NewPendingDeployment(deploymentID, templateID, poolID string) PendingDeployment {
	return PendingDeployment{
		status:       utils.NewValueChangePublisher("Starting"),
		templateID:   templateID,
		deploymentID: deploymentID,
		poolID:       poolID,
		mutex:        sync.RWMutex{},
	}
}

// UpdateStatus updates the status of a PendingDeployment.
func (d *PendingDeployment) UpdateStatus(status string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.status.Set(status)
}

// SetTemplateName sets the template name of a PendingDeployment.
func (d *PendingDeployment) SetTemplateName(name string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.templateName = name
}

// GetStatus returns the current status of a PendingDeployment and a
// channel which will be closewhen the status changes.
func (d *PendingDeployment) GetStatus() (string, <-chan struct{}) {
	// mutex lock not necessary; d.status.Get is protected
	status, notify := d.status.Get()
	return status.(string), notify
}

// GetInfo returns all information about a pending deployment in the
// form of a map, appropriate for json serialization.
func (d *PendingDeployment) GetInfo() map[string]string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	status, _ := d.status.Get()
	return map[string]string{
		"TemplateID":   d.templateID,
		"templateName": d.templateName,
		"PoolID":       d.poolID,
		"DeploymentID": d.deploymentID,
		"status":       status.(string),
	}
}

// PendingDeploymentMGr maintains a map from deployment ID to PendingDeployment
// objects.
type PendingDeploymentMgr struct {
	deployments map[string]*PendingDeployment
	mutex       sync.RWMutex
}

// NewPendingDeployment allocates a new PendingDeployment with the given attributes
// and associates it with the PendingDeploymentMgr
func (dm *PendingDeploymentMgr) NewPendingDeployment(deploymentID, templateID, poolID string) (*PendingDeployment, error) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	if _, ok := dm.deployments[deploymentID]; ok {
		return nil, ErrPendingDeploymentConflict
	}
	deployment := NewPendingDeployment(deploymentID, templateID, poolID)
	dm.deployments[deploymentID] = &deployment
	return &deployment, nil
}

// GetPendingDeployment returns the PendingDeployment associated with the
// given deploymentID if one exists; nil otherwise.
func (dm *PendingDeploymentMgr) GetPendingDeployment(deploymentID string) *PendingDeployment {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	return dm.deployments[deploymentID]
}

// DeletePendingDeployment removes the PendingDeployment associated with the
// given deploymentID from the PendingDeploymentMgr.
func (dm *PendingDeploymentMgr) DeletePendingDeployment(deploymentID string) {
	dm.mutex.Lock()
	delete(dm.deployments, deploymentID)
	dm.mutex.Unlock()
}
