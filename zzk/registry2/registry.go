// Copyright 2016 The Serviced Authors.
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

package registry

import (
	"fmt"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/logging"
)

// initialize the package logger
var plog = logging.PackageLogger()

// RegistryError describes an error with a registry lookup
type RegistryError struct {
	Action  string
	Path    string
	Message string
}

func (err RegistryError) Error() string {
	return fmt.Sprintf("could not %s path %s: %s", err.Action, err.Path, err.Message)
}

// PublicPortKey points to a specific public port node
type PublicPortKey struct {
	HostID      string
	PortAddress string
}

// VHostKey points to a specific vhost node
type VHostKey struct {
	HostID    string
	Subdomain string
}

// DeleteExports deletes all export data for a tenant id
func DeleteExports(conn client.Connection, tenantID string) error {
	pth := path.Join("/net/export", tenantID)
	logger := plog.WithFields(log.Fields{
		"tenantid": tenantID,
		"zkpath":   pth,
	})

	if err := conn.Delete(pth); err == client.ErrNoNode {
		logger.Debug("No exports for tenant id")
		return nil
	} else if err != nil {
		logger.WithError(err).Debug("Could not delete exports for tenant id")
		return err
	}

	logger.Debug("Removed exports for tenant id")
	return nil
}

// GetPublicPort returns port data for a specific port address on a host
func GetPublicPort(conn client.Connection, key PublicPortKey) (*PublicPort, error) {
	pth := path.Join("/net/pub", key.HostID, key.PortAddress)

	logger := plog.WithFields(log.Fields{
		"hostid":      key.HostID,
		"portaddress": key.PortAddress,
		"zkpath":      pth,
	})

	pub := &PublicPort{}
	err := conn.Get(pth, pub)
	if err == client.ErrNoNode {
		logger.WithError(err).Debug("Port not found on host")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "port address not found on host",
		}
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up port")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "could not look up port address",
		}
	}

	logger.WithFields(log.Fields{
		"tenantid":    pub.TenantID,
		"application": pub.Application,
	}).Debug("Found port")

	return pub, nil
}

// GetVHost returns port data for a specific port address on a host
func GetVHost(conn client.Connection, key VHostKey) (*VHost, error) {
	pth := path.Join("/net/vhost", key.HostID, key.Subdomain)

	logger := plog.WithFields(log.Fields{
		"hostid":    key.HostID,
		"subdomain": key.Subdomain,
		"zkpath":    pth,
	})

	vhost := &VHost{}
	err := conn.Get(pth, vhost)
	if err == client.ErrNoNode {
		logger.WithError(err).Debug("Virtual host subdomain not found")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "virtual host subdomain not found",
		}
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up virtual host subdomain")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "could not look up virtual host subdomain",
		}
	}

	logger.WithFields(log.Fields{
		"tenantid":    vhost.TenantID,
		"application": vhost.Application,
	}).Debug("Found vhost subdomain")

	return vhost, nil
}

// SyncServiceRegistry syncs all vhosts and public ports to those of a matching
// service.
// FIXME: need to optimize
func SyncServiceRegistry(conn client.Connection, serviceID string, pubs map[PublicPortKey]PublicPort, vhosts map[VHostKey]VHost) error {
	logger := plog.WithField("serviceid", serviceID)

	tx := conn.NewTransaction()

	if err := syncServicePublicPorts(conn, tx, serviceID, pubs); err != nil {
		return err
	}

	if err := syncServiceVHosts(conn, tx, serviceID, vhosts); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.WithError(err).Debug("Could not sync registry for service")

		// TODO: wrap error?
		return err
	}

	logger.Debug("Updated registry for service")
	return nil
}

