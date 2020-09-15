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

package service

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	svcdef "github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/utils"
)

// DesiredState is the desired state of a service.
type DesiredState int

var protocolPrefixRegex = regexp.MustCompile("^(.+://)")

// ToAuditAction returns the audit name of a DesiredState value.
func (state DesiredState) ToAuditAction() string {
	switch state {
	case SVCRestart:
		return "restart"
	case SVCStop:
		return "stop"
	case SVCRun:
		return "start"
	case SVCPause:
		return "pause"
	default:
		return "unknown"
	}
}

// String returns the text representation of a DesiredState value.
func (state DesiredState) String() string {
	switch state {
	case SVCRestart:
		return "restart"
	case SVCStop:
		return "stop"
	case SVCRun:
		return "go"
	case SVCPause:
		return "pause"
	default:
		return "unknown"
	}
}

const (
	// SVCRestart is the 'restart' DesiredState value.
	SVCRestart = DesiredState(-1)

	// SVCStop is the 'stop' DesiredState value.
	SVCStop = DesiredState(0)

	// SVCRun is the 'run' DesiredState value.
	SVCRun = DesiredState(1)

	// SVCPause is the 'pause' DesiredState value.
	SVCPause = DesiredState(2)
)

// ServiceCurrentState is the current state of a service.
type ServiceCurrentState string

const (
	// SVCCSUnknown is the 'unknown' current state.
	SVCCSUnknown = ServiceCurrentState("unknown")

	// SVCCSStopped is the 'stopped' current state.
	SVCCSStopped = ServiceCurrentState("stopped")

	// SVCCSPendingStart is the 'pending start' current state.
	SVCCSPendingStart = ServiceCurrentState("pending_start")

	// SVCCSStarting is the 'starting' current state.
	SVCCSStarting = ServiceCurrentState("starting")

	// SVCCSRunning is the 'started' current state.
	SVCCSRunning = ServiceCurrentState("started")

	// SVCCSPendingRestart is the 'pending restart' current state.
	SVCCSPendingRestart = ServiceCurrentState("pending_restart")

	// SVCCSRestarting is the 'restarting' current state.
	SVCCSRestarting = ServiceCurrentState("restarting")

	// SVCCSPendingStop is the 'pending stop' current state.
	SVCCSPendingStop = ServiceCurrentState("pending_stop")

	// SVCCSStopping is the 'stopping' current state.
	SVCCSStopping = ServiceCurrentState("stopping")

	// SVCCSPendingPause is the 'pending pause' current state.
	SVCCSPendingPause = ServiceCurrentState("pending_pause")

	// SVCCSPausing is the 'pausing' current state.
	SVCCSPausing = ServiceCurrentState("pausing")

	// SVCCSPaused is the 'paused' current state.
	SVCCSPaused = ServiceCurrentState("paused")

	// SVCCSPendingEmergencyStop is the 'pending emergency stop' current state.
	SVCCSPendingEmergencyStop = ServiceCurrentState("pending_emergency_stop")

	// SVCCSEmergencyStopping is the 'emergency stopping' current state.
	SVCCSEmergencyStopping = ServiceCurrentState("emergency_stopping")

	// SVCCSEmergencyStopped is the 'emergency stopped' current state.
	SVCCSEmergencyStopped = ServiceCurrentState("emergency_stopped")
)

var (
	serviceCurrentStates = map[ServiceCurrentState]struct{}{
		SVCCSUnknown:              struct{}{},
		SVCCSStopped:              struct{}{},
		SVCCSPendingStart:         struct{}{},
		SVCCSStarting:             struct{}{},
		SVCCSRunning:              struct{}{},
		SVCCSPendingRestart:       struct{}{},
		SVCCSRestarting:           struct{}{},
		SVCCSPendingStop:          struct{}{},
		SVCCSStopping:             struct{}{},
		SVCCSPendingPause:         struct{}{},
		SVCCSPausing:              struct{}{},
		SVCCSPaused:               struct{}{},
		SVCCSPendingEmergencyStop: struct{}{},
		SVCCSEmergencyStopping:    struct{}{},
		SVCCSEmergencyStopped:     struct{}{},
	}
)

