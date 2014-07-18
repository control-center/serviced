// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package serviceconfigfile

import (
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

//SvcConfigFile is used to store and track service config files that have been modified
type SvcConfigFile struct {
	ID              string
	ServiceTenantID string
	ServicePath     string
	ConfFile        servicedefinition.ConfigFile
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
