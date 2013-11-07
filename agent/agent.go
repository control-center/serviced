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
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/zzk"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zenoss/glog"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

// An instance of the control plane Agent.
type HostAgent struct {
	master          string               // the connection string to the master agent
	hostId          string               // the hostID of the current host
	currentServices map[string]*exec.Cmd // the current running services
	zookeepers      []string
	mux             proxy.TCPMux
}

// assert that this implemenents the Agent interface
var _ serviced.Agent = &HostAgent{}

// Create a new HostAgent given the connection string to the
func NewHostAgent(master string, mux proxy.TCPMux, zookeepers []string) (agent *HostAgent, err error) {
	agent = &HostAgent{}
	agent.master = master
	agent.mux = mux
	hostId, err := serviced.HostId()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)
	agent.zookeepers = zookeepers
	if len(agent.zookeepers) == 0 {
		agent.zookeepers = []string{"127.0.0.1:2181"}
	}

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

	glog.Infof("%s", s.Context)
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(s.Context), &ctx); err != nil {
		return err
	}

	glog.Infof("%+v", ctx)
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
func (a *HostAgent) updateCurrentState(conn *zk.Conn, ssStats *zk.Stat, hssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState, hss *zzk.HostServiceState) (err error) {

	// get docker status
	containerState, err := getDockerState(serviceState.DockerId)
	switch {
	case err == nil && !containerState.State.Running:
		err = markTerminated(conn, ssStats, hssStats, service, serviceState, hss)
	case err != nil && strings.HasPrefix(err.Error(), "no container"):
		err = markTerminated(conn, ssStats, hssStats, service, serviceState, hss)
	}
	return err
}

func markTerminated(conn *zk.Conn, ssStats *zk.Stat, hssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState, hss *zzk.HostServiceState) error {
	ssPath := zzk.ServiceStatePath(service.Id, serviceState.Id)
	err := conn.Delete(ssPath, ssStats.Version)
	if err != nil {
		glog.Errorf("Unable to delete service state %s", ssPath)
		return err
	}
	hssPath := zzk.HostServiceStatePath(hss.HostId, serviceState.Id)
	err = conn.Delete(hssPath, hssStats.Version)
	if err != nil {
		glog.Errorf("Unable to delete host service state %s", hssPath)
		return err
	}
	return err
}

// Terminate a particular service instance (serviceState) on the localhost.
func (a *HostAgent) terminateInstance(conn *zk.Conn, ssStats *zk.Stat, hssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState, hss *zzk.HostServiceState) (err error) {
	// get docker status
	cmd := exec.Command("docker", "kill", serviceState.DockerId)
	err = cmd.Run()
	if err != nil {
		glog.Errorf("problem killing container instance %s", serviceState.DockerId)
		return err
	}
	return markTerminated(conn, ssStats, hssStats, service, serviceState, hss)
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

// Start a service instance and update the CP with the state.
func (a *HostAgent) startService(conn *zk.Conn, ssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState) (err error) {
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

	dir, binary, err := serviced.ExecPath()
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
	ssBytes, err := json.Marshal(serviceState)
	if err != nil {
		return err
	}

	ssPath := zzk.ServiceStatePath(service.Id, serviceState.Id)
	_, err = conn.Set(ssPath, ssBytes, ssStats.Version)
	return err
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	glog.Infoln("Starting HostAgent")
	for {
		// create a wrapping function so that client.Close() can be handled via defer
		func() {
			conn, _, err := zk.Connect(a.zookeepers, time.Second*10)
			if err != nil {
				glog.Info("Unable to connect, retrying.")
				time.Sleep(time.Second * 3)
				return
			}
			defer conn.Close() // Executed after lambda function finishes

			node_path := zzk.HostPath(a.hostId)
			zzk.CreateNode(node_path, conn)
			glog.Infof("Connected to node %s", node_path)

			for { // This loop keeps running until we get an error
				err = a.processChildrenAndWait(conn, node_path)
				if err != nil {
					time.Sleep(time.Second * 3)
					break // An error here forces reconnect
				}
			}
		}()
	}
}

func (a *HostAgent) processChildrenAndWait(conn *zk.Conn, node_path string) (err error) {
	children, _, event, err := conn.ChildrenW(node_path)
	if err != nil {
		glog.Info("Unable to read children, retrying.")
		return err
	}

	glog.Infof("Found %d children", len(children))
	for _, childName := range children {
		childPath := node_path + "/" + childName
		hostServiceStateNode, hssStats, err := conn.Get(childPath)
		if err != nil {
			glog.Errorf("Got error for %s: %v", childName, err)
			return err
		}
		var hss zzk.HostServiceState
		err = json.Unmarshal(hostServiceStateNode, &hss)
		if err != nil {
			glog.Errorf("Unable to unmarshal %s", childName)
			return err
		}
		// TODO: Create validateServiceState function
		if len(hss.ServiceStateId) == 0 || len(hss.ServiceId) == 0 {
			glog.Errorf("Service for %s is invalid", childPath)
			return errors.New("Invalid service")
		}

		serviceStatePath := zzk.ServiceStatePath(hss.ServiceId, hss.ServiceStateId)
		var serviceState dao.ServiceState
		ssNode, ssStats, err := conn.Get(serviceStatePath)
		err = json.Unmarshal(ssNode, &serviceState)
		if err != nil {
			glog.Errorf("Unable to unmarshal %s", serviceStatePath)
			return err
		}

		glog.Infof("Attempting to locate service %s", serviceState.ServiceId)
		servicePath := "/services/" + serviceState.ServiceId
		serviceNode, _, err := conn.Get(servicePath)
		if err != nil {
			glog.Errorf("Got error loading %s: %v", serviceState.ServiceId, err)
			return err
		}
		var service dao.Service
		err = json.Unmarshal(serviceNode, &service)
		if err != nil {
			glog.Errorf("Unable to unmarshal %s", serviceState.ServiceId)
			return err
		}

		switch {
		case serviceState.Started.Year() <= 1:
			err = a.startService(conn, ssStats, &service, &serviceState)
		case serviceState.Started.Year() > 1 && serviceState.Terminated.Year() <= 1:
			err = a.updateCurrentState(conn, ssStats, hssStats, &service, &serviceState, &hss)
		case serviceState.Started.Year() > 1 && hss.DesiredState == dao.SVC_STOP:
			err = a.terminateInstance(conn, ssStats, hssStats, &service, &serviceState, &hss)
		}
		if err != nil {
			glog.Errorf("Problem servicing state %s,  %v", service.Name, err)
		}
	}

	select {
	case evt := <-event:
		glog.Infof("Received event: %v", evt)
	}

	return nil
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
