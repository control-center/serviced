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

type ServiceRegistrySyncRequest struct {
	ServiceID       string
	PortsToDelete   []PublicPortKey
	PortsToPublish  map[PublicPortKey]PublicPort
	VHostsToDelete  []VHostKey
	VHostsToPublish map[VHostKey]VHost
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

// GetPublicPort returns the service id and application of the public port
func GetPublicPort(conn client.Connection, key PublicPortKey) (string, string, error) {
	pth := path.Join("/net/pub", key.HostID, key.PortAddress)

	logger := plog.WithFields(log.Fields{
		"hostid":      key.HostID,
		"portaddress": key.PortAddress,
		"zkpath":      pth,
	})

	pub := &PublicPort{}
	err := conn.Get(pth, pub)
	if err == client.ErrNoNode {
		logger.WithError(err).Debug("Port address not found")
		return "", "", nil
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up port")
		return "", "", &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "could not look up port address",
		}
	}

	logger.WithFields(log.Fields{
		"tenantid":    pub.TenantID,
		"application": pub.Application,
	}).Debug("Found port")

	return pub.ServiceID, pub.Application, nil
}

func GetPublicPorts(conn client.Connection) (map[PublicPortKey]PublicPort, error) {
	ports := make(map[PublicPortKey]PublicPort, 0)

	pth := "/net/pub"
	logger := plog.WithField("zkpath", pth)

	hostIDs, err := conn.Children(pth)
	if err == client.ErrNoNode {
		conn.CreateDir(pth)
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up public port path")
		return nil, &RegistryError{
			Action:  "get",
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
			return nil, &RegistryError{
				Action:  "get",
				Path:    hostpth,
				Message: "could not look up public ports for host id",
			}
		}

		for _, portAddr := range portAddrs {
			addrpth := path.Join(hostpth, portAddr)
			addrLogger := hostLogger.WithFields(log.Fields{
				"portaddress": portAddr,
				"zkpath":    addrpth,
			})

			pub := &PublicPort{}
			if err := conn.Get(addrpth, pub); err == client.ErrNoNode {
				addrLogger.Debug("Port address has been deleted for public port")
				continue
			} else if err != nil {
				addrLogger.WithError(err).Debug("could not look up public port")
				return nil, &RegistryError{
					Action:  "get",
					Path:    addrpth,
					Message: "could not look up public port address",
				}
			}

			key := PublicPortKey{HostID: hostID, PortAddress: portAddr}
			ports[key] = *pub
		}
	}

	return ports, nil
}

// GetVHost returns the service id and application of the vhost
func GetVHost(conn client.Connection, key VHostKey) (string, string, error) {
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
		return "", "", nil
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up virtual host subdomain")
		return "", "", &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "could not look up virtual host subdomain",
		}
	}

	logger.WithFields(log.Fields{
		"tenantid":    vhost.TenantID,
		"application": vhost.Application,
	}).Debug("Found vhost subdomain")

	return vhost.ServiceID, vhost.Application, nil
}

func GetVHosts(conn client.Connection) (map[VHostKey]VHost, error) {
	vhosts := make(map[VHostKey]VHost, 0)

	pth := "/net/vhost"
	logger := plog.WithField("zkpath", pth)

	hostIDs, err := conn.Children(pth)
	if err == client.ErrNoNode {
		conn.CreateDir(pth)
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up virtual hosts path")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "could not look up virtual hosts path",
		}
	}

	// get all the public ports for each host
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
			return nil, &RegistryError{
				Action:  "get",
				Path:    hostpth,
				Message: "could not look up virtual hosts for host id",
			}
		}

		for _, subdomain := range subdomains {
			addrpth := path.Join(hostpth, subdomain)
			addrLogger := hostLogger.WithFields(log.Fields{
				"subdomain": subdomain,
				"zkpath":    addrpth,
			})

			vhost := &VHost{}
			if err := conn.Get(addrpth, vhost); err == client.ErrNoNode {
				addrLogger.Debug("Port address has been deleted for virtual host")
				continue
			} else if err != nil {
				addrLogger.WithError(err).Debug("could not look up virtaul host")
				return nil, &RegistryError{
					Action:  "get",
					Path:    addrpth,
					Message: "could not look up virtual host subdomaion",
				}
			}

			key := VHostKey{HostID: hostID, Subdomain: subdomain}
			vhosts[key] = *vhost
		}
	}

	return vhosts, nil
}

