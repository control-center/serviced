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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

/*
 glog levels:
 0: important info that should always be shown
 1: info that might be important for debugging
 2: very verbose debug info
 3: trace level info
*/

// An instance of the control plane Agent.
type HostAgent struct {
	master          string               // the connection string to the master agent
	hostId          string               // the hostID of the current host
	resourcePath    string               // directory to bind mount docker volumes
	currentServices map[string]*exec.Cmd // the current running services
	zookeepers      []string
	mux             proxy.TCPMux
	shutdown        chan int // Used to shutdown gracefully
	finished        chan int // Signal when finished
}

// assert that this implemenents the Agent interface
var _ serviced.Agent = &HostAgent{}

// Create a new HostAgent given the connection string to the

func NewHostAgent(master string, resourcePath string, mux proxy.TCPMux, zookeepers []string) (*HostAgent, error) {
	agent := &HostAgent{}
	agent.master = master
	agent.mux = mux
	agent.resourcePath = resourcePath
	agent.shutdown = make(chan int, 1)
	agent.finished = make(chan int, 1)


	hostId, err := serviced.HostId()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)
	agent.zookeepers = zookeepers
	if len(agent.zookeepers) == 0 {
		defaultZK := "127.0.0.1:2181"
		glog.V(1).Infoln("Zookeepers not specified: using default of ", defaultZK)
		agent.zookeepers = []string{defaultZK}
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
		glog.V(1).Infoln("Context was empty")
		return nil
	}

	glog.V(2).Infof("Context string: %s", s.Context)
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(s.Context), &ctx); err != nil {
		return err
	}

	glog.V(2).Infof("Context unmarshalled: %+v", ctx)

	t := template.Must(template.New(s.Name).Parse(s.Startup))
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return err
	}
	s.Startup = buf.String()
	glog.V(2).Infof("injectContext set startup to '%s'", s.Startup)

	return nil
}

func (a *HostAgent) Shutdown() {
	glog.V(2).Info("Issuing shutdown signal")
	a.shutdown <- 1
	glog.V(2).Info("Waiting for all done signal")
	<-a.finished
	glog.V(1).Info("Agent shutdown complete")
}

// Attempts to attach to a running container
func (a *HostAgent) attachToService(conn *zk.Conn, procFinished chan<- int, serviceState *dao.ServiceState, hss *zzk.HostServiceState) (bool, error) {

	// get docker status
	containerState, err := getDockerState(serviceState.DockerId)
	glog.V(2).Infof("Agent.updateCurrentState got container state for docker ID %s: %v", serviceState.DockerId, containerState)

	switch {
	case err == nil && !containerState.State.Running:
		glog.V(1).Infof("Container does not appear to be running: %s", serviceState.Id)
		return false, errors.New("Container not running for " + serviceState.Id)

	case err != nil && strings.HasPrefix(err.Error(), "no container"):
		glog.Warningf("Error retrieving container state: %s", serviceState.Id)
		return false, err

	}

	cmd := exec.Command("docker", "attach", serviceState.DockerId)
	go waitForProcessToDie(conn, cmd, procFinished, serviceState)
	return true, nil
}

func markTerminated(conn *zk.Conn, hss *zzk.HostServiceState) {
	ssPath := zzk.ServiceStatePath(hss.ServiceId, hss.ServiceStateId)
	_, stats, err := conn.Get(ssPath)
	if err != nil {
		glog.V(0).Infof("Unable to get service state %s for delete because: %v", ssPath, err)
		return
	}
	err = conn.Delete(ssPath, stats.Version)
	if err != nil {
		glog.V(0).Infof("Unable to delete service state %s because: %v", ssPath, err)
		return
	}

	hssPath := zzk.HostServiceStatePath(hss.HostId, hss.ServiceStateId)
	_, stats, err = conn.Get(hssPath)
	if err != nil {
		glog.V(0).Infof("Unable to get host service state %s for delete becaus: %v", hssPath, err)
		return
	}
	err = conn.Delete(hssPath, stats.Version)
	if err != nil {
		glog.V(0).Infof("Unable to delete host service state %s", hssPath)
	}
}

