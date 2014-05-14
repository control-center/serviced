package health

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"time"
	"sync"
)

var healthStatus map[string]map[string]string = make(map[string]map[string]string)
var healthTime map[string]map[string]time.Time = make(map[string]map[string]time.Time)
var exitChannel = make(chan bool)
var lock = &sync.Mutex{}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	w.WriteJson(&healthStatus)
}

// RegisterHealthCheck updates the healthStatus and healthTime structures with a health check result.
func RegisterHealthCheck(serviceId string, name string, passed string) {
	ensureHealthCheck(serviceId, name)
	lock.Lock()
	healthStatus[serviceId][name] = passed
	healthTime[serviceId][name] = time.Now()
	lock.Unlock()
}

// ensureHealthCheck makes sure that a health check field exists before it is edited.
func ensureHealthCheck(serviceId string, name string) {
	lock.Lock()
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
	lock.Unlock()
}

// Updates the services list and checks for a lack of data from a service.
func StartHealthMonitor(d dao.ControlPlane) {
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case _ = <-ticker:
			var services []*service.Service
			if d.GetServices(new(dao.EntityRequest), &services) != nil {
				glog.Errorf("Unable to acquire services.")
				continue
			}
			for _, service := range services {
				for name, check := range service.HealthChecks {
					ensureHealthCheck(service.Id, name)
					if time.Now().Sub(healthTime[service.Id][name]).Nanoseconds() > check.Interval.Nanoseconds()*2 {
						lock.Lock()
						healthStatus[service.Id][name] = "unknown"
						lock.Unlock()
					}
				}
			}
		case _ = <-exitChannel:
			return
		}
	}
}

// Stops the StartHealthMonitor function.
func StopHealthMonitor() {
	exitChannel<-true;
}