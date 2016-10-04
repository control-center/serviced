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
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/logging"
)

var (
	// initialize the package logger
	plog = logging.PackageLogger()

	cacheLock = sync.Mutex{}

	publicPortCache map[string][]PublicPortCache
	vhostCache map[string][]VHostCache
)

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

// PublicPortCache holds cached information used to optimize the registry sync process
type PublicPortCache struct {
	Key       PublicPortKey
	Value     PublicPort
	Path      string
	ServiceID string
}

// VHostCache holds cached information used to optimize the registry sync process
type VHostCache struct {
	Key       VHostKey
	Value     VHost
	Path      string
	ServiceID string
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

// SyncServiceRegistry syncs all vhosts and public ports to those of a matching servuce.
func SyncServiceRegistry(conn client.Connection, serviceID string, pubs map[PublicPortKey]PublicPort, vhosts map[VHostKey]VHost) error {
	logger := plog.WithField("serviceid", serviceID)

	cacheLock.Lock()
	defer cacheLock.Unlock()
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
	pth := "/net/pub"
	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
		"zkpath":    pth,
	})

	if publicPortCache == nil {
		if err := buildPublicPortCache(conn, tx); err != nil {
			return err
		}
	}

	zkPaths, ok := publicPortCache[serviceID]
	if ok {
		i := 0
		for _, cachedEntry := range zkPaths {
			hostpth := path.Join(pth, cachedEntry.Key.HostID)
			addrpth := path.Join(hostpth, cachedEntry.Key.PortAddress)
			logger = logger.WithFields(log.Fields{
				"hostid":    cachedEntry.Key.HostID,
				"portaddress": cachedEntry.Key.PortAddress,
				"zkpath":    addrpth,
			})

			pub, ok := pubs[cachedEntry.Key]
			if ok {
				// Update ZK (and the cache) with the specified entry
				conn.CreateDir(hostpth)
				value := &PublicPort{}
				if err := conn.Get(addrpth, value); err == client.ErrNoNode {
					logger.Debug("Port address has been deleted for public port")
					continue
				} else if err != nil {
					logger.WithError(err).Debug("could not look up public port address")
					return &RegistryError{
						Action:  "sync",
						Path:    addrpth,
						Message: "could not look up public port address",
					}
				}

				pub.SetVersion(value.Version())
				tx.Set(addrpth, &pub)
				cachedEntry.Value = pub
				zkPaths[i] =  cachedEntry
				i++
				logger.Debug("Updated cache entry")

				// delete this value from the public ports map
				delete(pubs, cachedEntry.Key)
			} else {
				// Delete the cached entry from ZK (and the cache)
				tx.Delete(addrpth)
				logger.Debug("Deleted cache entry")
			}
			zkPaths = zkPaths[:i]
		}
	}

	// create a new entry for each public end point that's not already in the cache.
	for key := range pubs {
		val := pubs[key]
		conn.CreateDir(path.Join(pth, key.HostID))
		addrpth := path.Join(pth, key.HostID, key.PortAddress)
		val.SetVersion(nil)
		logger.WithFields(log.Fields{
			"hostid":      key.HostID,
			"portaddress": key.PortAddress,
			"zkpath":      addrpth,
		}).Debug("Creating public port address")
		tx.Create(addrpth, &val)

		entry := PublicPortCache{
			Key:       key,
			Path:      addrpth,
			ServiceID: serviceID,
		}
		publicPortCache[entry.ServiceID] = append(publicPortCache[entry.ServiceID], entry)
	}

	logger.Debug("Updated transaction to sync public ports for service")
	return nil
}