// Terminate a particular service instance (serviceState) on the localhost.
func (a *HostAgent) terminateInstance(conn *zk.Conn, serviceState *dao.ServiceState) error {
	err := a.dockerTerminate(serviceState.DockerId)
	if err != nil {
		return err
	}
	markTerminated(conn, zzk.SsToHss(serviceState))
	return nil
}

func (a *HostAgent) terminateAttached(conn *zk.Conn, procFinished <-chan int, ss *dao.ServiceState) error {
	err := a.dockerTerminate(ss.DockerId)
	if err != nil {
		return err
	}
	<-procFinished
	markTerminated(conn, zzk.SsToHss(ss))
	return nil
}

func (a *HostAgent) dockerTerminate(dockerId string) error {
	// get docker status
	cmd := exec.Command("docker", "kill", dockerId)
	glog.V(1).Infof("dockerTerminate: %s", dockerId)
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

func waitForProcessToDie(conn *zk.Conn, cmd *exec.Cmd, procFinished chan<- int, serviceState *dao.ServiceState) {
	tmpName := os.TempDir() + "/" + serviceState.Id + ".log"
	defer func() {
		err := os.Remove(tmpName)
		if err != nil {
			glog.V(1).Infof("Unable to remove tmp file %s: %v", err)
		}
		procFinished <- 1
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		glog.Warningf("Unable to read standard out for service state %s: %v", serviceState.Id, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		glog.Warningf("Unable to read standard error for service state %s: %v", serviceState.Id, err)
	}

	tmpLog, err := os.Create(tmpName)
	if err != nil {
		glog.Warningf("Unable to create temp file %s", tmpName)
	}

	err = cmd.Start()

	go io.Copy(tmpLog, stdout)
	go io.Copy(tmpLog, stderr)

	if err != nil {
		glog.Errorf("Problem starting command '%s %s': %v", cmd.Path, cmd.Args, err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		glog.V(0).Infof("Docker process did not exit cleanly: %v", err)
		out, err := ioutil.ReadFile(tmpName)
		if err != nil {
			glog.V(1).Infof("Unable to read file %s", tmpName)
		} else {
			glog.V(0).Infof("Process out:\n%s", out)
		}
	} else {
		glog.V(0).Infof("Process for service state %s finished", serviceState.Id)
	}

	err = zzk.ResetServiceState(conn, serviceState.ServiceId, serviceState.Id)
	if err != nil {
		glog.Errorf("Caught error marking process termination time for %s: %v", serviceState.Id, err)
	}
}

// Start a service instance and update the CP with the state.
func (a *HostAgent) startService(conn *zk.Conn, procFinished chan<- int, ssStats *zk.Stat, service *dao.Service, serviceState *dao.ServiceState) (bool, error) {
	glog.V(2).Infof("About to start service %s with name %s", service.Id, service.Name)
	portOps := ""
	if service.Endpoints != nil {
		glog.V(1).Info("Endpoints for service: ", service.Endpoints)
		for _, endpoint := range service.Endpoints {
			if endpoint.Purpose == "export" { // only expose remote endpoints
				portOps += fmt.Sprintf(" -p %d", endpoint.PortNumber)
			}
		}
	}

	volumeOpts := ""
	for _, volume := range service.Volumes {
		fileMode := os.FileMode(volume.Permission)
		resourcePath, _ := filepath.Abs(a.resourcePath + "/" + volume.ResourcePath)
		err := serviced.CreateDirectory(resourcePath, volume.Owner, fileMode)
		if err == nil {
			volumeOpts += fmt.Sprintf(" -v %s:%s", resourcePath, volume.ContainerPath)
		} else {
			glog.Errorf("Error creating resource path: %v", err)
			return false, err
		}
	}

	dir, binary, err := serviced.ExecPath()
	if err != nil {
		glog.Errorf("Error getting exec path: %v", err)
		return false, err
	}
	volumeBinding := fmt.Sprintf("%s:/serviced", dir)

	if err := injectContext(service); err != nil {
		glog.Errorf("Error injecting context: %s", err)
		return false, err
	}

	proxyCmd := fmt.Sprintf("/serviced/%s proxy %s '%s'", binary, service.Id, service.Startup)
	cmdString := fmt.Sprintf("docker run %s -rm -name=%s -v %s %s %s %s", portOps, serviceState.Id, volumeBinding, volumeOpts, service.ImageId, proxyCmd)
	glog.V(0).Infof("Starting: %s", cmdString)

	cmd := exec.Command("bash", "-c", cmdString)

	go waitForProcessToDie(conn, cmd, procFinished, serviceState)

	time.Sleep(1 * time.Second) // Sleep just a second to give it a chance to get a PID

	// We are name the container the same as its service state ID, so use that as an alias
	dockerId := serviceState.Id
	serviceState.DockerId = dockerId
	containerState, err := getDockerState(dockerId)
	if err != nil {
		glog.Errorf("Problem getting service state :%v", err)
		a.dockerTerminate(serviceState.DockerId)
		return false, err
	}
	serviceState.Started = time.Now()
	serviceState.Terminated = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
	serviceState.PrivateIp = containerState.NetworkSettings.IPAddress
	serviceState.PortMapping = containerState.NetworkSettings.Ports

	glog.V(1).Infof("SSPath: %s, PortMapping: %v", zzk.ServiceStatePath(service.Id, serviceState.Id), serviceState.PortMapping)

	ssBytes, err := json.Marshal(serviceState)
	if err != nil {
		glog.Errorf("Unable to marshal service state: %s", serviceState.Id)
		a.dockerTerminate(serviceState.DockerId)
		return false, err
	}

	ssPath := zzk.ServiceStatePath(service.Id, serviceState.Id)
	_, err = conn.Set(ssPath, ssBytes, ssStats.Version)
	if err != nil {
		glog.Errorf("Unable to save updated service state: %v", err)
		a.dockerTerminate(serviceState.DockerId)
		return false, err
	}
	return true, nil
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	glog.V(1).Info("Starting HostAgent")
	defer func() {
		glog.V(1).Info("Agent finally exiting")
		a.finished <- 1
	}()
	for {
		// create a wrapping function so that client.Close() can be handled via defer
		keepGoing := func() bool {
			conn, _, err := zk.Connect(a.zookeepers, time.Second*10)
			if err != nil {
				glog.V(0).Info("Unable to connect, retrying.")
				time.Sleep(time.Second * 3)
				return true
			}
			defer conn.Close() // Executed after lambda function finishes

			zzk.CreateNode(zzk.SCHEDULER_PATH, conn)
			node_path := zzk.HostPath(a.hostId)
			zzk.CreateNode(node_path, conn)
			glog.V(0).Infof("Connected to zookeeper node %s", node_path)
			return a.processChildrenAndWait(conn)
		}()
		if !keepGoing {
			break
		}
	}
}

func (a *HostAgent) processChildrenAndWait(conn *zk.Conn) bool {
	processing := make(map[string]chan int)
	ssDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		glog.V(0).Info("Agent shutting down child goroutines.")
		for key, shutdown := range processing {
			glog.V(1).Infof("Agent signaling for %s to shutdown.", key)
			shutdown <- 1
		}

		// Wait for goroutines to shutdown
		for len(processing) > 0 {
			select {
			case ssId := <-ssDone:
				glog.V(1).Info("Agent cleaning up for service state ", ssId)
				delete(processing, ssId)
			}
		}
	}()

	for { // This loop keeps running until we get an error

		hostPath := zzk.HostPath(a.hostId)
		children, _, zkEvent, err := conn.ChildrenW(hostPath)
		if err != nil {
			glog.V(0).Infoln("Unable to read children, retrying.")
			time.Sleep(3 * time.Second)
			return true
		}
		glog.V(1).Infof("Agent for %s processing %d children", a.hostId, len(children))

		for _, childName := range children {
			if processing[childName] == nil {
				glog.V(2).Info("Agent starting goroutine to watch ", childName)
				childChannel := make(chan int)
				processing[childName] = childChannel
				go a.processServiceState(conn, childChannel, ssDone, childName)
			}
		}

		select {
		case evt := <-zkEvent:
			glog.V(1).Info("Agent event: ", evt)
		case ssId := <-ssDone:
			glog.V(1).Info("Agent cleaning up for service state ", ssId)
			delete(processing, ssId)
		case <-a.shutdown:
			glog.V(1).Info("Agent received interrupt")
			return false
		}
	}
}

func (a *HostAgent) processServiceState(conn *zk.Conn, shutdown <-chan int, done chan<- string, ssId string) {
	defer func() {
		glog.V(3).Info("Exiting function processServiceState ", ssId)
		done <- ssId
	}()
	failures := 0
	procFinished := make(chan int, 1)

	var attached bool
	for {
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
			// This goroutine is watching a node for a service state that does not
			// exist or could not be loaded. We should *probably* delete this node.
			hssPath := zzk.HostServiceStatePath(a.hostId, ssId)
			err = conn.Delete(hssPath, hssStats.Version)
			if err != nil {
				glog.Warningf("Unable to delete host service state %s", hssPath)
			}
			return
		}

		if failures >= 5 {
			glog.V(0).Infof("Gave up trying to process %s", ssId)
			a.terminateInstance(conn, &ss)
			return
		}

		var service dao.Service
		_, err = zzk.LoadService(conn, ss.ServiceId, &service)
		if err != nil {
			glog.Errorf("Host service state unable to load service %s", ss.ServiceId)
			return
		}

		glog.V(1).Infof("Processing %s, desired state: %d", service.Name, hss.DesiredState)

		switch {

		// This node is marked for death
		case hss.DesiredState == dao.SVC_STOP:
			glog.V(1).Infof("Service %s was marked for death, quitting", service.Name)
			if attached {
				err = a.terminateAttached(conn, procFinished, &ss)
			} else {
				err = a.terminateInstance(conn, &ss)
			}
			return

		case attached:
			glog.V(1).Infof("Service %s is attached in a child goroutine", service.Name)

		// Should run, and either not started or process died
		case hss.DesiredState == dao.SVC_RUN &&
			ss.Started.Year() <= 1 || ss.Terminated.Year() > 2:
			glog.V(1).Infof("Service %s does not appear to be running; starting", service.Name)
			attached, err = a.startService(conn, procFinished, ssStats, &service, &ss)

		// Service superficially seems to be running. We need to attach
		case ss.Started.Year() > 1 && ss.Terminated.Year() <= 1:
			glog.V(1).Infof("Service %s appears to be running; attaching", service.Name)
			attached, err = a.attachToService(conn, procFinished, &ss, &hss)

		default:
			glog.V(0).Infof("Unhandled service %s", service.Name)
		}

		if err != nil || !attached {
			glog.Errorf("Problem servicing state %s,  %v", service.Name, err)
			time.Sleep(10 * time.Second)
			failures += 1
			continue
		}

		glog.V(3).Infoln("Successfully processed state for %s", service.Name)
		failures = 0

		select {

		case evt := <-zkEvent:
			if evt.Type == zk.EventNodeDeleted {
				glog.V(0).Info("Host service state deleted: ", ssId)
				err = a.terminateAttached(conn, procFinished, &ss)
				if err != nil {
					glog.Errorf("Error terminating %s: %v", service.Name, err)
				}
				return
			}
			glog.V(1).Infof("Host service state %s received event %v", ssId, evt)
			continue

		case <-procFinished:
			glog.V(1).Infof("Process finished")
			attached = false
			continue

		case <-shutdown:
			glog.V(0).Info("Agent goroutine will stop watching ", ssId)
			err = a.terminateAttached(conn, procFinished, &ss)
			if err != nil {
				glog.Errorf("Error terminating %s: %v", service.Name, err)
			}

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
