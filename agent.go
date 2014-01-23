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
package serviced

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/zzk"

	"encoding/json"
	"errors"
	"fmt"
	"github.com/zenoss/glog"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
	master          string   // the connection string to the master agent
	hostId          string   // the hostID of the current host
	resourcePath    string   // directory to bind mount docker volumes
	mount           []string // each element is in the form: container_image:host_path:container_path
	zookeepers      []string
	currentServices map[string]*exec.Cmd // the current running services
	mux             TCPMux
	closing         chan chan error
}

// assert that this implemenents the Agent interface
var _ Agent = &HostAgent{}

// Create a new HostAgent given the connection string to the

func NewHostAgent(master string, resourcePath string, mount []string, zookeepers []string, mux TCPMux) (*HostAgent, error) {
	// save off the arguments
	agent := &HostAgent{}
	agent.master = master
	agent.resourcePath = resourcePath
	agent.mount = mount
	agent.zookeepers = zookeepers
	if len(agent.zookeepers) == 0 {
		defaultZK := "127.0.0.1:2181"
		glog.V(1).Infoln("Zookeepers not specified: using default of ", defaultZK)
		agent.zookeepers = []string{defaultZK}
	}
	agent.mux = mux
	if agent.mux.Enabled {
		go agent.mux.ListenAndMux()
	}

	agent.closing = make(chan chan error)
	hostId, err := HostId()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)

	go agent.start()
	return agent, err
}

// Use the Context field of the given template to fill in all the templates in
// the Command fields of the template's ServiceDefinitions
func injectContext(s *dao.Service, cp dao.ControlPlane) error {
	return s.EvaluateStartupTemplate(cp)
}

func (a *HostAgent) Shutdown() error {
	glog.V(2).Info("Issuing shutdown signal")
	errc := make(chan error)
	a.closing <- errc
	return <-errc
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
	go a.waitForProcessToDie(conn, cmd, procFinished, serviceState)
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
	err := a.dockerTerminate(serviceState.Id)
	if err != nil {
		return err
	}
	markTerminated(conn, zzk.SsToHss(serviceState))
	return nil
}

func (a *HostAgent) terminateAttached(conn *zk.Conn, procFinished <-chan int, ss *dao.ServiceState) error {
	err := a.dockerTerminate(ss.Id)
	if err != nil {
		return err
	}
	<-procFinished
	markTerminated(conn, zzk.SsToHss(ss))
	return nil
}

func (a *HostAgent) dockerRemove(dockerId string) error {
	glog.V(1).Infof("Ensuring that container %s does not exist", dockerId)
	cmd := exec.Command("docker", "rm", dockerId)
	err := cmd.Run()
	if err != nil {
		glog.V(1).Infof("problem removing container instance %s", dockerId)
		return err
	}
	glog.V(2).Infof("Successfully removed %s", dockerId)
	return nil
}

func (a *HostAgent) dockerTerminate(dockerId string) error {
	glog.V(1).Infof("Killing container %s", dockerId)
	cmd := exec.Command("docker", "kill", dockerId)
	err := cmd.Run()
	if err != nil {
		glog.V(1).Infof("problem killing container instance %s", dockerId)
		return err
	}
	glog.V(2).Infof("Successfully killed %s", dockerId)
	return nil
}

// Get the state of the docker container given the dockerId
func getDockerState(dockerId string) (containerState ContainerState, err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", dockerId)
	output, err := cmd.Output()
	if err != nil {
		glog.Errorln("problem getting docker state")
		return containerState, err
	}
	var containerStates []ContainerState
	err = json.Unmarshal(output, &containerStates)
	if err != nil {
		glog.Errorf("bad state	happened: %v,	\n\n\n%s", err, string(output))
		return containerState, dao.ControlPlaneError{"no state"}
	}
	if len(containerStates) < 1 {
		return containerState, dao.ControlPlaneError{"no container"}
	}
	return containerStates[0], err
}