// Validate verifies that a ServiceCurrentState has a valid value.
func (state ServiceCurrentState) Validate() error {
	if _, ok := serviceCurrentStates[state]; !ok {
		return errors.New("invalid current state")
	}
	return nil
}

// DesiredToCurrentPendingState converts a DesiredState value to a ServiceCurrentState 'pending' value.
func DesiredToCurrentPendingState(state DesiredState, emergency bool) ServiceCurrentState {
	switch state {
	case SVCRestart:
		return SVCCSPendingRestart
	case SVCStop:
		if emergency {
			return SVCCSPendingEmergencyStop
		}
		return SVCCSPendingStop
	case SVCRun:
		return SVCCSPendingStart
	case SVCPause:
		return SVCCSPendingPause
	default:
		return SVCCSUnknown
	}
}

// DesiredToCurrentTransitionState converts a DesiredState value to a ServiceCurrentState 'transition' value.
func DesiredToCurrentTransitionState(state DesiredState, emergency bool) ServiceCurrentState {
	switch state {
	case SVCRestart:
		return SVCCSRestarting
	case SVCStop:
		if emergency {
			return SVCCSEmergencyStopping
		}
		return SVCCSStopping
	case SVCRun:
		return SVCCSStarting
	case SVCPause:
		return SVCCSPausing
	default:
		return SVCCSUnknown
	}
}

// DesiredToCurrentFinalState converts a DesiredState value to a ServiceCurrentState 'final' value.
func DesiredToCurrentFinalState(state DesiredState, emergency bool) ServiceCurrentState {
	switch state {
	case SVCRestart:
		return SVCCSRunning
	case SVCStop:
		if emergency {
			return SVCCSEmergencyStopped
		}
		return SVCCSStopped
	case SVCRun:
		return SVCCSRunning
	case SVCPause:
		return SVCCSPaused
	default:
		return SVCCSUnknown
	}
}

// DesiredCancelsPending returns true if the desired state acts as a "cancel" to a pending current state.
func DesiredCancelsPending(pendingState ServiceCurrentState, desiredState DesiredState) bool {
	switch pendingState {
	case SVCCSPendingStart, SVCCSPendingRestart:
		return desiredState == SVCRun
	case SVCCSPendingStop, SVCCSPendingPause:
		return desiredState == SVCStop
	}
	return false
}

// DesiredStateIsRedundant returns true if setting the desired state would be unnecessary.
func DesiredStateIsRedundant(desiredState DesiredState, emergency bool, currentState ServiceCurrentState) bool {
	switch desiredState {
	case SVCRun:
		return currentState == SVCCSRunning || currentState == SVCCSStarting || currentState == SVCCSPendingStart
	case SVCRestart:
		return currentState == SVCCSRestarting || currentState == SVCCSPendingRestart
	case SVCStop:
		if emergency {
			return currentState == SVCCSEmergencyStopped || currentState == SVCCSEmergencyStopping || currentState == SVCCSPendingEmergencyStop
		}
		return currentState == SVCCSStopped || currentState == SVCCSStopping || currentState == SVCCSPendingStop
	case SVCPause:
		return currentState == SVCCSPaused || currentState == SVCCSPausing || currentState == SVCCSPendingPause
	}

	return false
}

