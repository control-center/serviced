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

// publicendpointregistry is used for storing a list of public endpoints under a public endpoint key.
// The zookeeper structurs is:
//    /publicendpoints
//      /<publicendpoint key 1>
//         |--<PublicEndpoint>
//         |--<PublicEndpoint>
//      /<publicendpoint key 2>

package registry

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"

	"fmt"
	"path"
)

const (
	zkPublicEndpoints = "/publicendpoints"
)

func endpointPath(nodes ...string) string {
	p := []string{zkPublicEndpoints}
	p = append(p, nodes...)
	return path.Join(p...)
}

// NewPublicEndpoint creates a new PublicEndpoint
func NewPublicEndpoint(endpointName string, appEndpoint applicationendpoint.ApplicationEndpoint) PublicEndpoint {
	return PublicEndpoint{ApplicationEndpoint: appEndpoint, EndpointName: endpointName}
}

// PublicEndpointType is an Enum for endpoint type (VHost or Port)
type PublicEndpointType uint8

const (
	EPTypeVHost PublicEndpointType = 0
	EPTypePort  PublicEndpointType = 1
)

// PublicEndpoint contains information about a public endpoint
type PublicEndpoint struct {
	applicationendpoint.ApplicationEndpoint
	EndpointName string
	version      interface{}
}

// Version is an implementation of client.Node
func (p *PublicEndpoint) Version() interface{} { return p.version }

// SetVersion is an implementation of client.Node
func (p *PublicEndpoint) SetVersion(version interface{}) { p.version = version }

// PublicEndpointRegistryType is a specific registryType for public endpoints (ports and vhosts)
type PublicEndpointRegistryType struct {
	registryType
}

// PublicEndpointRegistry ensures the public endpoint registry and returns the PublicEndpointRegistryType type
func PublicEndpointRegistry(conn client.Connection) (*PublicEndpointRegistryType, error) {
	return &PublicEndpointRegistryType{registryType{getPath: endpointPath, ephemeral: true}}, nil
}

type PublicEndpointKey string

func GetPublicEndpointKey(endpointName string, epType PublicEndpointType) PublicEndpointKey {
	return PublicEndpointKey(fmt.Sprintf("%s-%d", endpointName, epType))
}

//SetItem adds or replaces the PublicEndpoint to the key in registry.  Returns the path of the node in the registry
func (per *PublicEndpointRegistryType) SetItem(conn client.Connection, key PublicEndpointKey, node PublicEndpoint) (string, error) {
	verr := validation.NewValidationError()

	verr.Add(validation.NotEmpty("ServiceID", node.ServiceID))
	verr.Add(validation.NotEmpty("EndpointName", node.EndpointName))
	if verr.HasError() {
		return "", verr
	}

	nodeID := fmt.Sprintf("%s_%s", node.ServiceID, node.EndpointName)
	return per.setItem(conn, string(key), nodeID, &node)
}

//GetItem gets PublicEndpoint at the given path.
func (per *PublicEndpointRegistryType) GetItem(conn client.Connection, path string) (*PublicEndpoint, error) {
	var pep PublicEndpoint
	if err := conn.Get(path, &pep); err != nil {
		glog.Infof("Could not get public endpoint at %s: %s", path, err)
		return nil, err
	}
	return &pep, nil
}

// GetChildren gets all child paths for a tenant and endpoint
func (per *PublicEndpointRegistryType) GetChildren(conn client.Connection, pepKey PublicEndpointKey) ([]string, error) {
	return per.getChildren(conn, string(pepKey))
}

// GetPublicEndpointKeyChildren gets the ephemeral nodes of a public endpoint key (example of a key is 'hbase')
func (per *PublicEndpointRegistryType) GetPublicEndpointKeyChildren(conn client.Connection, publicendpointkey PublicEndpointKey) ([]PublicEndpoint, error) {
	var publicEndpointEphemeralNodes []PublicEndpoint

	publicEndpointChildren, err := conn.Children(endpointPath(string(publicendpointkey)))
	if err == client.ErrNoNode {
		return publicEndpointEphemeralNodes, nil
	}
	if err != nil {
		return publicEndpointEphemeralNodes, err
	}

	for _, publicEndpointChild := range publicEndpointChildren {
		var pep PublicEndpoint
		if err := conn.Get(endpointPath(string(publicendpointkey), publicEndpointChild), &pep); err != nil {
			return publicEndpointEphemeralNodes, err
		}
		publicEndpointEphemeralNodes = append(publicEndpointEphemeralNodes, pep)
	}

	return publicEndpointEphemeralNodes, nil
}

//WatchPublicEndpoint watch a specific PublicEnpoint
func (per *PublicEndpointRegistryType) WatchPublicEndpoint(conn client.Connection, path string, cancel <-chan interface{}, processPublicEndpoint func(conn client.Connection,
	node *PublicEndpoint), errorHandler WatchError) error {

	processNode := func(conn client.Connection, node client.Node) {
		publicEndpoint := node.(*PublicEndpoint)
		processPublicEndpoint(conn, publicEndpoint)
	}

	var pep PublicEndpoint
	return per.watchItem(conn, path, &pep, cancel, processNode, errorHandler)
}
