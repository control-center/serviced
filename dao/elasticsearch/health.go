// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/health"
)

func (this *ControlPlaneDao) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	health.RegisterHealthCheck(result.ServiceID, result.Name, result.Passed, this)
	return nil
}