// Service is a service that can run in serviced.
// A Service is created when a ServiceDefinition is deployed.
type Service struct {
	ID              string
	Name            string
	Title           string // Title is a label used when describing this service in the context of a service tree
	Version         string
	Context         map[string]interface{}
	Environment     []string
	Startup         string
	RunAs           string
	Description     string
	Tags            []string
	OriginalConfigs map[string]svcdef.ConfigFile
	ConfigFiles     map[string]svcdef.ConfigFile
	Instances       int
	InstanceLimits  domain.MinMax
	ChangeOptions   []svcdef.ChangeOption
	ImageID         string
	PoolID          string
	DesiredState    int
	CurrentState    string
	HostPolicy      svcdef.HostPolicy
	Hostname        string
	Privileged      bool
	Launch          string
	Endpoints       []ServiceEndpoint
	ParentServiceID string
	Volumes         []svcdef.Volume
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeploymentID    string
	DisableImage    bool
	LogConfigs      []svcdef.LogConfig
	Snapshot        svcdef.SnapshotCommands
	DisableShell    bool
	Runs            map[string]string // FIXME: This field is deprecated. Remove when possible.
	Commands        map[string]domain.Command
	RAMCommitment   utils.EngNotation
	RAMThreshold    uint
	CPUCommitment   uint64
	Actions         map[string]string
	HealthChecks    map[string]health.HealthCheck // A health check for the service.

	// Prereqs is a list of scripts that must run successfully before running the command in the Startup field.
	Prereqs []domain.Prereq

	MonitoringProfile domain.MonitorProfile
	MemoryLimit       float64
	CPUShares         int64
	OomKillDisable    bool
	OomScoreAdj       int64
	PIDFile           string

	// StartLevel represents the order in which services are started and stopped
	// in normal operations.  All services of a given level start before any services
	// at higher levels.  Stopping services occurs in the reverse order.  Services
	// with StartLevel 0 (representing undefined) start after and stop before services
	// with a defined StartLevel.
	StartLevel uint

	// EmergencyShutdownLevel represents the order in which services are stopped in an
	// emergency low-storage situation.  All services of a given EmergencyShutdownLevel are
	// stopped before any services of a higher EmergencyShutdownLevel.  In an emergency
	// low-storage shutdown, services with EmergencyShutdownLevel 0 (representing undefined)
	// are stopped after services with a defined EmergencyShutdownLevel, in the normal order
	// dictated by their StartLevel.
	EmergencyShutdownLevel uint

	// EmergencyShutdown is a flag that indicates whether this service has been shutdown due
	// to an emergency (low-storage) situation.  Services with this flag set can not be started
	EmergencyShutdown bool
	datastore.VersionedEntity
}

// NewService Create a new Service.
func NewService() (s *Service, err error) {
	s = &Service{}
	s.ID, err = utils.NewUUID36()
	return s, err
}

// HasEndpointsFor determines if the service has any imports
// for the specified purpose, eg import
func (s *Service) HasEndpointsFor(purpose string) bool {
	if s.Endpoints == nil {
		return false
	}

	for _, ep := range s.Endpoints {
		if ep.Purpose == purpose {
			return true
		}
	}
	return false
}

// BuildService build a service from a ServiceDefinition.
func BuildService(
	sd svcdef.ServiceDefinition, parentServiceID string, poolID string, desiredState int, deploymentID string,
) (*Service, error) {
	svcuuid, err := utils.NewUUID36()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	svc := Service{}
	svc.ID = svcuuid
	svc.Name = sd.Name
	svc.Title = sd.Title
	svc.Version = sd.Version
	svc.Context = sd.Context
	svc.Startup = sd.Command
	svc.RunAs = sd.RunAs
	svc.Description = sd.Description
	svc.Environment = sd.Environment
	svc.Tags = sd.Tags
	if sd.Instances.Default != 0 {
		svc.Instances = sd.Instances.Default
	} else {
		svc.Instances = sd.Instances.Min
	}
	svc.InstanceLimits = sd.Instances
	svc.ChangeOptions = sd.ChangeOptions
	svc.ImageID = sd.ImageID
	svc.PoolID = poolID
	svc.DesiredState = desiredState
	svc.Launch = sd.Launch
	svc.HostPolicy = sd.HostPolicy
	svc.Hostname = sd.Hostname
	svc.Privileged = sd.Privileged
	svc.OriginalConfigs = sd.ConfigFiles
	svc.ConfigFiles = sd.ConfigFiles
	svc.ParentServiceID = parentServiceID
	svc.CreatedAt = now
	svc.UpdatedAt = now
	svc.Volumes = sd.Volumes
	svc.DeploymentID = deploymentID
	svc.LogConfigs = sd.LogConfigs
	svc.Snapshot = sd.Snapshot
	svc.RAMCommitment = sd.RAMCommitment
	svc.RAMThreshold = sd.RAMThreshold
	svc.CPUCommitment = sd.CPUCommitment
	svc.DisableShell = sd.DisableShell
	svc.Runs = sd.Runs
	svc.Commands = sd.Commands
	svc.Actions = sd.Actions
	svc.HealthChecks = sd.HealthChecks
	svc.Prereqs = sd.Prereqs
	svc.PIDFile = sd.PIDFile
	svc.StartLevel = sd.StartLevel
	svc.EmergencyShutdownLevel = sd.EmergencyShutdownLevel
	svc.OomKillDisable = sd.OomKillDisable
	svc.OomScoreAdj = sd.OomScoreAdj

	svc.Endpoints = make([]ServiceEndpoint, 0)
	for _, ep := range sd.Endpoints {
		svc.Endpoints = append(svc.Endpoints, BuildServiceEndpoint(ep))
	}

	tags := map[string][]string{
		"controlplane_service_id": []string{svc.ID},
	}
	profile, err := sd.MonitoringProfile.ReBuild("1h-ago", tags)
	if err != nil {
		return nil, err
	}
	svc.MonitoringProfile = *profile
	svc.MemoryLimit = sd.MemoryLimit
	svc.CPUShares = sd.CPUShares

	return &svc, nil
}

