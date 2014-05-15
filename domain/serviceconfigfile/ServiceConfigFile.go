// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package serviceconfigfile

import (
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/utils"
)

type SvcConfigFile struct {
	ID              string
	ServiceTenantID string
	ServicePath     string
	ConfFile        servicedefinition.ConfigFile
}

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