// syncServicePublicPorts updates the transaction to include public port updates
func syncServicePublicPorts(conn client.Connection, tx client.Transaction, serviceID string, pubs map[PublicPortKey]PublicPort) error {
	logger := plog.WithField("serviceid", serviceID)

	// pull all the hosts of all the public ports
	pth := "/net/pub"
	logger = logger.WithField("zkpath", pth)

	hostIDs, err := conn.Children(pth)
	if err == client.ErrNoNode {
		conn.CreateDir(pth)
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up public ports path")
		return &RegistryError{
			Action:  "sync",
			Path:    pth,
			Message: "could not look up public ports path",
		}
	}

	// get all the public ports for each host
	for _, hostID := range hostIDs {
		hostpth := path.Join(pth, hostID)
		hostLogger := logger.WithFields(log.Fields{
			"hostid": hostID,
			"zkpath": hostpth,
		})

		portAddrs, err := conn.Children(hostpth)
		if err == client.ErrNoNode {
			hostLogger.Debug("Host has been deleted for public ports")
		} else if err != nil {
			hostLogger.WithError(err).Debug("Could not look up public ports for host id")
			return &RegistryError{
				Action:  "sync",
				Path:    hostpth,
				Message: "could not look up public ports for host id",
			}
		}

		for _, portAddr := range portAddrs {
			key := PublicPortKey{HostID: hostID, PortAddress: portAddr}
			addrpth := path.Join(hostpth, portAddr)
			addrLogger := hostLogger.WithFields(log.Fields{
				"portaddress": portAddr,
				"zkpath":      addrpth,
			})

			pub := &PublicPort{}
			if err := conn.Get(addrpth, pub); err == client.ErrNoNode {
				addrLogger.Debug("Port address has been deleted for public port")
				continue
			} else if err != nil {
				addrLogger.WithError(err).Debug("could not look up public port address")
				return &RegistryError{
					Action:  "sync",
					Path:    addrpth,
					Message: "could not look up public port address",
				}
			}

			// update the public address if there is a key reference, otherwise
			// delete it if the service matches.
			if val, ok := pubs[key]; ok {
				addrLogger.Debug("Updating public port address")
				val.SetVersion(pub.Version())
				tx.Set(addrpth, &val)
				delete(pubs, key)
			} else if pub.ServiceID == serviceID {
				addrLogger.Debug("Deleting public port address")
				tx.Delete(addrpth)
			}
		}
	}

	// create the remaining public ports
	for key, val := range pubs {
		conn.CreateDir(path.Join(pth, key.HostID))
		addrpth := path.Join(pth, key.HostID, key.PortAddress)
		val.SetVersion(nil)
		logger.WithFields(log.Fields{
			"hostid":      key.HostID,
			"portaddress": key.PortAddress,
			"zkpath":      addrpth,
		}).Debug("Creating public port address")
		tx.Create(addrpth, &val)
	}

	logger.Debug("Updated transaction to sync public ports for service")
	return nil
}

// syncServiceVHosts updates the transaction to include virtual host updates
func syncServiceVHosts(conn client.Connection, tx client.Transaction, serviceID string, vhosts map[VHostKey]VHost) error {
	logger := plog.WithField("serviceid", serviceID)

	// pull all the hosts of all the virtual hosts
	pth := "/net/vhost"
	logger = logger.WithField("zkpath", pth)

	hostIDs, err := conn.Children(pth)
	if err == client.ErrNoNode {
		conn.CreateDir(pth)
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up virtual hosts path")
		return &RegistryError{
			Action:  "sync",
			Path:    pth,
			Message: "could not look up virtual hosts path",
		}
	}

	// get all the virtual host subdomains for each host
	for _, hostID := range hostIDs {
		hostpth := path.Join(pth, hostID)
		hostLogger := logger.WithFields(log.Fields{
			"hostid": hostID,
			"zkpath": hostpth,
		})

		subdomains, err := conn.Children(hostpth)
		if err == client.ErrNoNode {
			hostLogger.Debug("Host has been deleted for virtual hosts")
		} else if err != nil {
			hostLogger.WithError(err).Debug("Could not look up virtual hosts for host id")
			return &RegistryError{
				Action:  "sync",
				Path:    hostpth,
				Message: "could not look up virtual hosts for host id",
			}
		}

		for _, subdomain := range subdomains {
			key := VHostKey{HostID: hostID, Subdomain: subdomain}
			addrpth := path.Join(hostpth, subdomain)
			addrLogger := hostLogger.WithFields(log.Fields{
				"subdomain": subdomain,
				"zkpath":    addrpth,
			})

			vhost := &VHost{}
			if err := conn.Get(addrpth, vhost); err == client.ErrNoNode {
				addrLogger.Debug("Subdomain has been deleted for virtual host")
				continue
			} else if err != nil {
				addrLogger.WithError(err).Debug("could not look up virtual host subdomain")
				return &RegistryError{
					Action:  "sync",
					Path:    addrpth,
					Message: "could not look up virtual host subdomain",
				}
			}

			// update the public address if there is a key reference, otherwise
			// delete it if the service matches.
			if val, ok := vhosts[key]; ok {
				addrLogger.Debug("Updating virtual host subdomain")
				val.SetVersion(vhost.Version())
				tx.Set(addrpth, &val)
				delete(vhosts, key)
			} else if vhost.ServiceID == serviceID {
				addrLogger.Debug("Deleting virtual host subdomain")
				tx.Delete(addrpth)
			}
		}
	}

	// create the remaining public ports
	for key, val := range vhosts {
		conn.CreateDir(path.Join(pth, key.HostID))
		addrpth := path.Join(pth, key.HostID, key.Subdomain)
		val.SetVersion(nil)
		logger.WithFields(log.Fields{
			"hostid":    key.HostID,
			"subdomain": key.Subdomain,
			"zkpath":    addrpth,
		}).Debug("Creating virtual address subdomain")
		tx.Create(addrpth, &val)
	}

	logger.Debug("Updated transaction to sync virtual hosts for service")
	return nil
}
