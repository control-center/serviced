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
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/client"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/zenoss/glog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// An instance of the control plane Agent.
type HostAgent struct {
	master          string               // the connection string to the master agent
	hostId          string               // the hostID of the current host
	currentServices map[string]*exec.Cmd // the current running services
	mux             proxy.TCPMux
}

// assert that this implemenents the Agent interface
var _ serviced.Agent = &HostAgent{}

// Create a new HostAgent given the connection string to the
func NewHostAgent(master string, mux proxy.TCPMux) (agent *HostAgent, err error) {
	agent = &HostAgent{}
	agent.master = master
	agent.mux = mux
	hostId, err := serviced.HostId()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)
	if agent.mux.Enabled {
		go agent.mux.ListenAndMux()
	}

	go agent.start()
	return agent, err
}

// Use the Context field of the given template to fill in all the templates in
// the Command fields of the template's ServiceDefinitions
func injectContext(s *dao.Service) error {
	if len(s.Context) == 0 {
		return nil
	}

  glog.Infof( "%s", s.Context)
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(s.Context), &ctx); err != nil {
		return err
	}

  glog.Infof( "%+v", ctx)
	t := template.Must(template.New(s.Name).Parse(s.Startup))
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return err
	}
	s.Startup = buf.String()

	return nil
}

// Update the current state of a service. client is the ControlPlane client,
// service is the reference to the service being updated, and serviceState is
// the actual service instance being updated.
func (a *HostAgent) updateCurrentState(controlClient *client.ControlClient, service *dao.Service, serviceState *dao.ServiceState) (err error) {
	// get docker status

	containerState, err := getDockerState(serviceState.DockerId)

	markTerminated := func() error {
		serviceState.Terminated = time.Now()
		var unused int
		return controlClient.UpdateServiceState(*serviceState, &unused)
	}

	switch {
	case err == nil && !containerState.State.Running:
		err = markTerminated()
	case err != nil && strings.HasPrefix(err.Error(), "no container"):
		err = markTerminated()
	}
	return err
}

// Terminate a particular service instance (serviceState) on the localhost.
func (a *HostAgent) terminateInstance(controlClient *client.ControlClient, service *dao.Service, serviceState *dao.ServiceState) (err error) {
	// get docker status

	cmd := exec.Command("docker", "kill", serviceState.DockerId)
	err = cmd.Run()
	if err != nil {
		glog.Errorf("problem killing container instance %s", serviceState.DockerId)
		return err
	}
	serviceState.Terminated = time.Now()
	var unused int
	err = controlClient.UpdateServiceState(*serviceState, &unused)
	if err != nil {
		glog.Errorf("Problem updating service state: %v", err)
	}
	return
}

// Get the state of the docker container given the dockerId
func getDockerState(dockerId string) (containerState serviced.ContainerState, err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", dockerId)
	output, err := cmd.Output()
	if err != nil {
		glog.Errorln("problem getting docker state")
		return containerState, err
	}
	var containerStates []serviced.ContainerState
	err = json.Unmarshal(output, &containerStates)
	if err != nil {
		glog.Errorf("bad state  happened: %v,   \n\n\n%s", err, string(output))
		return containerState, dao.ControlPlaneError{"no state"}
	}
	if len(containerStates) < 1 {
		return containerState, dao.ControlPlaneError{"no container"}
	}
	return containerStates[0], err
}

// Get the path to the currently running executable.
func execPath() (string, string, error) {
	path, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return "", "", err
	}
	return filepath.Dir(path), filepath.Base(path), nil
}

