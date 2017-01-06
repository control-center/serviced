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

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/health"

	"github.com/control-center/serviced/domain/host"
	"github.com/pivotal-golang/bytefmt"
)

const ()

var ()

// ServiceConfig is the deserialized object from the command-line
type ServiceConfig struct {
	Name            string
	ParentServiceID string
	ImageID         string
	Command         string
	LocalPorts      *PortMap
	RemotePorts     *PortMap
}

type SchedulerConfig struct {
	ServiceID   string
	AutoLaunch  bool
	Synchronous bool
}

// IPConfig is the deserialized object from the command-line
type IPConfig struct {
	ServiceID string
	IPAddress string
}

// Type of method that controls the state of a service
type ServiceStateController func(SchedulerConfig) (int, error)

func (a *api) GetAllServiceDetails() ([]service.ServiceDetails, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetAllServiceDetails(0)
}

func (a *api) GetServiceDetails(serviceID string) (*service.ServiceDetails, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	if svc, err := client.GetServiceDetails(serviceID); err != nil {
		return nil, err
	} else {
		return svc, nil
	}
}

func (a *api) GetServiceStatus(serviceID string) (map[string]map[string]interface{}, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}
	masterClient, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	// get services
	var svcs []service.ServiceDetails
	if serviceID = strings.TrimSpace(serviceID); serviceID != "" {
		for serviceID != "" {
			if svc, err := masterClient.GetServiceDetails(serviceID); err != nil {
				return nil, err
			} else {
				svcs = append(svcs, *svc)
				serviceID = svc.ParentServiceID
			}
		}
	} else {
		svcs, err = masterClient.GetAllServiceDetails(0)
		if err != nil {
			return nil, err
		}
	}

	// get status
	rowmap := make(map[string]map[string]interface{})
	for _, svc := range svcs {
		var status []service.Instance
		if err := client.GetServiceStatus(svc.ID, &status); err != nil {
			return nil, err
		}

		if status == nil || len(status) == 0 {
			row := make(map[string]interface{})
			row["ServiceID"] = svc.ID
			row["Name"] = svc.Name
			if svc.ParentServiceID != "" {
				row["ParentID"] = fmt.Sprintf("%s/%d", svc.ParentServiceID, 0) //make this match the rowmap key
			} else {
				row["ParentID"] = ""
			}
			row["RAM"] = bytefmt.ByteSize(svc.RAMCommitment.Value)

			if svc.Instances > 0 {
				switch service.DesiredState(svc.DesiredState) {
				case service.SVCRun:
					row["Status"] = "Scheduled"
				case service.SVCPause:
					row["Status"] = service.Paused
				case service.SVCStop:
					row["Status"] = service.Stopped
				}
			}
			rowmap[fmt.Sprintf("%s/%d", svc.ID, 0)] = row
		} else {
			for _, stat := range status {
				row := make(map[string]interface{})
				row["ServiceID"] = svc.ID
				if svc.ParentServiceID != "" {
					row["ParentID"] = fmt.Sprintf("%s/%d", svc.ParentServiceID, 0) //make this match the rowmap key
				} else {
					row["ParentID"] = ""
				}

				//round to uptime to nearest second
				var uptime time.Duration
				if stat.Started.After(stat.Terminated) {
					uptime = time.Since(stat.Started)
				}
				remainder := uptime % time.Second
				uptime = uptime - remainder
				if remainder/time.Millisecond >= 500 {
					uptime += 1 * time.Second
				}

				row["RAM"] = bytefmt.ByteSize(svc.RAMCommitment.Value)
				row["Status"] = stat.CurrentState
				row["Hostname"] = stat.HostName
				row["DockerID"] = fmt.Sprintf("%.12s", stat.ContainerID)
				row["Uptime"] = uptime.String()

				if stat.ImageSynced {
					row["InSync"] = "Y"
				} else {
					row["InSync"] = "N"
				}
				if svc.Instances > 1 {
					row["Name"] = fmt.Sprintf("%s/%d", svc.Name, stat.InstanceID)
				} else {
					row["Name"] = svc.Name
				}
				cur := bytefmt.ByteSize(uint64(stat.MemoryUsage.Cur))
				max := bytefmt.ByteSize(uint64(stat.MemoryUsage.Max))
				avg := bytefmt.ByteSize(uint64(stat.MemoryUsage.Avg))
				row["Cur/Max/Avg"] = fmt.Sprintf("%s / %s / %s", cur, max, avg)

				rowmap[fmt.Sprintf("%s/%d", svc.ID, stat.InstanceID)] = row

				if stat.CurrentState == service.Running && len(stat.HealthStatus) > 0 {

					explicitFailure := false

					for hcName, hcResult := range stat.HealthStatus {
						newrow := make(map[string]interface{})
						newrow["ParentID"] = fmt.Sprintf("%s/%d", svc.ID, stat.InstanceID) //make this match the rowmap key
						newrow["Healthcheck"] = hcName
						newrow["Healthcheck Status"] = hcResult

						if hcResult == health.Failed {
							explicitFailure = true
						}

						rowmap[fmt.Sprintf("%s/%d-%v", svc.ID, stat.InstanceID, hcName)] = newrow
					}

					//go back and add the healthcheck field for the parent row
					if explicitFailure {
						row["HC Fail"] = "X"
					}
				}

			}
		}
	}

	return rowmap, nil

}