// CloneService copies a service and mutates id and names
func CloneService(fromSvc *Service, suffix string) (*Service, error) {
	svcuuid, err := utils.NewUUID36()
	if err != nil {
		return nil, err
	}

	svc := *fromSvc
	svc.ID = svcuuid
	svc.DesiredState = int(SVCStop)

	now := time.Now()
	svc.CreatedAt = now
	svc.UpdatedAt = now

	// add suffix to make certain things unique
	suffix = strings.TrimSpace(suffix)
	if len(suffix) == 0 {
		suffix = "-" + svc.ID[0:12]
	}
	svc.Name += suffix
	for idx, ep := range svc.Endpoints {
		if ep.Purpose == "export" {
			svc.Endpoints[idx].Name += suffix
			svc.Endpoints[idx].Application += suffix
			svc.Endpoints[idx].ApplicationTemplate += suffix
		}
	}
	for idx := range svc.Volumes {
		svc.Volumes[idx].ResourcePath += suffix
	}

	// CC-2750: filter invalid monitoring profile data
	var metricConfigs = []domain.MetricConfig{}
	for _, mc := range svc.MonitoringProfile.MetricConfigs {
		if mc.ID == "metrics" {
			continue
		}

		var metrics = []domain.Metric{}
		for _, m := range mc.Metrics {
			if !m.BuiltIn {
				metrics = append(metrics, m)
			}
		}
		mc.Metrics = metrics
		metricConfigs = append(metricConfigs, mc)
	}
	svc.MonitoringProfile.MetricConfigs = metricConfigs

	var graphConfigs = []domain.GraphConfig{}
	for _, gc := range svc.MonitoringProfile.GraphConfigs {
		if !gc.BuiltIn {
			graphConfigs = append(graphConfigs, gc)
		}
	}
	svc.MonitoringProfile.GraphConfigs = graphConfigs

	return &svc, nil
}

// GetServiceImports retrieves service endpoints whose purpose is "import"
func (s *Service) GetServiceImports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "import" || ep.Purpose == "import_all" {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServiceExports retrieves service endpoints whose purpose is "export"
func (s *Service) GetServiceExports() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if ep.Purpose == "export" {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServiceVHosts retrieves service endpoints that specify a virtual HostPort
func (s *Service) GetServiceVHosts() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if len(ep.VHostList) > 0 {
				result = append(result, ep)
			}
		}
	}

	return result
}

// GetServicePorts retrieves service endpoints that specify additional port(s)
func (s *Service) GetServicePorts() []ServiceEndpoint {
	result := []ServiceEndpoint{}

	if s.Endpoints != nil {
		for _, ep := range s.Endpoints {
			if len(ep.PortList) > 0 {
				result = append(result, ep)
			}
		}
	}

	return result
}

// AddVirtualHost Add a virtual host for given service, this method avoids duplicates vhosts
func (s *Service) AddVirtualHost(application, vhostName string, isEnabled bool) (*svcdef.VHost, error) {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				_vhostName := strings.ToLower(vhostName)
				vhosts := make([]svcdef.VHost, 0)
				for _, vhost := range ep.VHostList {
					if strings.ToLower(vhost.Name) != _vhostName {
						vhosts = append(vhosts, vhost)
					}
				}
				vhost := &svcdef.VHost{Name: _vhostName, Enabled: isEnabled}
				ep.VHostList = append(vhosts, *vhost)
				return vhost, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to find application %s in service: %s", application, s.Name)
}