// SyncServiceRegistry syncs all vhosts and public ports to those of a matching
// service.
func SyncServiceRegistry(conn client.Connection, request ServiceRegistrySyncRequest) error {
	logger := plog.WithField("serviceid", request.ServiceID)

	if len(request.PortsToDelete) == 0 &&
	   len(request.PortsToPublish) == 0 &&
	   len(request.VHostsToDelete) == 0 &&
	   len(request.VHostsToPublish) == 0 {
		// Don't even create a transaction if there's nothing to do (which is typical for many kinds of services)
		return nil
	}

	tx := conn.NewTransaction()

	if err := syncServicePublicPorts(conn, tx, request); err != nil {
		logger.WithError(err).Debug("Could not sync public ports for service")
		return err
	}
	if err := syncServiceVHosts(conn, tx, request); err != nil {
		logger.WithError(err).Debug("Could not sync virtual hosts for service")
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
func syncServicePublicPorts(conn client.Connection, tx client.Transaction, request ServiceRegistrySyncRequest) error {
	logger := plog.WithField("serviceid", request.ServiceID)
	pth := "/net/pub"

	for _, pubKey := range request.PortsToDelete {
		addrpth := path.Join(pth, pubKey.HostID, pubKey.PortAddress)
		addrLogger := logger.WithFields(log.Fields{
			"hostid":      pubKey.HostID,
			"portaddress": pubKey.PortAddress,
			"zkpath":      addrpth,
		})
		tx.Delete(addrpth)
		addrLogger.Debug("Deleted public port address")
	}

	for pubKey, pubValue := range request.PortsToPublish {
		addrpth := path.Join(pth, pubKey.HostID, pubKey.PortAddress)
		addrLogger := logger.WithFields(log.Fields{
			"hostid":      pubKey.HostID,
			"portaddress": pubKey.PortAddress,
			"zkpath":      addrpth,
		})
		err := conn.CreateIfExists(addrpth, &pubValue)
		if err == client.ErrNoNode {
			if err := conn.CreateDir(path.Join(pth, pubKey.HostID)); err != nil {
				return &RegistryError{
					Action:  "sync",
					Path:    path.Join(pth, pubKey.HostID),
					Message: "could not register public port address",
				}
			}
			pubValue.SetVersion(nil)
			tx.Create(addrpth, &pubValue)
			addrLogger.Debug("Created public port address")
		} else if err == client.ErrNodeExists {
			existingPub := &PublicPort{}
			if err := conn.Get(addrpth, existingPub); err != nil {
				return &RegistryError{
					Action:  "sync",
					Path:    addrpth,
					Message: "could not read current public port address",
				}
			}
			pubValue.SetVersion(existingPub.Version())
			tx.Set(addrpth, &pubValue)
			addrLogger.Debug("Updated public port address")
		} else if err != nil {
			addrLogger.WithError(err).Debug("skipped public port address because of an unexpected error")
			return &RegistryError{
				Action:  "sync",
				Path:   addrpth,
				Message: "could not register public port address",
			}
		}
	}

	logger.Debug("Updated transaction to sync public ports for service")
	return nil
}

// syncServiceVHosts updates the transaction to include virtual host updates
func syncServiceVHosts(conn client.Connection, tx client.Transaction, request ServiceRegistrySyncRequest) error {
	logger := plog.WithField("serviceid", request.ServiceID)
	pth := "/net/vhost"

	for _, vhostKey := range request.VHostsToDelete {
		addrpth := path.Join(pth, vhostKey.HostID, vhostKey.Subdomain)
		addrLogger := logger.WithFields(log.Fields{
			"hostid":    vhostKey.HostID,
			"subdomain": vhostKey.Subdomain,
			"zkpath":    addrpth,
		})
		tx.Delete(addrpth)
		addrLogger.Debug("Deleted virtual host")
	}

	for vhostKey, vhost := range request.VHostsToPublish {
		addrpth := path.Join(pth, vhostKey.HostID, vhostKey.Subdomain)
		addrLogger := logger.WithFields(log.Fields{
			"hostid":    vhostKey.HostID,
			"subdomain": vhostKey.Subdomain,
			"zkpath":    addrpth,
		})
		err := conn.CreateIfExists(addrpth, &vhost)
		if err == client.ErrNoNode {
			if err := conn.CreateDir(path.Join(pth, vhostKey.HostID)); err != nil {
				return &RegistryError{
					Action:  "sync",
					Path:    path.Join(pth, vhostKey.HostID),
					Message: "could not register virtual host",
				}
			}
			vhost.SetVersion(nil)
			tx.Create(addrpth, &vhost)
			addrLogger.Debug("Created public port address")
		} else if err == client.ErrNodeExists {
			existingVHost := &VHost{}
			if err := conn.Get(addrpth, existingVHost); err != nil {
				return &RegistryError{
					Action:  "sync",
					Path:    addrpth,
					Message: "could not read current virtual host",
				}
			}
			vhost.SetVersion(existingVHost.Version())
			tx.Set(addrpth, &vhost)
			addrLogger.Debug("Updated virtual host")
		} else if err != nil {
			addrLogger.WithError(err).Debug("skipped virtual host because of an unexpected error")
			return &RegistryError{
				Action:  "sync",
				Path:   addrpth,
				Message: "could not register public port address",
			}
		}
	}

	logger.Debug("Updated transaction to sync virtual hosts for service")
	return nil
}
