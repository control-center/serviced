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

	glog.Infof("Context: %s", s.Context)
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(s.Context), &ctx); err != nil {
		return err
	}

	glog.Infof("Context: %+v", ctx)
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
func (a *HostAgent) updateCurrentState(conn *zk.Conn, ssStats *zk.Stat, hssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState, hss *zzk.HostServiceState) (bool, error) {

	// get docker status
	containerState, err := getDockerState(serviceState.DockerId)
	switch {
	case err == nil && containerState.State.Running && hss.DesiredState == dao.SVC_STOP:
		glog.Infof("Host service state should stop: %s", service.Name)
		a.terminateInstance(conn, ssStats, hssStats, service, serviceState, hss)
		return true, nil
	case err == nil && !containerState.State.Running:
		glog.Infof("Container does not appear to be running: %s", service.Name)
		markTerminated(conn, ssStats, hssStats, service, serviceState, hss)
		return true, nil
	case err != nil && strings.HasPrefix(err.Error(), "no container"):
		glog.Warningf("Error retrieving container state: %s", service.Name)
		markTerminated(conn, ssStats, hssStats, service, serviceState, hss)
		return true, err
	}
	return false, err
}

func markTerminated(conn *zk.Conn, ssStats *zk.Stat, hssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState, hss *zzk.HostServiceState) {
	ssPath := zzk.ServiceStatePath(service.Id, serviceState.Id)
	err := conn.Delete(ssPath, ssStats.Version)
	if err != nil {
		glog.Warningf("Unable to delete service state %s", ssPath)
		// Nevermind... Keep going
	}
	hssPath := zzk.HostServiceStatePath(hss.HostId, serviceState.Id)
	err = conn.Delete(hssPath, hssStats.Version)
	if err != nil {
		glog.Warningf("Unable to delete host service state %s", hssPath)
	}
}

// Terminate a particular service instance (serviceState) on the localhost.
func (a *HostAgent) terminateInstance(conn *zk.Conn, ssStats *zk.Stat, hssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState, hss *zzk.HostServiceState) error {
	err := a.dockerTerminate(serviceState.DockerId)
	if err != nil {
		return err
	}
	markTerminated(conn, ssStats, hssStats, service, serviceState, hss)
	return nil
}

func (a *HostAgent) dockerTerminate(dockerId string) error {
	// get docker status
	cmd := exec.Command("docker", "kill", dockerId)
	err := cmd.Run()
	if err != nil {
		glog.Errorf("problem killing container instance %s", dockerId)
		return err
	}
	return nil
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

			zzk.CreateNode(zzk.SCHEDULER_PATH, conn)
			node_path := zzk.HostPath(a.hostId)
			zzk.CreateNode(node_path, conn)
			glog.Infof("Connected to node %s", node_path)
			a.processChildrenAndWait(conn)
		}()
	}
}

func (a *HostAgent) processChildrenAndWait(conn *zk.Conn) {
	processing := make(map[string]chan int)
	ssDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		for _, shutdown := range processing {
			shutdown <- 1
		}
	}()

	for { // This loop keeps running until we get an error

		hostPath := zzk.HostPath(a.hostId)
		children, _, zkEvent, err := conn.ChildrenW(hostPath)
		if err != nil {
			glog.Info("Unable to read children, retrying.")
			time.Sleep(3 * time.Second)
			continue
		}
		glog.Infof("Agent for %s processing %d children", a.hostId, len(children))

		for _, childName := range children {
			if processing[childName] == nil {
				childChannel := make(chan int)
				processing[childName] = childChannel
				go a.processServiceState(conn, childChannel, ssDone, childName)
			}
		}

		select {
		case evt := <-zkEvent:
			glog.Infof("Agent event: %v", evt)
		case ssId := <-ssDone:
			glog.Infof("Agent cleaning up for service state %s", ssId)
			delete(processing, ssId)
		}
	}
}

func (a *HostAgent) processServiceState(conn *zk.Conn, shutdown <-chan int, done chan<- string, ssId string) {
	defer func() { done <- ssId }()
	failures := 0
	for {
		if failures >= 10 {
			glog.Errorf("Gave up trying to process %s", ssId)
			return
		}

		var hss zzk.HostServiceState
		hssStats, zkEvent, err := zzk.LoadHostServiceStateW(conn, a.hostId, ssId, &hss)
		if err != nil {
			glog.Errorf("Unable to load host service state %s: %v", ssId, err)
			return
		}
		if len(hss.ServiceStateId) == 0 || len(hss.ServiceId) == 0 {
			glog.Errorf("Service for %s is invalid", zzk.HostServiceStatePath(a.hostId, ssId))
			return
		}

		var ss dao.ServiceState
		ssStats, err := zzk.LoadServiceState(conn, hss.ServiceId, hss.ServiceStateId, &ss)
		if err != nil {
			glog.Errorf("Host service state unable to load service state %s", ssId)
			// TODO: TIDY this up!
			hssPath := zzk.HostServiceStatePath(a.hostId, ssId)
			err = conn.Delete(hssPath, hssStats.Version)
			if err != nil {
				glog.Warningf("Unable to delete host service state %s", hssPath)
			}

			return
		}

		var service dao.Service
		_, err = zzk.LoadService(conn, ss.ServiceId, &service)
		if err != nil {
			glog.Infof("Host service state unable to load service %s", ss.ServiceId)
			return
		}

		glog.Infof("Processing %s, desired state: %d", service.Name, hss.DesiredState)
		var term bool
		switch {

		// Not started and should start
		case ss.Started.Year() <= 1 && hss.DesiredState == dao.SVC_RUN:
			err = a.startService(conn, ssStats, &service, &ss)

		// Started and not marked as stopped
		case ss.Started.Year() > 1 && ss.Terminated.Year() <= 1:
			term, err = a.updateCurrentState(conn, ssStats, hssStats, &service, &ss, &hss)
			if term {
				glog.Infof("Service %s not running, quitting", service.Name)
				// If we marked the state as terminated, quit watching this node
				return
			}

		// This node is marked for death
		case hss.DesiredState == dao.SVC_STOP:
			glog.Info("Service %s was marked for death, quitting", service.Name)
			err = a.terminateInstance(conn, ssStats, hssStats, &service, &ss, &hss)
			return
		}

		if err != nil {
			glog.Errorf("Problem servicing state %s,  %v", service.Name, err)
			time.Sleep(10 * time.Second)
			failures += 1
			continue
		} else {
			failures = 0
		}

		select {

		case evt := <-zkEvent:
			if evt.Type == zk.EventNodeDeleted {
				glog.Infof("Shutting down due to node delete %s", ssId)
				err = a.terminateInstance(conn, ssStats, hssStats, &service, &ss, &hss)
				return
			}
			glog.Infof("Host service state %s received event %v", ssId, evt)
			continue

		case <-shutdown:
			glog.Info("Shutting down")
			return
		}
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