// syncServiceVHosts updates the transaction to include virtual host updates
func syncServiceVHosts(conn client.Connection, tx client.Transaction, serviceID string, vhosts map[VHostKey]VHost) error {

	pth := "/net/vhost"
	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
		"zkpath":    pth,
	})

	if vhostCache == nil {
		if err := buildVhostCache(conn, tx); err != nil {
			return err
		}
	}

	zkPaths, ok := vhostCache[serviceID]
	if ok {
		i := 0
		for _, cachedEntry := range zkPaths {
			hostpth := path.Join(pth, cachedEntry.Key.HostID)
			addrpth := path.Join(hostpth, cachedEntry.Key.Subdomain)
			logger = logger.WithFields(log.Fields{
				"hostid":    cachedEntry.Key.HostID,
				"subdomain": cachedEntry.Key.Subdomain,
				"zkpath":    addrpth,
			})

			vhost, ok := vhosts[cachedEntry.Key]
			if ok {
				// Update ZK (and the cache) with the specified entry
				conn.CreateDir(hostpth)
				value := &VHost{}
				if err := conn.Get(addrpth, value); err == client.ErrNoNode {
					logger.Debug("Subdomain has been deleted for virtual host")
					continue
				} else if err != nil {
					logger.WithError(err).Debug("could not look up virtual host subdomain")
					return &RegistryError{
						Action:  "sync",
						Path:    addrpth,
						Message: "could not look up virtual host subdomain",
					}
				}

				vhost.SetVersion(value.Version())
				tx.Set(addrpth, &vhost)
				cachedEntry.Value = vhost
				zkPaths[i] =  cachedEntry
				i++
				logger.Debug("Updated cache entry")

				// delete this value from the vhosts map
				delete(vhosts, cachedEntry.Key)
			} else {
				// Delete the cached entry from ZK (and the cache)
				tx.Delete(addrpth)
				logger.Debug("Deleted cache entry")
			}
			zkPaths = zkPaths[:i]
		}
	}

	// create a new entry for each vhost that's not already in the cache.
	for key := range vhosts {
		val := vhosts[key]
		conn.CreateDir(path.Join(pth, key.HostID))
		addrpth := path.Join(pth, key.HostID, key.Subdomain)
		val.SetVersion(nil)
		logger.WithFields(log.Fields{
			"hostid":    key.HostID,
			"subdomain": key.Subdomain,
			"zkpath":    addrpth,
		}).Debug("Creating virtual address subdomain")
		tx.Create(addrpth, &val)

		entry := VHostCache{
			Key:       key,
			Path:      addrpth,
			ServiceID: serviceID,
		}
		vhostCache[entry.ServiceID] = append(vhostCache[entry.ServiceID], entry)
	}

	logger.Debug("Updated transaction to sync virtual hosts for service")
	return nil
}

// buildVhostCache creates a new instance of vhostCache based soley on the data in zookeeper
func buildVhostCache(conn client.Connection, tx client.Transaction) error {
	vhostCache = make(map[string][]VHostCache, 0)

	// pull all the hosts of all the virtual hosts
	pth := "/net/vhost"
	logger := plog.WithField("zkpath", pth)

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

			key := VHostKey{HostID: hostID, Subdomain: subdomain}
			entry := VHostCache{
				Key:       key,
				Path:      addrpth,
				ServiceID: vhost.ServiceID,
			}
			vhostCache[entry.ServiceID] = append(vhostCache[entry.ServiceID], entry)
			addrLogger.Debug("Creating virtual address subdomain")
		}
	}

	return nil
}

// buildPublicPortCache creates a new instance of publicPortCache based soley on the data in zookeeper
func buildPublicPortCache(conn client.Connection, tx client.Transaction) error {
	publicPortCache = make(map[string][]PublicPortCache, 0)

	// pull all the hosts of all the virtual hosts
	pth := "/net/pub"
	logger := plog.WithField("zkpath", pth)

	hostIDs, err := conn.Children(pth)
	if err == client.ErrNoNode {
		conn.CreateDir(pth)
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up public port path")
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
				Message: "could not look up public portsfor host id",
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
				addrLogger.WithError(err).Debug("could not look up p")
				return &RegistryError{
					Action:  "sync",
					Path:    addrpth,
					Message: "could not look up public port address",
				}
			}

			key := PublicPortKey{HostID: hostID, PortAddress: portAddr}
			entry := PublicPortCache{
				Key:       key,
				Path:      addrpth,
				ServiceID: pub.ServiceID,
			}
			publicPortCache[entry.ServiceID] = append(publicPortCache[entry.ServiceID], entry)
			addrLogger.Debug("Creating public port address")
		}
	}

	return nil
}
