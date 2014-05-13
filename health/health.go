package health


import (
	"github.com/zenoss/serviced/datastore"
	// "github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/glog"
	"time"
)


var healthStatus map[string]map[string]string = make(map[string]map[string]string)


func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *ControlClient) {
	w.WriteJson(&healthStatus)
}


func ensureHealthCheck(serviceId string, name string) {
	_, ok := healthStatus[serviceId]
	if !ok {
		healthStatus[serviceId] = make(map[string]string)
	}
	_, ok = healthStatus[serviceId][name]
	if !ok {
		healthStatus[serviceId][name] = "unknown"
	}
}


func RegisterHealthCheck(serviceId string, name string, passed string) {
	ensureHealthCheck(serviceId, name)
	healthStatus[serviceId][name] = passed
}


func StartHealthMonitor() {
	ctx := datastore.Get()
	ds := service.NewStore()
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case _ = <-ticker:
			services, err := ds.GetServices(ctx)
			if err != nil {
				glog.Errorf("Unable to acquire services.")
				continue
			}
			for _, service := range services {
				for name, _ := range service.HealthChecks {
					ensureHealthCheck(service.Id, name)
				}
			}
		default:
		}
	}
}