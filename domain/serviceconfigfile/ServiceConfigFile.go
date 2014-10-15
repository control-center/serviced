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

package serviceconfigfile

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

//SvcConfigFile is used to store and track service config files that have been modified
type SvcConfigFile struct {
	ID              string
	ServiceTenantID string
	ServicePath     string
	ConfFile        servicedefinition.ConfigFile
	datastore.VersionedEntity
}

//New creates a SvcConfigFile
func New(tenantID string, svcPath string, conf servicedefinition.ConfigFile) (*SvcConfigFile, error) {
	uuid, err := utils.NewUUID()
	if err != nil {
		return nil, err
	}
	svcCF := &SvcConfigFile{ID: uuid, ServiceTenantID: tenantID, ServicePath: svcPath, ConfFile: conf}
	if err = svcCF.ValidEntity(); err != nil {
		return nil, err
	}
	return svcCF, nil
}
