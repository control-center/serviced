
package elasticsearch

import (
	"github.com/zenoss/serviced/health"
	"github.com/zenoss/serviced/domain"
)

func (this *ControlPlaneDao) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	health.RegisterHealthCheck(result.ServiceId, result.Name, result.Passed)
	return nil
}
