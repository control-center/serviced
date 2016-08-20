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

// GetPublicPort returns port data for a specific port address on a host
func GetPublicPort(conn client.Connection, hostID, portAddr string) (*PublicPort, error) {
	pth := path.Join("/net/pub", hostID, portAddr)

	logger := log.WithFields(log.Fields{
		"hostid":      hostID,
		"portaddress": portAddr,
		"zkpath":      pth,
	})

	pub := &PublicPort{}
	err := conn.Get(pth, pub)
	if err == client.ErrNoNode {
		logger.WithError(err).Debug("Port not found on host")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "port not found on host",
		}
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up port")
		return nil, &RegistryError{
			Action:  "get",
			Path:    pth,
			Message: "could not look up port",
		}
	}

	logger.WithFields(log.Fields{
		"tenantid":    pub.TenantID,
		"application": pub.Application,
	}).Debug("Found port")

	return pub, nil
}
