package servicestate

import (
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/servicestate"
)

const (
	zkService = "/service"
)

func servicePath(nodes ...string) string {
	p := []string{zkService}
	p = append(p, nodes...)
	return path.Join(p...)
}

func LoadServiceState(conn client.Connection, state *servicestate.ServiceState, serviceID string, ssID string) error {
	node := servicePath(serviceID, ssID)
	if err := conn.Get(node, state); err != nil {
		glog.Errorf("Could not load service state at path %s: %s", node, err)
		return err
	}
	return nil
}