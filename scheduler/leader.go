package scheduler

import (
	"fmt"
	"time"

	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
	zkservice "github.com/zenoss/serviced/zzk/service"
	"github.com/zenoss/serviced/zzk/snapshot"
	"github.com/zenoss/serviced/zzk/virtualips"
)

type leader struct {
	facade       *facade.Facade
	dao          dao.ControlPlane
	conn         coordclient.Connection
	context      datastore.Context
	poolID       string
	hostRegistry *zkservice.HostRegistryListener
}

// Lead is executed by the "leader" of the control plane cluster to handle its management responsibilities of:
//    services
//    snapshots
//    virtual IPs
func Lead(facade *facade.Facade, dao dao.ControlPlane, conn coordclient.Connection, zkEvent <-chan coordclient.Event, poolID string) {
	glog.V(0).Info("Entering Lead()!")
	defer glog.V(0).Info("Exiting Lead()!")
	shutdownmode := false

	hostRegistry, err := zkservice.NewHostRegistryListener(conn)
	if err != nil {
		glog.Errorf("Could not initialize registry listener for pool %s", poolID)
		return
	}

	leader := leader{facade: facade, dao: dao, conn: conn, context: datastore.Get(), poolID: poolID, hostRegistry: hostRegistry}
	for {
		shutdown := make(chan interface{})
		if shutdownmode {
			glog.V(1).Info("Shutdown mode encountered.")
			close(shutdown)
			break
		}

		time.Sleep(time.Second)
		func() error {
			select {
			case evt := <-zkEvent:
				// shut this thing down
				shutdownmode = true
				glog.V(0).Info("Got a zkevent, leaving lead: ", evt)
				return nil
			default:
				glog.V(0).Info("Processing leader duties")
				// passthru
			}

			snapshotListener := snapshot.NewSnapshotListener(conn, &leader)
			serviceListener := zkservice.NewServiceListener(conn, &leader)
			zzk.Start(shutdown, serviceListener, snapshotListener, hostRegistry)
			return nil
		}()
	}
}

func (l *leader) TakeSnapshot(serviceID string) (string, error) {
	var label string
	err := l.dao.TakeSnapshot(serviceID, &label)
	return label, err
}

// SelectHost chooses a host from the pool for the specified service. If the service
// has an address assignment the host will already be selected. If not the host with the least amount
// of memory committed to running containers will be chosen.
func (l *leader) SelectHost(s *service.Service) (*host.Host, error) {
	var assignmentType string
	var ipAddr string
	var hostid string

	hosts, err := l.hostRegistry.GetHosts()
	if err != nil {
		glog.Errorf("Could not get available hosts for pool %s: %s", l.poolID, err)
		return nil, err
	}

	for _, ep := range s.Endpoints {
		if ep.AddressAssignment != (addressassignment.AddressAssignment{}) {
			assignmentType = ep.AddressAssignment.AssignmentType
			ipAddr = ep.AddressAssignment.IPAddr
			hostid = ep.AddressAssignment.HostID
			break
		}
	}

	if assignmentType == "virtual" {
		// populate hostid
		if err := virtualips.GetVirtualIPHostID(l.conn, ipAddr, &hostid); err != nil {
			return nil, err
		}
		glog.Infof("Service: %v has been assigned virtual IP: %v which has been locked and configured on host %s", s.Name, ipAddr, hostid)
	}

	if hostid != "" {
		return poolHostFromAddressAssignments(hostid, hosts)
	}

	return NewServiceHostPolicy(s, l.dao).SelectHost(hosts)
}

// poolHostFromAddressAssignments determines the pool host for the service from its address assignment(s).
func poolHostFromAddressAssignments(hostid string, hosts []*host.Host) (*host.Host, error) {
	// ensure the assigned host is in the pool
	for _, h := range hosts {
		if h.ID == hostid {
			return h, nil
		}
	}

	return nil, fmt.Errorf("assigned host is not in pool")
}