// Start a service instance and update the CP with the state.
func (a *HostAgent) startService(controlClient *client.ControlClient, service *dao.Service, serviceState *dao.ServiceState) (err error) {

	glog.Infof("About to start service %s with name %s", service.Id, service.Name)
	portOps := ""
	if service.Endpoints != nil {
		glog.Infof("Endpoints for service: %v", service.Endpoints)
		for _, endpoint := range *service.Endpoints {
			if endpoint.Purpose == "export" { // only expose remote endpoints
				portOps += fmt.Sprintf(" -p %d", endpoint.PortNumber)
			}
		}
	}

	dir, binary, err := execPath()
	if err != nil {
		glog.Errorf("Error getting exec path: %v", err)
		return err
	}
	volumeBinding := fmt.Sprintf("%s:/serviced", dir)

	if err := injectContext(service); err != nil {
		glog.Errorf("Error injecting context: %s", err)
		return err
	}

	proxyCmd := fmt.Sprintf("/serviced/%s proxy %s '%s'", binary, service.Id, service.Startup)

	cmdString := fmt.Sprintf("docker run %s -d -v %s %s %s", portOps, volumeBinding, service.ImageId, proxyCmd)

	glog.Infof("Starting: %s", cmdString)

	cmd := exec.Command("bash", "-c", cmdString)
	output, err := cmd.Output()
	if err != nil {
		glog.Errorf("Problem starting service: %v, %s", err, string(output))
		return err
	}
	dockerId := strings.TrimSpace(string(output))
	serviceState.DockerId = dockerId
	containerState, err := getDockerState(dockerId)
	if err != nil {
		glog.Errorf("Problem getting service state :%v", err)
		return err
	}
	serviceState.Started = time.Now()
	serviceState.PrivateIp = containerState.NetworkSettings.IPAddress
	serviceState.PortMapping = containerState.NetworkSettings.PortMapping
	var unused int
	err = controlClient.UpdateServiceState(*serviceState, &unused)
	if err != nil {
		glog.Errorf("Problem updating service state: %v", err)
	}
	return err
}

func (a *HostAgent) handleServiceStatesForService(service *dao.Service, hostId string, controlClient *client.ControlClient) (err error) {
	// find current service states defined on the master
	var serviceStates []*dao.ServiceState
	err = controlClient.GetServiceStates(service.Id, &serviceStates)
	if err != nil {
		if strings.Contains(err.Error(), "Not found") {
			return nil
		}
		glog.Errorf("Got an error getting service states: %v", err)
		return err
	}
	for _, serviceInstance := range serviceStates {
		if serviceInstance.HostId != hostId {
			continue
		}
		switch {
		case serviceInstance.Started.Year() <= 1:
			err = a.startService(controlClient, service, serviceInstance)
		case serviceInstance.Started.Year() > 1 && serviceInstance.Terminated.Year() <= 1:
			err = a.updateCurrentState(controlClient, service, serviceInstance)
		case serviceInstance.Started.Year() > 1 && serviceInstance.Terminated.Year() == 2:
			err = a.terminateInstance(controlClient, service, serviceInstance)
		}
		if err != nil {
			glog.Errorf("Problem servicing state %s,  %v", service.Name, err)
		}
	}
	return nil
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	glog.Infoln("Starting HostAgent")
	for {
		// create a wrapping function so that client.Close() can be handled via defer
		func() {
			controlClient, err := client.NewControlClient(a.master)
			if err != nil {
				glog.Errorf("Could not start ControlPlane client %v", err)
				return
			}
			defer controlClient.Close() /* this connection gets cleaned up when
			   the surrounding lamda is exited */
			for {
				time.Sleep(time.Second * 10)
				var services []*dao.Service
				// Get the services that should be running on this host
				err = controlClient.GetServicesForHost(a.hostId, &services)
				if err != nil {
					glog.Errorf("Could not get services for host %s: %v", a.hostId, err)
					break
				}

				if len(services) == 0 {
					glog.Infoln("No services are schedule to run on this host.")
				}

				// iterate over this host's services
				for _, service := range services {
					a.handleServiceStatesForService(service, a.hostId, controlClient)
				}
			}
		}()
	}
}

func (a *HostAgent) GetServiceEndpoints(serviceId string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
	controlClient, err := client.NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return
	}
	defer controlClient.Close()
	return controlClient.GetServiceEndpoints(serviceId, response)
}

// Create a Host object from the host this function is running on.
func (a *HostAgent) GetInfo(unused int, host *dao.Host) error {
	hostInfo, err := serviced.CurrentContextAsHost("UNKNOWN")
	if err != nil {
		return err
	}
	*host = *hostInfo
	return nil
}