// Get all of the exported endpoints
func (a *api) GetEndpoints(serviceID string, reportImports, reportExports, validate bool) ([]applicationendpoint.EndpointReport, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, 0)
	serviceIDs = append(serviceIDs, serviceID)
	if endpoints, err := client.GetServiceEndpoints(serviceIDs, reportImports, reportExports, validate); err != nil {
		return nil, err
	} else {
		return endpoints, nil
	}
}

// Gets the service definition identified by its service ID. This is the full service object
func (a *api) GetService(id string) (*service.Service, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var s service.Service
	if err := client.GetService(id, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// Adds a new service
func (a *api) AddService(config ServiceConfig) (*service.ServiceDetails, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	endpoints := make([]servicedefinition.EndpointDefinition, len(*config.LocalPorts)+len(*config.RemotePorts))
	i := 0
	for _, e := range *config.LocalPorts {
		e.Purpose = "local"
		endpoints[i] = e
		i++
	}
	for _, e := range *config.RemotePorts {
		e.Purpose = "remote"
		endpoints[i] = e
		i++
	}

	sd := &servicedefinition.ServiceDefinition{
		Name:      config.Name,
		Command:   config.Command,
		Instances: domain.MinMax{Min: 1, Max: 1, Default: 1},
		ImageID:   config.ImageID,
		Launch:    commons.AUTO,
		Endpoints: endpoints,
	}

	var serviceID string
	if err := client.DeployService(dao.ServiceDeploymentRequest{ParentID: config.ParentServiceID, Service: *sd}, &serviceID); err != nil {
		return nil, err
	}

	return a.GetServiceDetails(serviceID)
}

// CloneService copies an existing service
func (a *api) CloneService(serviceID string, suffix string) (*service.ServiceDetails, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	request := dao.ServiceCloneRequest{ServiceID: serviceID, Suffix: suffix}
	clonedServiceID := ""
	if err := client.CloneService(request, &clonedServiceID); err != nil {
		return nil, fmt.Errorf("copy service failed: %s", err)
	}
	return a.GetServiceDetails(clonedServiceID)
}

// RemoveService removes an existing service
func (a *api) RemoveService(id string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	if err := client.RemoveService(id, new(int)); err != nil {
		return fmt.Errorf("could not remove service %s: %s", id, err)
	}
	return nil
}

// UpdateService updates an existing service
func (a *api) UpdateService(reader io.Reader) (*service.ServiceDetails, error) {
	// Unmarshal JSON from the reader
	var s service.Service
	if err := json.NewDecoder(reader).Decode(&s); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}

	// Connect to the client
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	// Update the service
	if err := client.UpdateService(s, &unusedInt); err != nil {
		return nil, err
	}

	return a.GetServiceDetails(s.ID)
}

// StartService starts a service
func (a *api) StartService(config SchedulerConfig) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 0, err
	}

	var affected int
	err = client.StartService(dao.ScheduleServiceRequest{config.ServiceID, config.AutoLaunch, config.Synchronous}, &affected)
	return affected, err
}

// Restart
func (a *api) RestartService(config SchedulerConfig) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 0, err
	}

	var affected int
	err = client.RestartService(dao.ScheduleServiceRequest{config.ServiceID, config.AutoLaunch, config.Synchronous}, &affected)
	return affected, err
}

// StopService stops a service
func (a *api) StopService(config SchedulerConfig) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 0, err
	}

	var affected int
	err = client.StopService(dao.ScheduleServiceRequest{config.ServiceID, config.AutoLaunch, config.Synchronous}, &affected)
	return affected, err
}

// AssignIP assigns an IP address to a service
func (a *api) AssignIP(config IPConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	req := addressassignment.AssignmentRequest{
		ServiceID:      config.ServiceID,
		IPAddress:      config.IPAddress,
		AutoAssignment: config.IPAddress == "",
	}

	if err := client.AssignIPs(req, nil); err != nil {
		return err
	}

	return nil
}

func (a *api) GetHostMap() (map[string]host.Host, error) {
	hosts, err := a.GetHosts()
	if err != nil {
		return nil, err
	}
	hostmap := make(map[string]host.Host)
	for _, host := range hosts {
		hostmap[host.ID] = host
	}
	return hostmap, nil
}

func (a *api) ResolveServicePath(path string) ([]service.ServiceDetails, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	return client.ResolveServicePath(path)
}

func (a *api) ClearEmergency(serviceID string) (int, error) {
	client, err := a.connectMaster()
	if err != nil {
		return 0, err
	}

	return client.ClearEmergency(serviceID)
}
