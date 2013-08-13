/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package serviced

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type HostAgent struct {
	master          string
	hostId          string
	currentServices map[string]*exec.Cmd
}

// assert that this implemenents the Agent interface
var _ Agent = &HostAgent{}

// Create a new HostAgent.
func NewHostAgent(master string) (agent *HostAgent, err error) {
	agent = &HostAgent{}
	agent.master = master
	hostId, err := HostId()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)
	go agent.start()
	return agent, err
}

func (a *HostAgent) updateCurrentState(client *ControlClient, service *Service, serviceState *ServiceState) (err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", serviceState.DockerId)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("problem getting docker state")
		return err
	}
	containerStates := make([]*ContainerState, 0)
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
		err = client.UpdateServiceState(*serviceState, &unused)
	}
	return
}

func (a *HostAgent) terminateInstance(client *ControlClient, service *Service, serviceState *ServiceState) (err error) {
	// get docker status

	cmd := exec.Command("docker", "kill", serviceState.DockerId)
	err = cmd.Run()
	if err != nil {
		log.Printf("problem killing container instance %s", serviceState.DockerId)
		return err
	}
	serviceState.Terminated = time.Now()
	var unused int
	err = client.UpdateServiceState(*serviceState, &unused)
	if err != nil {
		log.Printf("Problem updating service state: %v", err)
	}
	return
}

func getDockerState(dockerId string) (containerState ContainerState, err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", dockerId)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("problem getting docker state")
		return containerState, err
	}
	containerStates := make([]*ContainerState, 0)
	err = json.Unmarshal(output, &containerStates)
	if len(containerStates) < 1 {
		log.Printf("bad state  happened: %v,   \n\n\n%s", err, string(output))
		return containerState, ControlPlaneError{"no state"}
	}
	return *containerStates[0], err
}

func (a *HostAgent) startService(client *ControlClient, service *Service, serviceState *ServiceState) (err error) {

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
	err = client.UpdateServiceState(*serviceState, &unused)
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
			client, err := NewControlClient(a.master)
			if err != nil {
				log.Printf("Could not start ControlPlane client %v", err)
				return
			}
			defer client.Close()
			for {
				time.Sleep(time.Second * 10)
				var services []*Service
				err = client.GetServicesForHost(a.hostId, &services)
				if err != nil {
					log.Printf("Could not get services for this host")
					break
				}
				for _, service := range services {
					var serviceStates []*ServiceState
					err = client.GetServiceStates(service.Id, &serviceStates)
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
							err = a.startService(client, service, serviceInstance)
						case serviceInstance.Started.Year() > 1 && serviceInstance.Terminated.Year() <= 1:
							err = a.updateCurrentState(client, service, serviceInstance)
						case serviceInstance.Started.Year() > 1 && serviceInstance.Terminated.Year() == 2:
							err = a.terminateInstance(client, service, serviceInstance)
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

func addRoute(network, netmask, gateway string) (err error) {

	log.Printf("executing: route add -net %s netmask %s gw %s", network, netmask, gateway)
	out, err := exec.Command("route", "add", "-net", network,
		"netmask", netmask, "gw", gateway).CombinedOutput()
	log.Printf("output from route: %s", string(out))
	return err
}

func (a *HostAgent) GetInfo(unused int, host *Host) error {
	hostInfo, err := CurrentContextAsHost()
	if err != nil {
		return err
	}
	*host = *hostInfo
	return nil
}