// GetVirtualHost returns the matching VHost entry or nil if not found.
func (s *Service) GetVirtualHost(application, vhostName string) *svcdef.VHost {
	if s.Endpoints != nil {
		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				vhostNameLower := strings.ToLower(vhostName)
				for _, vhost := range ep.VHostList {
					if strings.ToLower(vhost.Name) == vhostNameLower {
						return &vhost
					}
				}
			}
		}
	}

	return nil
}

// AddPort Add a port for given service, this method avoids duplicate ports
func (s *Service) AddPort(
	application string, portAddr string, usetls bool, protocol string, isEnabled bool,
) (*svcdef.Port, error) {
	portAddr = ScrubPortString(portAddr)
	if s.Endpoints != nil {
		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				var ports = make([]svcdef.Port, 0)
				portAddrLower := strings.ToLower(portAddr)
				for _, port := range ep.PortList {
					if strings.ToLower(port.PortAddr) != portAddrLower {
						ports = append(ports, port)
					}
				}
				port := &svcdef.Port{PortAddr: portAddr, Enabled: isEnabled, UseTLS: usetls, Protocol: protocol}
				ep.PortList = append(ports, *port)
				return port, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to find application %s in service: %s", application, s.Name)
}

// GetPort returns the matching Port entry or nil if not found.
func (s *Service) GetPort(application, portAddr string) *svcdef.Port {
	if s.Endpoints != nil {
		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				portAddrLower := strings.ToLower(portAddr)
				for _, port := range ep.PortList {
					if strings.ToLower(port.PortAddr) == portAddrLower {
						return &port
					}
				}
			}
		}
	}

	return nil
}

// RemovePort Remove a port for given service
func (s *Service) RemovePort(application string, portAddr string) error {
	if s.Endpoints == nil {
		return fmt.Errorf("Service %s has no Endpoints", s.Name)
	}

	//find the matching endpoint
	for i := range s.Endpoints {
		ep := &s.Endpoints[i]

		if ep.Application == application && ep.Purpose == "export" {
			if len(ep.PortList) == 0 {
				break
			}

			portFound := false
			var ports = make([]svcdef.Port, 0)
			for _, port := range ep.PortList {
				if port.PortAddr != portAddr {
					ports = append(ports, port)
				} else {
					portFound = true
				}
			}

			//error removing an unknown port
			if !portFound {
				return fmt.Errorf("endpoint %s does not have a port endpoint %s", application, portAddr)
			}

			ep.PortList = ports
			return nil
		}
	}

	return fmt.Errorf("unable to find application %s in service: %s", application, s.Name)
}

// EnablePort enables or disables a port for given service
func (s *Service) EnablePort(application string, portAddr string, enable bool) error {
	appFound := false
	portFound := false
	for _, ep := range s.GetServicePorts() {
		if ep.Application == application {
			appFound = true
			for i, port := range ep.PortList {
				if port.PortAddr == portAddr {
					portFound = true
					ep.PortList[i].Enabled = enable
					plog.WithFields(log.Fields{
						"portaddr":    portAddr,
						"serviceid":   s.ID,
						"application": application,
						"enable":      enable,
					}).Debug("Enable port")
				}
			}
		}
	}
	if !appFound {
		return fmt.Errorf(
			"port %s not found; application %s not found in service %s:%s", portAddr, application, s.ID, s.Name,
		)
	}
	if !portFound {
		return fmt.Errorf("port %s not found in service %s:%s", portAddr, s.ID, s.Name)
	}

	return nil
}

// ScrubPortString makes a best effort to produce a valid port address.
func ScrubPortString(port string) string {
	// remove possible protocol at string beginning
	scrubbed := protocolPrefixRegex.ReplaceAllString(port, "")

	matched, _ := regexp.MatchString("^[0-9]+$", scrubbed)
	if matched {
		scrubbed = fmt.Sprintf(":%s", scrubbed)
	}

	return scrubbed
}

