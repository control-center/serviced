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

// AggregateService is a lighter service object for providing aggregate service
// status information
type AggregateService struct {
	ServiceID    string
	DesiredState DesiredState
	Status       []StatusInstance
	NotFound     bool
}

// PublicEndpoint is a minimal service object that describes a public endpoint
// for a service
type PublicEndpoint struct {
	ServiceID   string
	ServiceName string
	Application string
	Protocol    string
	VHostName   string `json:",omitempty"`
	PortAddress string `json:",omitempty"`
	Enabled     bool
}

// IPAssignment is a minimal service object that describes an address assignment
// for a service.
type IPAssignment struct {
	ServiceID   string
	ServiceName string
	PoolID      string
	HostID      string
	HostName    string
	Type        string
	IPAddress   string
	Port        uint16
}

// ExportedEndpoint is a minimal service object that describes exported
// endpoints for a service.
// NOTE: Could add booleans for ip assignment, vhost, and ports
type ExportedEndpoint struct {
	ServiceID   string
	ServiceName string
	Application string
	Protocol    string
}

// Config displays the most basic information about a service config file
type Config struct {
	ID       string
	Filename string
}
