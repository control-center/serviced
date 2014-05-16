package elasticsearch

import (
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/health"
)

func (this *ControlPlaneDao) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	health.RegisterHealthCheck(result.ServiceId, result.Name, result.Passed, this)
	return nil
}