// EnableVirtualHost enable or disable a virtual host for given service
func (s *Service) EnableVirtualHost(application, vhostName string, enable bool) error {
	appFound := false
	vhostFound := false
	for _, ep := range s.GetServiceVHosts() {
		if ep.Application == application {
			appFound = true
			for i, vhost := range ep.VHostList {
				if vhost.Name == vhostName {
					vhostFound = true
					ep.VHostList[i].Enabled = enable
					plog.WithFields(log.Fields{
						"vhostname":   vhostName,
						"serviceid":   s.ID,
						"application": application,
						"enable":      enable,
					}).Debug("Enable vhost")
				}
			}
		}
	}
	if !appFound {
		return fmt.Errorf(
			"vhost %s not found; application %s not found in service %s:%s",
			vhostName, application, s.ID, s.Name,
		)
	}
	if !vhostFound {
		return fmt.Errorf("vhost %s not found in service %s:%s", vhostName, s.ID, s.Name)
	}

	return nil
}

// RemoveVirtualHost Remove a virtual host for given service
func (s *Service) RemoveVirtualHost(application, vhostName string) error {
	if s.Endpoints != nil {

		//find the matching endpoint
		for i := range s.Endpoints {
			ep := &s.Endpoints[i]

			if ep.Application == application && ep.Purpose == "export" {
				if len(ep.VHostList) == 0 {
					break
				}

				_vhostName := strings.ToLower(vhostName)
				found := false
				var vhosts = make([]svcdef.VHost, 0)
				for _, vhost := range ep.VHostList {
					if vhost.Name != _vhostName {
						vhosts = append(vhosts, vhost)
					} else {
						found = true
					}
				}
				//error removing an unknown vhost
				if !found {
					return fmt.Errorf("endpoint %s does not have a vhost endpoint %s", application, vhostName)
				}

				ep.VHostList = vhosts
				return nil
			}
		}
	}

	return fmt.Errorf("unable to find application %s in service: %s", application, s.Name)
}

// GetPath uses the GetService function to determine the / delimited name path i.e. /test/app/sevicename
func (s Service) GetPath(gs GetService) (string, error) {
	var err error
	svc := s
	path := fmt.Sprintf("/%s", s.Name)
	for svc.ParentServiceID != "" {
		svc, err = gs(svc.ParentServiceID)
		if err != nil {
			return "", err
		}
		path = fmt.Sprintf("/%s%s", svc.Name, path)
	}
	return path, nil
}

// SetAddressConfig sets the AddressConfig for the endpoint
func (s Service) SetAddressConfig(endpointName string, sa svcdef.AddressResourceConfig) error {
	if s.Endpoints == nil {
		return errors.New("service has no endpoints: " + s.Name)
	}

	for i := range s.Endpoints {
		ep := &s.Endpoints[i]

		if ep.Application == endpointName {
			ep.AddressConfig = sa
			return nil
		}
	}

	return errors.New("endpoint not found: " + endpointName)
}

// GetID returns its Service ID.
// It return the ID as a string
func (s *Service) GetID() string {
	return s.ID
}

// GetType return a service Entity type or kind.
// It returns the Kind as a string.
func (s *Service) GetType() string {
	return GetType()
}

// Equals are they the same
func (s *Service) Equals(b *Service) bool {
	if s.ID != b.ID {
		return false
	}
	if s.Name != b.Name {
		return false
	}
	if s.Version != b.Version {
		return false
	}
	if !reflect.DeepEqual(s.Context, b.Context) {
		return false
	}
	if s.Startup != b.Startup {
		return false
	}
	if s.Description != b.Description {
		return false
	}
	if s.Instances != b.Instances {
		return false
	}
	if s.ImageID != b.ImageID {
		return false
	}
	if s.PoolID != b.PoolID {
		return false
	}
	if s.DesiredState != b.DesiredState {
		return false
	}
	if s.Launch != b.Launch {
		return false
	}
	if s.Hostname != b.Hostname {
		return false
	}
	if s.Privileged != b.Privileged {
		return false
	}
	if s.HostPolicy != b.HostPolicy {
		return false
	}
	if s.ParentServiceID != b.ParentServiceID {
		return false
	}
	if s.CreatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}
	if s.UpdatedAt.Unix() != b.UpdatedAt.Unix() {
		return false
	}
	return true
}

// GetType returns a Services's type or kind, can be used to get
// the string value of Service's type without a Service instance.
// It returns the kind as a string.
func GetType() string {
	return kind
}