func dumpOut(tmpName string) {
	out, err := ioutil.ReadFile(tmpName)
	if err != nil {
		glog.V(1).Infof("Unable to read file %s", tmpName)
	} else {
		glog.V(0).Infof("Process out:\n%s", out)
	}

}

func (a *HostAgent) waitForProcessToDie(conn *zk.Conn, cmd *exec.Cmd, procFinished chan<- int, serviceState *dao.ServiceState) {
	a.dockerRemove(serviceState.Id)

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
		glog.Errorf("Unable to read standard out for service state %s: %v", serviceState.Id, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		glog.Errorf("Unable to read standard error for service state %s: %v", serviceState.Id, err)
		return
	}

	tmpLog, err := os.Create(tmpName)
	if err != nil {
		glog.Errorf("Unable to create temp file %s", tmpName)
		return
	}

	err = cmd.Start()

	go io.Copy(tmpLog, stdout)
	go io.Copy(tmpLog, stderr)

	if err != nil {
		glog.Errorf("Problem starting command '%s %s': %v", cmd.Path, cmd.Args, err)
		dumpOut(tmpName)
		return
	}

	time.Sleep(1 * time.Second) // Sleep to give docker a chance to start

	// We are name the container the same as its service state ID, so use that as an alias
	dockerId := serviceState.Id
	serviceState.DockerId = dockerId
	containerState, err := getDockerState(dockerId)
	if err != nil {
		glog.Errorf("Problem getting service state :%v", err)
		a.dockerTerminate(dockerId)
		dumpOut(tmpName)
		return
	}

	err = zzk.LoadAndUpdateServiceState(conn, serviceState.ServiceId, serviceState.Id, func(ss *dao.ServiceState) {
		ss.DockerId = containerState.ID
		ss.Started = time.Now()
		ss.Terminated = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
		ss.PrivateIp = containerState.NetworkSettings.IPAddress
		ss.PortMapping = containerState.NetworkSettings.Ports
	})
	if err != nil {
		glog.Warningf("Unable to update service state %s: %v", serviceState.Id, err)
	}

	glog.V(1).Infof("SSPath: %s, PortMapping: %v", zzk.ServiceStatePath(serviceState.ServiceId, serviceState.Id), serviceState.PortMapping)

	err = cmd.Wait()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				statusCode := status.ExitStatus()
				switch {
				case statusCode == 137:
					glog.V(1).Infof("Docker process killed: %s", serviceState.Id)

				case statusCode == 2:
					glog.V(1).Infof("Docker process stopped: %s", serviceState.Id)

				default:
					glog.V(0).Infof("Docker process %s exited with code %d", serviceState.Id, statusCode)
					dumpOut(tmpName)
				}
			}
		} else {
			glog.V(1).Info("Unable to determine exit code for %s", serviceState.Id)
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
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return false, err
	}
	defer client.Close()

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
		err := CreateDirectory(resourcePath, volume.Owner, fileMode)
		if err == nil {
			volumeOpts += fmt.Sprintf(" -v %s:%s", resourcePath, volume.ContainerPath)
		} else {
			glog.Errorf("Error creating resource path: %v", err)
			return false, err
		}
	}

	dir, binary, err := ExecPath()
	if err != nil {
		glog.Errorf("Error getting exec path: %v", err)
		return false, err
	}
	volumeBinding := fmt.Sprintf("%s:/serviced", dir)

	if err := injectContext(service, client); err != nil {
		glog.Errorf("Error injecting context: %s", err)
		return false, err
	}

	// config files
	configFiles := ""
	for filename, config := range service.ConfigFiles {
		prefix := fmt.Sprintf("cp_%s_%s_", service.Id, strings.Replace(filename, "/", "__", -1))
		f, err := ioutil.TempFile("", prefix)
		if err != nil {
			glog.Errorf("Could not generate tempfile for config %s %s", service.Id, filename)
			return false, err
		}
		_, err = f.WriteString(config.Content)
		if err != nil {
			glog.Errorf("Could not write out config file %s %s", service.Id, filename)
			return false, err
		}
		if len(config.Owner) != 0 {
			parts := strings.Split(config.Owner, ":")
			if len(parts) != 2 {
				glog.Errorf("Unsupported owner specification, only %%d:%%d supported for now: %s, %s", service.Id, filename)
				continue
			}
			uid, err := strconv.Atoi(parts[0])
			if err != nil {
				glog.Warningf("Malformed UID: %s %s: %s", service.Id, filename, err)
				continue
			}
			gid, err := strconv.Atoi(parts[0])
			if err != nil {
				glog.Warningf("Malformed GID: %s %s: %s", service.Id, filename, err)
				continue
			}
			err = f.Chown(uid, gid)
			if err != nil {
				glog.Warningf("Could not chown config file: %s %s: %s", service.Id, filename, err)
			}
		}
		configFiles += fmt.Sprintf(" -v %s:%s ", f.Name(), filename)
	}

	// add arguments to mount requested directory (if requested)
	requestedMount := ""
	for _, bindMountString := range a.mount {
		splitMount := strings.Split(bindMountString, ":")
		if len(splitMount) == 3 {
			requestedImage := splitMount[0]
			hostPath := splitMount[1]
			containerPath := splitMount[2]
			if requestedImage == service.ImageId {
				requestedMount += " -v " + hostPath + ":" + containerPath
			}
		} else {
			glog.Warningf("Could not bind mount the following: %s", bindMountString)
		}
	}

	//get this service's tenantId for env injection
	var tenantId string
	err = client.GetTenantId(service.Id, &tenantId)
	if err != nil {
		glog.Errorf("Failed getting tenantId for service: %s, %s", service.Id, err)
	}

	// add arguments for environment variables
	environmentVariables := "-e CONTROLPLANE=1"
	environmentVariables = environmentVariables + " -e CONTROLPLANE_SERVICE_ID=" + service.Id
	environmentVariables = environmentVariables + " -e CONTROLPLANE_TENANT_ID=" + tenantId

	proxyCmd := fmt.Sprintf("/serviced/%s proxy %s '%s'", binary, service.Id, service.Startup)
	cmdString := fmt.Sprintf("docker run %s -rm -name=%s %s -v %s %s %s %s %s %s", portOps, serviceState.Id, environmentVariables, volumeBinding, requestedMount, volumeOpts, configFiles, service.ImageId, proxyCmd)

	glog.V(0).Infof("Starting: %s", cmdString)

	a.dockerTerminate(serviceState.Id)
	a.dockerRemove(serviceState.Id)

	cmd := exec.Command("bash", "-c", cmdString)

	go a.waitForProcessToDie(conn, cmd, procFinished, serviceState)

	glog.V(2).Info("Process started in goroutine")
	return true, nil
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	glog.V(1).Info("Starting HostAgent")
	for {
		// create a wrapping function so that client.Close() can be handled via defer
		keepGoing := func() bool {
			conn, zkEvt, err := zk.Connect(a.zookeepers, time.Second*10)
			if err != nil {
				glog.V(0).Info("Unable to connect, retrying.")
				return true
			}

			connectEvent := false
			for !connectEvent {
				select {
				case errc := <-a.closing:
					glog.V(0).Info("Received shutdown notice")
					errc <- errors.New("Unable to connect to zookeeper")
					return false

				case evt := <-zkEvt:
					glog.V(1).Infof("Got ZK connect event: %v", evt)
					if evt.State == zk.StateConnected {
						connectEvent = true
					}
				}
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

type stateResult struct {
	id  string
	err error
}

func (a *HostAgent) startMissingChildren(conn *zk.Conn, children []string, processing map[string]chan int, ssDone chan stateResult) {
	glog.V(1).Infof("Agent for %s processing %d children", a.hostId, len(children))
	for _, childName := range children {
		if processing[childName] == nil {
			glog.V(2).Info("Agent starting goroutine to watch ", childName)
			childChannel := make(chan int, 1)
			processing[childName] = childChannel
			go a.processServiceState(conn, childChannel, ssDone, childName)
		}
	}
}

func waitForSsNodes(processing map[string]chan int, ssResultChan chan stateResult) (err error) {
	for key, shutdown := range processing {
		glog.V(1).Infof("Agent signaling for %s to shutdown.", key)
		shutdown <- 1
	}

	// Wait for goroutines to shutdown
	for len(processing) > 0 {
		select {
		case ssResult := <-ssResultChan:
			glog.V(1).Infof("Goroutine finished %s", ssResult.id)
			if err == nil && ssResult.err != nil {
				err = ssResult.err
			}
			delete(processing, ssResult.id)
		}
	}
	glog.V(0).Info("All service state nodes are shut down")
	return
}

func (a *HostAgent) processChildrenAndWait(conn *zk.Conn) bool {
	processing := make(map[string]chan int)
	ssDone := make(chan stateResult, 25)

	hostPath := zzk.HostPath(a.hostId)

	for {

		children, _, zkEvent, err := conn.ChildrenW(hostPath)
		if err != nil {
			glog.V(0).Infoln("Unable to read children, retrying.")
			time.Sleep(3 * time.Second)
			return true
		}
		a.startMissingChildren(conn, children, processing, ssDone)

		select {

		case errc := <-a.closing:
			glog.V(1).Info("Agent received interrupt")
			err = waitForSsNodes(processing, ssDone)
			errc <- err
			return false

		case ssResult := <-ssDone:
			glog.V(1).Infof("Goroutine finished %s", ssResult.id)
			delete(processing, ssResult.id)

		case evt := <-zkEvent:
			glog.V(1).Info("Agent event: ", evt)
		}
	}
}

func (a *HostAgent) processServiceState(conn *zk.Conn, shutdown <-chan int, done chan<- stateResult, ssId string) {
	procFinished := make(chan int, 1)
	var attached bool

	for {

		var hss zzk.HostServiceState
		hssStats, zkEvent, err := zzk.LoadHostServiceStateW(conn, a.hostId, ssId, &hss)
		if err != nil {
			errS := fmt.Sprintf("Unable to load host service state %s: %v", ssId, err)
			glog.Error(errS)
			done <- stateResult{ssId, errors.New(errS)}
			return
		}
		if len(hss.ServiceStateId) == 0 || len(hss.ServiceId) == 0 {
			errS := fmt.Sprintf("Service for %s is invalid", zzk.HostServiceStatePath(a.hostId, ssId))
			glog.Error(errS)
			done <- stateResult{ssId, errors.New(errS)}
			return
		}

		var ss dao.ServiceState
		ssStats, err := zzk.LoadServiceState(conn, hss.ServiceId, hss.ServiceStateId, &ss)
		if err != nil {
			errS := fmt.Sprintf("Host service state unable to load service state %s", ssId)
			glog.Error(errS)
			// This goroutine is watching a node for a service state that does not
			// exist or could not be loaded. We should *probably* delete this node.
			hssPath := zzk.HostServiceStatePath(a.hostId, ssId)
			err = conn.Delete(hssPath, hssStats.Version)
			if err != nil {
				glog.Warningf("Unable to delete host service state %s", hssPath)
			}
			done <- stateResult{ssId, errors.New(errS)}
			return
		}

		var service dao.Service
		_, err = zzk.LoadService(conn, ss.ServiceId, &service)
		if err != nil {
			errS := fmt.Sprintf("Host service state unable to load service %s", ss.ServiceId)
			glog.Errorf(errS)
			done <- stateResult{ssId, errors.New(errS)}
			return
		}

		glog.V(1).Infof("Processing %s, desired state: %d", service.Name, hss.DesiredState)

		switch {

		case hss.DesiredState == dao.SVC_STOP:
			// This node is marked for death
			glog.V(1).Infof("Service %s was marked for death, quitting", service.Name)
			if attached {
				err = a.terminateAttached(conn, procFinished, &ss)
			} else {
				err = a.terminateInstance(conn, &ss)
			}
			done <- stateResult{ssId, err}
			return

		case attached:
			// Something uninteresting happened. Why are we here?
			glog.V(1).Infof("Service %s is attached in a child goroutine", service.Name)

		case hss.DesiredState == dao.SVC_RUN &&
			ss.Started.Year() <= 1 || ss.Terminated.Year() > 2:
			// Should run, and either not started or process died
			glog.V(1).Infof("Service %s does not appear to be running; starting", service.Name)
			attached, err = a.startService(conn, procFinished, ssStats, &service, &ss)

		case ss.Started.Year() > 1 && ss.Terminated.Year() <= 1:
			// Service superficially seems to be running. We need to attach
			glog.V(1).Infof("Service %s appears to be running; attaching", service.Name)
			attached, err = a.attachToService(conn, procFinished, &ss, &hss)

		default:
			glog.V(0).Infof("Unhandled service %s", service.Name)
		}

		if !attached || err != nil {
			errS := fmt.Sprintf("Service state %s unable to start or attach to process", ssId)
			glog.V(1).Info(errS)
			a.terminateInstance(conn, &ss)
			done <- stateResult{ssId, errors.New(errS)}
			return
		}

		glog.V(3).Infoln("Successfully processed state for %s", service.Name)

		select {

		case <-shutdown:
			glog.V(0).Info("Agent goroutine will stop watching ", ssId)
			err = a.terminateAttached(conn, procFinished, &ss)
			if err != nil {
				glog.Errorf("Error terminating %s: %v", service.Name, err)
			}
			done <- stateResult{ssId, err}
			return

		case <-procFinished:
			glog.V(1).Infof("Process finished %s", ssId)
			attached = false
			continue

		case evt := <-zkEvent:
			if evt.Type == zk.EventNodeDeleted {
				glog.V(0).Info("Host service state deleted: ", ssId)
				err = a.terminateAttached(conn, procFinished, &ss)
				if err != nil {
					glog.Errorf("Error terminating %s: %v", service.Name, err)
				}
				done <- stateResult{ssId, err}
				return
			}

			glog.V(1).Infof("Host service state %s received event %v", ssId, evt)
			continue
		}
	}
}

func (a *HostAgent) GetServiceEndpoints(serviceId string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return
	}
	defer controlClient.Close()

	err = controlClient.GetServiceEndpoints(serviceId, response)
	if err != nil {
		return err
	}
	endpoints := *response
	// add our internal services
	for key, endpoint := range a.getInternalServiceEndpoints() {
		if _, ok := (endpoints)[key]; ok {
			endpoints[key] = append(endpoints[key], &endpoint)
		}
		endpoints[key] = make([]*dao.ApplicationEndpoint, 0)
		endpoints[key] = append(endpoints[key], &endpoint)
	}
	return nil
}

// getInternalServiceEndpoints lists every internal service that we wish to expose to the containers running on this agent
func (a *HostAgent) getInternalServiceEndpoints() map[string]dao.ApplicationEndpoint {
	// master is of the form host:port, we just need the host
	master := strings.Split(a.master, ":")[0]
	return map[string]dao.ApplicationEndpoint{
		"tcp:8787": dao.ApplicationEndpoint{
			"controlplane",
			8787,
			8787,
			master,
			"127.0.0.1",
			"tcp",
		},
	}
}

// Create a Host object from the host this function is running on.
func (a *HostAgent) GetInfo(unused int, host *dao.Host) error {
	hostInfo, err := CurrentContextAsHost("UNKNOWN")
	if err != nil {
		return err
	}
	*host = *hostInfo
	return nil
}
