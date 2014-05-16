package health

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"sync"
	"time"
)

type healthStatus struct {
	Status    string
	Timestamp int64
	Interval  float64
}

var healthStatuses map[string]map[string]*healthStatus = make(map[string]map[string]*healthStatus)
var exitChannel = make(chan bool)
var lock = &sync.Mutex{}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	w.WriteJson(&healthStatuses)
}

// RegisterHealthCheck updates the healthStatus and healthTime structures with a health check result.
func RegisterHealthCheck(serviceId string, name string, passed string, d dao.ControlPlane) {
	lock.Lock()
	defer lock.Unlock()
	_, ok := healthStatuses[serviceId]
	if !ok {
		healthStatuses[serviceId] = make(map[string]*healthStatus)
		var service service.Service
		err := d.GetService(serviceId, &service)
		if err != nil {
			glog.Errorf("Unable to acquire services.")
			return
		}
		for iname, icheck := range service.HealthChecks {
			_, ok = healthStatuses[serviceId][iname]
			if !ok {
				healthStatuses[serviceId][name] = &healthStatus{"unknown", 0, icheck.Interval.Seconds()}
			}
		}
	}
	healthStatuses[serviceId][name].Status = passed
	healthStatuses[serviceId][name].Timestamp = time.Now().UTC().Unix()
}
