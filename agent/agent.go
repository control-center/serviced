/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.
package agent

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/client"
	"log"
	"os/exec"
	"strings"
	"time"
)

// An instance of the control plane Agent.
type HostAgent struct {
	master          string               // the connection string to the master agent
	hostId          string               // the hostID of the current host
	currentServices map[string]*exec.Cmd // the current running services
}

// assert that this implemenents the Agent interface
var _ serviced.Agent = &HostAgent{}

// Create a new HostAgent given the connection string to the
func NewHostAgent(master string) (agent *HostAgent, err error) {
	agent = &HostAgent{}
	agent.master = master
	hostId, err := serviced.HostId()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)
	go agent.start()
	return agent, err
}

// Update the current state of a service. client is the ControlPlane client,
// service is the reference to the service being updated, and serviceState is
// the actual service instance being updated.
func (a *HostAgent) updateCurrentState(controlClient *client.ControlClient, service *serviced.Service, serviceState *serviced.ServiceState) (err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", serviceState.DockerId)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("problem getting docker state")
		return err
	}
	containerStates := make([]*serviced.ContainerState, 0)
	err = json.Unmarshal(output, &containerStates)
	if err != nil {
		log.Printf("updateCurrentState: there was a problem unmarshalling docker output: %v", err)
		return err
	}
	if len(containerStates) < 1 {
		log.Printf("Problem getting docker state")
		return err
	}
	containerState := containerStates[0]
	if !containerState.State.Running {
		serviceState.Terminated = time.Now()
		var unused int
		err = controlClient.UpdateServiceState(*serviceState, &unused)
	}
	return
}

// Terminate a particular service instance (serviceState) on the localhost.
func (a *HostAgent) terminateInstance(controlClient *client.ControlClient, service *serviced.Service, serviceState *serviced.ServiceState) (err error) {
	// get docker status

	cmd := exec.Command("docker", "kill", serviceState.DockerId)
	err = cmd.Run()
	if err != nil {
		log.Printf("problem killing container instance %s", serviceState.DockerId)
		return err
	}
	serviceState.Terminated = time.Now()
	var unused int
	err = controlClient.UpdateServiceState(*serviceState, &unused)
	if err != nil {
		log.Printf("Problem updating service state: %v", err)
	}
	return
}

// Get the state of the docker container.
func getDockerState(dockerId string) (containerState serviced.ContainerState, err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", dockerId)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("problem getting docker state")
		return containerState, err
	}
	containerStates := make([]*serviced.ContainerState, 0)
	err = json.Unmarshal(output, &containerStates)
	if len(containerStates) < 1 {
		log.Printf("bad state  happened: %v,   \n\n\n%s", err, string(output))
		return containerState, serviced.ControlPlaneError{"no state"}
	}
	return *containerStates[0], err
}

// Start a service instance and update the CP with the state.
func (a *HostAgent) startService(controlClient *client.ControlClient, service *serviced.Service, serviceState *serviced.ServiceState) (err error) {

	cmdString := fmt.Sprintf("docker run -d %s %s", service.ImageId, service.Startup)
	log.Printf("Starting: %s", cmdString)

	cmd := exec.Command("bash", "-c", cmdString)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Problem starting service: %v, %s", err, string(output))
		return err
	}
	dockerId := strings.TrimSpace(string(output))
	serviceState.DockerId = dockerId
	containerState, err := getDockerState(dockerId)
	if err != nil {
		log.Printf("Problem getting service state :%v", err)
		return err
	}
	serviceState.Started = time.Now()
	serviceState.PrivateIp = containerState.NetworkSettings.IPAddress
	var unused int
	err = controlClient.UpdateServiceState(*serviceState, &unused)
	if err != nil {
		log.Printf("Problem updating service state: %v", err)
	}
	return err
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	log.Printf("Starting HostAgent\n")
	for {
		// create a wrapping function so that client.Close() can be handled via defer
		func() {
			controlClient, err := client.NewControlClient(a.master)
			if err != nil {
				log.Printf("Could not start ControlPlane client %v", err)
				return
			}
			defer controlClient.Close()
			for {
				time.Sleep(time.Second * 10)
				var services []*serviced.Service
				err = controlClient.GetServicesForHost(a.hostId, &services)
				if err != nil {
					log.Printf("Could not get services for this host")
					break
				}
				for _, service := range services {
					var serviceStates []*serviced.ServiceState
					err = controlClient.GetServiceStates(service.Id, &serviceStates)
					if err != nil {
						if strings.Contains(err.Error(), "Not found") {
							continue
						}
						log.Printf("Got an error getting service states: %v", err)
						break
					}
					for _, serviceInstance := range serviceStates {
						switch {
						case serviceInstance.Started.Year() <= 1:
							err = a.startService(controlClient, service, serviceInstance)
						case serviceInstance.Started.Year() > 1 && serviceInstance.Terminated.Year() <= 1:
							err = a.updateCurrentState(controlClient, service, serviceInstance)
						case serviceInstance.Started.Year() > 1 && serviceInstance.Terminated.Year() == 2:
							err = a.terminateInstance(controlClient, service, serviceInstance)
						}
						if err != nil {
							log.Printf("Problem servicing state %s,  %v", service.Name, err)
						}
					}
				}
			}
		}()
	}
}

// Create a Host object from the host this function is running on.
func (a *HostAgent) GetInfo(unused int, host *serviced.Host) error {
	hostInfo, err := serviced.CurrentContextAsHost("UNKNOWN")
	if err != nil {
		return err
	}
	*host = *hostInfo
	return nil
}
