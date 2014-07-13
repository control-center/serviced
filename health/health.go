package health

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/node"
	"sync"
	"time"
)

type healthStatus struct {
	Status    string
	Timestamp int64
	Interval  float64
}

type messagePacket struct {
	Timestamp int64
	Statuses  map[string]map[string]*healthStatus
}

var healthStatuses = make(map[string]map[string]*healthStatus)
var exitChannel = make(chan bool)
var lock = &sync.Mutex{}

// RestGetHealthStatus writes a JSON response with the health status of all services that have health checks.
func RestGetHealthStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	packet := messagePacket{time.Now().UTC().Unix(), healthStatuses}
	w.WriteJson(&packet)
}

// RegisterStatusCheck updates the healthStatus and healthTime structures with a health check result.
func RegisterStatusCheck(serviceID string, name string, passed string, d dao.ControlPlane) {
	lock.Lock()
	defer lock.Unlock()

	serviceStatus, ok := healthStatuses[serviceID]
	if !ok {
		// healthStatuses[serviceID]
		serviceStatus = make(map[string]*healthStatus)
		healthStatuses[serviceID] = serviceStatus
	}
	thisStatus, ok := serviceStatus[name]
	if !ok {
		var service service.Service
		err := d.GetService(serviceID, &service)
		if err != nil {
			glog.Errorf("Unable to acquire services.")
			return
		}
		for iname, icheck := range service.StatusChecks {
			_, ok = serviceStatus[iname]
			if !ok {
				serviceStatus[name] = &healthStatus{"unknown", 0, icheck.Interval.Seconds()}
			}
		}
	}
	thisStatus, ok = serviceStatus[name]
	if !ok {
		glog.Warningf("ignoring health status, not found in service: %s %s %s", serviceID, name, passed)
		return
	}
	thisStatus.Status = passed
	thisStatus.Timestamp = time.Now().UTC().Unix()
}
