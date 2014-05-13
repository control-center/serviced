package serviced


import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/glog"
	"time"
)


var healthStatus map[string]map[string]string = make(map[string]map[string]string)
var healthTime map[string]map[string]time.Time = make(map[string]map[string]time.Time)


func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *ControlClient) {
	w.WriteJson(&healthStatus)
}


func RegisterHealthCheck(serviceId string, name string, passed string) {
	ensureHealthCheck(serviceId, name)
	healthStatus[serviceId][name] = passed
	healthTime[serviceId][name] = time.Now()
}


func ensureHealthCheck(serviceId string, name string) {
	_, ok := healthStatus[serviceId]
	if !ok {
		healthStatus[serviceId] = make(map[string]string)
		healthTime[serviceId] = make(map[string]time.Time)
	}
	_, ok = healthStatus[serviceId][name]
	if !ok {
		healthStatus[serviceId][name] = "unknown"
		healthTime[serviceId][name] = time.Now()
	}
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
				for name, check := range service.HealthChecks {
					ensureHealthCheck(service.Id, name)
					if time.Now().Sub(healthTime[service.Id][name]).Nanoseconds() > check.Interval.Nanoseconds() * 2 {
						healthStatus[service.Id][name] = "late"
					}
				}
			}
		default:
		}
	}
}