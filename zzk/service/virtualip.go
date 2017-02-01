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

package service

import (
	"fmt"
	"path"
	"strings"

	"github.com/control-center/serviced/coordinator/client"
)

// PoolIP provides information about the virtual ip with respect to its state
// on the host.
type PoolIP struct {
	version interface{}
}

// Version implements client.Node
func (ip *PoolIP) Version() interface{} {
	return ip.version
}

// SetVersion implements client.Node
func (ip *PoolIP) SetVersion(version interface{}) {
	ip.version = version
}

// HostIP describes to the delegate how to set up the virtual ip
type HostIP struct {
	version interface{}
}

// Version implements client.Node
func (ip *HostIP) Version() interface{} {
	return ip.version
}

// SetVersion implements client.Node
func (ip *HostIP) SetVersion(version interface{}) {
	ip.version = version
}

// IP is the concatenation of the PoolIP and HostIP objects
type IP struct {
	PoolIP
	HostIP
	HostID    string
	IPAddress string
}

// IPRequest provides information for ip CRUD
type IPRequest struct {
	PoolID    string
	HostID    string
	IPAddress string
}

// IPID is the string identifier of the node in zookeeper
func (req IPRequest) IPID() string {
	return fmt.Sprintf("%s-%s", req.HostID, req.IPAddress)
}

// ParseIPID returns the host and ip address from a given IP id
func ParseIPID(ipid string) (hostid string, ipaddress string, err error) {
	parts := strings.SplitN(ipid, "-", 2)
	if len(parts) != 2 {
		// TODO: error
		return "", "", ErrInvalidIPID
	}
	return parts[0], parts[1], nil
}

// GetIP returns the ip data for a given virtual ip
func GetIP(conn client.Connection, req IPRequest) (*IP, error) {
	logger := plog.WithFields(log.Fields{
		"hostid":    req.HostID,
		"ipaddress": req.IPAddress,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current pool ip
	ppth := path.Join(basepth, "/ips", req.IPID())
	pdat := &PoolIP{}
	if err := conn.Get(ppth, pdat); err != nil {
		logger.WithError(err).Debug("Could not look up virtual ip on resource pool")
		// TODO: error
		return nil, err
	}
	logger.Debug("Found pool ip")

	// Get the current host ip
	hpth := path.Join(basepth, "/hosts", req.HostID, "ips", req.IPID())
	hdat := &HostIP{}
	if err := conn.Get(hpth, hdat); err != nil {
		logger.WithError(err).Debug("Could not look up virtual ip on host")
		// TODO: error
		return nil, err
	}
	logger.Debug("Found host ip")

	return &IP{
		PoolIP:    *pdat,
		HostIP:    *hdat,
		HostID:    req.HostID,
		IPAddress: req.IPAddress,
	}, nil
}

// CreateIP adds a new ip for pool and the host
func CreateIP(conn client.Connection, req IPRequest) error {
	logger := plog.WithFields(log.Fields{
		"hostid": req.HostID,
		"ipaddress", req.IPAddress,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Prepare the pool ip
	ppth := path.Join(basepth, "/ips")
	err := conn.CreateIfExists(ppth, &client.Dir{})
	if err != nil && err != client.ErrNodeExists {
		logger.WithError(err).Debug("Could not initialize pool ip path")
		// TODO: error
		return err
	}

	ppth = path.Join(ppth, req.IPID())
	pdat := &PoolIP{}
	t.Create(ppth, pdat)

	// Prepare the host ip
	hpth := path.Join(basepth, "/hosts", req.HostID, "/ips")
	err = conn.CreateIfExists(hpth, &client.Dir{})
	if err != nil && err != client.ErrNodeExists {
		logger.WithError(err).Debug("Could not initialize host ip path")
		// TODO: error
		return err
	}

	hpth = path.Join(hpth, req.IPID())
	hdat := &HostIP{}
	t.Create(hpth, hdat)

	if err := t.Commit(); err != nil {
		logger.WithError(err).Debug("Could not commit transaction")
		// TODO: err
		return err
	}
	logger.Debug("Created ip")
	return nil
}

// UpdateIP updates the ip for the pool and the host
func UpdateIP(conn client.Connection, req IPRequest, mutate func(*IP) bool) error {
	logger := plog.WithFields(log.Fields{
		"hostid": req.HostID,
		"ipaddress", req.IPAddress,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	// Get the current pool ip
	ppth := path.Join(basepth, "/ips", req.IPID())
	pdat := &PoolIP{}
	if err := conn.Get(ppth, pdat); err != nil {
		logger.WithError(err).Debug("Could not look up virtual ip on resource pool")
		// TODO: error
		return nil, err
	}

	// Get the current host ip
	hpth := path.Join(basepth, "/hosts", req.HostID, "ips", req.IPID())
	hdat := &HostIP{}
	if err := conn.Get(hpth, hdat); err != nil {
		logger.WithError(err).Debug("Could not look up virtual ip on host")
		// TODO: error
		return nil, err
	}

	// mutate the state
	pver, hver := pdat.Version(), hdat.Version()
	ip := &IP{
		PoolIP:    *pdat,
		HostIP:    *hdat,
		HostID:    req.HostID,
		IPAddress: req.IPAddress,
	}

	// only commit the transaction if mutate returns true
	if !mutate(ip) {
		logger.Debug("Transaction aborted")
		return nil
	}

	// set the version object on the respective ips
	*pdat = ip.PoolIP
	pdat.SetVersion(pver)
	*hdat = ip.HostIP
	hdat.SetVersion(hver)

	if err := conn.NewTransaction().Set(ppth, pdat).Set(hpth, hdat).Commit(); err != nil {
		logger.WithError(err).Debug("Could not commit transaction")
		// TODO: error
		return err
	}
	logger.Debug("Updated ip")
	return nil
}

// DeleteIP removes the ip from the pool and the host
func DeleteIP(conn client.Connection, req IPRequest) error {
	logger := plog.WithFields(log.Fields{
		"hostid": req.HostID,
		"ipaddress", req.IPAddress,
	})

	basepth := "/"
	if req.PoolID != "" {
		basepth = path.Join("/pools", req.PoolID)
	}

	t := conn.NewTransaction()

	// Delete the pool ip
	ppth := path.Join(basepth, "/ips", req.IPID())
	if ok, err := conn.Exists(ppth); err != nil {
		logger.WithError(err).Debug("Could not look up pool ip")
		// TODO: error
		return err
	} else if ok {
		t.Delete(ppth)
	} else {
		logger.Debug("No ip to delete on the pool")
	}

	// Delete the host ip
	hpth := path.Join(basepth, "/hosts", req.HostID, "/ips", req.IPID())
	if ok, err := conn.Exists(hpth); err != nil {
		logger.WithError(err).Debug("Could not look up host ip")
		// TODO: error
		return err
	} else if ok {
		t.Delete(hpth)
	} else {
		logger.Debug("No ip to delete on the host")
	}

	if err := t.Commit(); err != nil {
		logger.WithError(err).Debug("Could not commit transaction")
		// TODO: error
		return err
	}
	logger.Debug("Deleted ip")
	return nil
}
