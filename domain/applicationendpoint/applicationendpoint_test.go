// Copyright 2015 The Serviced Authors.
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

// +build unit

package applicationendpoint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"reflect"
)

func TestApplicationEndpointGetID_EmptyObject(t *testing.T) {
	endpoint := &ApplicationEndpoint{}

	assert.Equal(t, endpoint.GetID(), "/0  ")
}

func TestApplicationEndpointGetID_ValidObject(t *testing.T) {
	endpoint := buildApplicationEndpoint()

	assert.Equal(t, endpoint.GetID(), "serviceid1/1 purpose1 application1")
}


func TestApplicationEndpointFind_EmptyList(t *testing.T) {
	endpoint := buildApplicationEndpoint()
	var endpoints []ApplicationEndpoint

	assert.Nil(t, endpoint.Find(endpoints))
}

func TestApplicationEndpointFind_NotFound(t *testing.T) {
	endpoint := buildApplicationEndpoint()
	var endpoints []ApplicationEndpoint
	for i := 0; i < 3; i++ {
		entry := buildApplicationEndpoint()
		entry.Application = fmt.Sprintf("%s_%d", endpoint.Application, i)
		endpoints = append(endpoints, *entry)
	}

	assert.Nil(t, endpoint.Find(endpoints))
}

func TestApplicationEndpointFind_MatchFirst(t *testing.T) {
	endpoint := buildApplicationEndpoint()
	var endpoints []ApplicationEndpoint
	for i := 0; i < 3; i++ {
		entry := buildApplicationEndpoint()
		entry.Application = fmt.Sprintf("%s_%d", endpoint.Application, i)
		endpoints = append(endpoints, *entry)
	}
	endpoints[0] = *endpoint

	result := endpoint.Find(endpoints)
	assert.NotNil(t, result)
	assert.True(t, reflect.DeepEqual(endpoint, result))
}

func TestApplicationEndpointFind_MatchLast(t *testing.T) {
	endpoint := buildApplicationEndpoint()
	var endpoints []ApplicationEndpoint
	for i := 0; i < 3; i++ {
		entry := buildApplicationEndpoint()
		entry.Application = fmt.Sprintf("%s_%d", endpoint.Application, i)
		endpoints = append(endpoints, *entry)
	}
	endpoints[2] = *endpoint

	result := endpoint.Find(endpoints)
	assert.NotNil(t, result)
	assert.True(t, reflect.DeepEqual(endpoint, result))
}

func TestApplicationEndpointEquals_EmptyObjects(t *testing.T) {
	endpoint1 := &ApplicationEndpoint{}
	endpoint2 := &ApplicationEndpoint{}

	assert.True(t, endpoint1.Equals(endpoint2))
}


func TestApplicationEndpointEquals_ValidObjects(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()

	assert.True(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_ServiceIDDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.ServiceID = "serviceID2"

	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_InstanceIDDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.InstanceID = 2

	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_ApplicationDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.Application = "application2"

	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_HostIDDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.HostID = "hostID2"
	
	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_HostIPDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.HostIP = "hostIP2"
	
	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_HostPortDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.HostPort = 2
	
	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_ContainerIDDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.ContainerID = "ContainerID2"
	
	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_ContainerIPDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.ContainerIP = "ContainerIP2"
	
	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_ContainerPortDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.ContainerPort = 2
	
	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_ProtocolDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.Protocol = "udp"

	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_VirtualAddressDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.VirtualAddress = "VirtualAddress2"

	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_PurposeDiffers(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.Purpose = "purpose2"

	assert.False(t, endpoint1.Equals(endpoint2))
}

func TestApplicationEndpointEquals_IngoresProxyPort(t *testing.T) {
	endpoint1 := buildApplicationEndpoint()
	endpoint2 := buildApplicationEndpoint()
	endpoint2.ProxyPort = 2

	assert.True(t, endpoint1.Equals(endpoint2))
}

func buildApplicationEndpoint() *ApplicationEndpoint {
	return &ApplicationEndpoint{
		ServiceID:      "serviceID1",
		InstanceID:     1,
		Application:    "application1",
		Purpose:        "purpose1",
		HostID:         "hostID1",
		HostIP:         "hostIP1",
		HostPort:       1,
		ContainerID:    "containerID1",
		ContainerIP:    "containerIP1",
		ContainerPort:  1,
		Protocol:       "tcp",
		VirtualAddress: "virtualAddress1",
		ProxyPort:      1,
	}
}
