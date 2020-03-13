// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons"
	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/scheduler/strategy"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
)

type leader struct {
	shutdown <-chan interface{}
	conn     coordclient.Connection
	cpClient dao.ControlPlane
	facade   *facade.Facade
	poolID   string

	hreg *zkservice.HostRegistryListener
}

// Lead is executed by the "leader" of the control center cluster to handle its management responsibilities of:
//    services
//    snapshots
//    virtual IPs
func Lead(shutdown <-chan interface{}, conn coordclient.Connection, cpClient dao.ControlPlane, facade *facade.Facade, poolID string) {

	// creates a listener for the host registry
	unassignmentHandler := zkservice.NewZKHostUnassignmentHandler(conn)
	hreg := zkservice.NewHostRegistryListener(poolID, unassignmentHandler)

	leader := leader{shutdown, conn, cpClient, facade, poolID, hreg}

	// creates a listener for services
	serviceListener := zkservice.NewServiceListener(poolID, &leader)

	// starts all of the listeners
	zzk.Start(shutdown, conn, serviceListener, hreg)
}

// SelectHost chooses a host from the pool for the specified service. If the
// service has an address assignment the host will already be selected. If not
// the host with the least amount of memory committed to running containers will
// be chosen.  Returns the hostid, hostip (if it has an address assignment).
func (l *leader) SelectHost(sn *zkservice.ServiceNode) (string, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":   sn.ID,
		"servicename": sn.Name,
	})

	plog.Debug("Looking for available hosts in resource pool")
	reghosts, err := l.hreg.GetRegisteredHosts(l.shutdown)
	if err != nil {
		plog.WithError(err).Debug("Could not get available hosts from resource pool")
		return "", err
	}

	// if no hosts are returned, then a shutdown has been triggered
	if len(reghosts) == 0 {
		plog.Debug("Scheduler is shutting down")
		return "", errors.New("scheduler is shutting down")
	}

	// filter out hosts that have not been authenticated
	hosts := []host.Host{}
	for _, h := range reghosts {
		hlogger := logger.WithField("hostid", h.ID)
		isAuthenticated, err := l.facade.HostIsAuthenticated(datastore.GetContext(), h.ID)
		if err != nil {
			hlogger.WithError(err).Debug("Unable to check if host is authenticated")
		} else if !isAuthenticated {
			hlogger.Debug("Host not authenticated")
		} else {
			hosts = append(hosts, h)
		}
	}

	//  Are there any hosts left?
	if len(hosts) == 0 {
		return "", ErrNoAuthenticatedHosts
	}

	assignment := sn.AddressAssignment
	if sn.ShouldHaveAddressAssignment && assignment.IPAddr == "" {
		plog.WithField("endpoint", sn.Name).Debug("Service is missing an address assignment")
		return "", errors.New("service is missing an address assignment")
	}

	if assignment.IPAddr != "" {
		logger = logger.WithFields(log.Fields{
			"ipaddress":      assignment.IPAddr,
			"assignmenttype": assignment.AssignmentType,
		})
		logger.Debug("Checking host availability for service address assignment")

		// find which host the address belongs to
		hostID := assignment.HostID
		if assignment.AssignmentType == commons.VIRTUAL {
			hostID, err = zkservice.GetHostID(l.conn, l.poolID, assignment.IPAddr)
			if err != nil {
				logger.WithError(err).Debug("Could not get host assignment of virtual ip")
				return "", err
			}

			request := zkservice.IPRequest{
				IPAddress: assignment.IPAddr,
				HostID:    hostID,
				PoolID:    l.poolID,
			}

			ip, err := zkservice.GetIP(l.conn, request)
			if err != nil {
				return "", errors.New("could not check if virtual ip is bound")
			} else if !ip.OK {
				return "", errors.New("assigned ip is not bound")
			}
		}

		// is the host available?
		for _, h := range hosts {
			if h.ID == hostID {
				return hostID, nil
			}
		}

		logger.WithField("hostid", hostID).Warn("Could not assign service to ip address.  Check to see if host is running and registered or reassign ips")
		return "", errors.New("assigned ip is not available")
	}

	hp := sn.HostPolicy
	strat, err := strategy.Get(string(hp))
	if err != nil {
		return "", err
	}

	return StrategySelectHost(sn, hosts, strat, l.facade)
}
