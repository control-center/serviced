// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.
package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	coordzk "github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/zzk"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
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

const (
	circularBufferSize = 1000
)

// An instance of the control plane Agent.
type HostAgent struct {
	master          string               // the connection string to the master agent
	uiport          string               // the port to the ui (legacy was port 8787, now default 443)
	hostId          string               // the hostID of the current host
	dockerDns       []string             // docker dns addresses
	varPath         string               // directory to store serviced	 data
	mount           []string             // each element is in the form: dockerImage,hostPath,containerPath
	vfs             string               // driver for container volumes
	currentServices map[string]*exec.Cmd // the current running services
	mux             TCPMux
	closing         chan chan error
	proxyRegistry   proxy.ProxyRegistry
	zkClient        *coordclient.Client
}

// assert that this implemenents the Agent interface
var _ Agent = &HostAgent{}

func getZkDSN(zookeepers []string) string {
	if len(zookeepers) == 0 {
		zookeepers = []string{"127.0.0.1:2181"}
	}
	dsn := coordzk.DSN{
		Servers: zookeepers,
		Timeout: time.Second * 15,
	}
	return dsn.String()
}

// Create a new HostAgent given the connection string to the
func NewHostAgent(master string, uiport string, dockerDns []string, varPath string, mount []string, vfs string, zookeepers []string, mux TCPMux) (*HostAgent, error) {
	// save off the arguments
	agent := &HostAgent{}
	agent.master = master
	agent.uiport = uiport
	agent.dockerDns = dockerDns
	agent.varPath = varPath
	agent.mount = mount
	agent.vfs = vfs
	agent.mux = mux
	if agent.mux.Enabled {
		go agent.mux.ListenAndMux()
	}

	dsn := getZkDSN(zookeepers)
	basePath := ""
	zkClient, err := coordclient.New("zookeeper", dsn, basePath, nil)
	if err != nil {
		return nil, err
	}
	agent.zkClient = zkClient

	agent.closing = make(chan chan error)
	hostId, err := HostID()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostId = hostId
	agent.currentServices = make(map[string]*exec.Cmd)

	agent.proxyRegistry = proxy.NewDefaultProxyRegistry()
	go agent.start()
	return agent, err
}

// Use the Context field of the given template to fill in all the templates in
// the Command fields of the template's ServiceDefinitions
func injectContext(s *dao.Service, cp dao.ControlPlane) error {
	err := s.EvaluateLogConfigTemplate(cp)
	if err != nil {
		return err
	}
	return s.EvaluateStartupTemplate(cp)
}

func (a *HostAgent) Shutdown() error {
	glog.V(2).Info("Issuing shutdown signal")
	errc := make(chan error)
	a.closing <- errc
	return <-errc
}

// Attempts to attach to a running container
func (a *HostAgent) attachToService(conn coordclient.Connection, procFinished chan<- int, serviceState *dao.ServiceState, hss *zzk.HostServiceState) (bool, error) {

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

	go a.waitForProcessToDie(conn, serviceState.DockerId, procFinished, serviceState)
	return true, nil
}

func markTerminated(conn coordclient.Connection, hss *zzk.HostServiceState) {
	ssPath := zzk.ServiceStatePath(hss.ServiceId, hss.ServiceStateId)
	exists, err := conn.Exists(ssPath)
	if err != nil {
		glog.V(0).Infof("Unable to get service state %s for delete because: %v", ssPath, err)
		return
	}
	if exists {
		err = conn.Delete(ssPath)
		if err != nil {
			glog.V(0).Infof("Unable to delete service state %s because: %v", ssPath, err)
			return
		}
	}

	hssPath := zzk.HostServiceStatePath(hss.HostId, hss.ServiceStateId)
	exists, err = conn.Exists(hssPath)
	if err != nil {
		glog.V(0).Infof("Unable to get host service state %s for delete becaus: %v", hssPath, err)
		return
	}
	if exists {
		err = conn.Delete(hssPath)
		if err != nil {
			glog.V(0).Infof("Unable to delete host service state %s", hssPath)
		}

	}
	return
}

// Terminate a particular service instance (serviceState) on the localhost.
func (a *HostAgent) terminateInstance(conn coordclient.Connection, serviceState *dao.ServiceState) error {
	err := a.dockerTerminate(serviceState.Id)
	if err != nil {
		return err
	}
	markTerminated(conn, zzk.SsToHss(serviceState))
	return nil
}

func (a *HostAgent) terminateAttached(conn coordclient.Connection, procFinished <-chan int, ss *dao.ServiceState) error {
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
	killout, killerr := cmd.CombinedOutput()
	if killerr != nil {
		//verify dockerId no longer exists
		cmd = exec.Command("docker", "inspect", dockerId)
		existsout, err := cmd.CombinedOutput()
		strout := string(existsout)
		if err != nil && strings.HasPrefix(strout, "Error: No such image or container:") {
			glog.V(4).Infof("Container does not exist; instance %s, %v", dockerId, strout)
			return nil
		}
		glog.V(1).Infof("problem killing container instance %s, %v;%v", dockerId, string(killout), killerr)
		return errors.New(string(killout))
	}

	glog.V(2).Infof("Successfully killed %s", dockerId)
	return nil
}

// Get the state of the docker container given the dockerId
func getDockerState(dockerId string) (containerState ContainerState, err error) {
	// get docker status

	cmd := exec.Command("docker", "inspect", dockerId)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.V(2).Infof("problem getting docker state: %s, err %s", dockerId, string(output))
		return containerState, err
	}
	var containerStates []ContainerState
	err = json.Unmarshal(output, &containerStates)
	if err != nil {
		glog.Errorf("bad state	happened: %v,	\n\n\n%s", err, string(output))
		return containerState, dao.ControlPlaneError{Msg: "no state"}
	}
	if len(containerStates) < 1 {
		return containerState, dao.ControlPlaneError{Msg: "no container"}
	}
	return containerStates[0], err
}

func dumpOut(stdout, stderr io.Reader, size int) {
	dumpBuffer(stdout, size, "stdout")
	dumpBuffer(stderr, size, "stderr")
}

func dumpBuffer(reader io.Reader, size int, name string) {
	buffer := make([]byte, size)
	if n, err := reader.Read(buffer); err != nil {
		glog.V(1).Infof("Unable to read %s of dump", name)
	} else {
		message := strings.TrimSpace(string(buffer[:n]))
		if len(message) > 0 {
			glog.V(0).Infof("Process %s:\n%s", name, message)
		}
	}
}

func (a *HostAgent) waitForProcessToDie(conn coordclient.Connection, containerId string, procFinished chan<- int, serviceState *dao.ServiceState) {

	defer func() {
		procFinished <- 1
	}()

	exited := make(chan error)

	go func() {
		var err error
		cmd := exec.Command("docker", "wait", containerId)
		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			glog.Errorf("docker wait exited with: %s : %s", err, string(output))
			// TODO: output of docker logs is potentially very large
			// this should be implemented another way, perhaps a docker attach
			// or extend docker to give the last N seconds
			cmd = exec.Command("docker", "logs", containerId)
			output, err := cmd.CombinedOutput()
			if err != nil {
				glog.Errorf("Could not get logs for container %s", containerId)
			} else {
				// get last 1000 bytes
				str := string(output)
				last := len(str) - 1000
				if last < 0 {
					last = 0
				}
				str = str[last:]
				glog.Warning("Last 1000 bytes of container %s: %s", containerId, str)

			}

		}
		glog.Infof("docker wait %s exited", containerId)
		// get rid of the container
		if rmErr := exec.Command("docker", "rm", containerId).Run(); rmErr != nil {
			glog.Errorf("Could not remove container: %s: %s", containerId, rmErr)
		}
		exited <- err
	}()

	// We are name the container the same as its service state ID, so use that as an alias
	dockerId := serviceState.Id
	serviceState.DockerId = dockerId

	time.Sleep(1 * time.Second) // Sleep to give docker a chance to start

	var containerState ContainerState
	var err error
	for i := 0; i < 30; i++ {
		if containerState, err = getDockerState(containerId); err != nil {
			time.Sleep(3 * time.Second) // Sleep to give docker a chance to start
			glog.V(2).Infof("Problem getting service state for %s :%v", serviceState.Id, err)
		} else {
			break
		}
	}

	if err != nil {
		return
		//TODO: should	"cmd" be cleaned up before returning?
	}

	var sState *dao.ServiceState
	if err = zzk.LoadAndUpdateServiceState(conn, serviceState.ServiceId, serviceState.Id, func(ss *dao.ServiceState) {
		ss.DockerId = containerState.ID
		ss.Started = time.Now()
		ss.Terminated = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
		ss.PrivateIp = containerState.NetworkSettings.IPAddress
		ss.PortMapping = containerState.NetworkSettings.Ports
		sState = ss
	}); err != nil {
		glog.Warningf("Unable to update service state %s: %v", serviceState.Id, err)
		//TODO: should	"cmd" be cleaned up before returning?
	} else {

		//start IP resource proxy for each endpoint
		var service dao.Service
		if err = zzk.LoadService(conn, serviceState.ServiceId, &service); err != nil {
			glog.Warningf("Unable to read service %s: %v", serviceState.Id, err)
		} else {
			glog.V(4).Infof("Looking for address assignment in service %s:%s", service.Name, service.Id)
			for _, endpoint := range service.Endpoints {
				if addressConfig := endpoint.GetAssignment(); addressConfig != nil {
					glog.V(4).Infof("Found address assignment for %s:%s endpoint %s", service.Name, service.Id, endpoint.Name)
					proxyId := fmt.Sprintf("%v:%v", sState.ServiceId, endpoint.Name)

					frontEnd := proxy.ProxyAddress{IP: addressConfig.IPAddr, Port: addressConfig.Port}
					backEnd := proxy.ProxyAddress{IP: sState.PrivateIp, Port: endpoint.PortNumber}

					err = a.proxyRegistry.CreateProxy(proxyId, endpoint.Protocol, frontEnd, backEnd)
					if err != nil {
						glog.Warningf("Could not start External address proxy for %v; error: proxyId", proxyId, err)
					}
					defer a.proxyRegistry.RemoveProxy(proxyId)

				}
			}

		}

		glog.V(1).Infof("SSPath: %s, PortMapping: %v", zzk.ServiceStatePath(serviceState.ServiceId, serviceState.Id), serviceState.PortMapping)

		loop := true
		stateUpdateEvery := time.Tick(time.Second * 20)
		for loop {
			select {
			case err := <-exited:
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
							}
						}
					} else {
						glog.V(1).Info("Unable to determine exit code for %s", serviceState.Id)
					}
				} else {
					glog.V(0).Infof("Process for service state %s finished", serviceState.Id)
				}
				loop = false
			case <-stateUpdateEvery:
				containerState, err := getDockerState(containerId)
				if err != nil {
					glog.Errorf("Could not get docker state: %v", err)
					continue
				}
				if err = zzk.LoadAndUpdateServiceState(conn, serviceState.ServiceId, serviceState.Id, func(ss *dao.ServiceState) {
					ss.DockerId = containerState.ID
					ss.Started = time.Now()
					ss.Terminated = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
					ss.PrivateIp = containerState.NetworkSettings.IPAddress
					ss.PortMapping = containerState.NetworkSettings.Ports
					sState = ss
				}); err != nil {
					glog.Warningf("Unable to update service state %s: %v", serviceState.Id, err)
				}

			}
		}
		if err = zzk.ResetServiceState(conn, serviceState.ServiceId, serviceState.Id); err != nil {
			glog.Errorf("Caught error marking process termination time for %s: %v", serviceState.Id, err)
		}

	}
}

func getSubvolume(varPath, poolId, tenantId, fs string) (*volume.Volume, error) {
	baseDir, _ := filepath.Abs(path.Join(varPath, "volumes"))
	if _, err := volume.Mount(fs, poolId, baseDir); err != nil {
		return nil, err
	}
	baseDir, _ = filepath.Abs(path.Join(varPath, "volumes", poolId))
	return volume.Mount(fs, tenantId, baseDir)
}

/*
writeConfFile is responsible for writing contents out to a file
Input string prefix	 : cp_cd67c62b-e462-5137-2cd8-38732db4abd9_zenmodeler_logstash_forwarder_conf_
Input string id		 : Service ID (example cd67c62b-e462-5137-2cd8-38732db4abd9)
Input string filename: zenmodeler_logstash_forwarder_conf
Input string content : the content that you wish to write to a file
Output *os.File	 f	 : file handler to the file that you've just opened and written the content to
Example name of file that is written: /tmp/cp_cd67c62b-e462-5137-2cd8-38732db4abd9_zenmodeler_logstash_forwarder_conf_592084261
*/
func writeConfFile(prefix string, id string, filename string, content string) (*os.File, error) {
	f, err := ioutil.TempFile("", prefix)
	if err != nil {
		glog.Errorf("Could not generate tempfile for config %s %s", id, filename)
		return f, err
	}
	_, err = f.WriteString(content)
	if err != nil {
		glog.Errorf("Could not write out config file %s %s", id, filename)
		return f, err
	}

	return f, nil
}

// chownConfFile() runs 'chown $owner $filename && chmod $permissions $filename'
// using the given dockerImage. An error is returned if owner is not specified,
// the owner is not in user:group format, or if there was a problem setting
// the permissions.
func chownConfFile(filename, owner, permissions string, dockerImage string) error {
	// TODO: reach in to the dockerImage and get the effective UID, GID so we can do this without a bind mount
	if !validOwnerSpec(owner) {
		return fmt.Errorf("Unsupported owner specification: %s", owner)
	}

	uid, gid, err := getInternalImageIds(owner, dockerImage)
	if err != nil {
		return err
	}
	// this will fail if we are not running as root
	if err := os.Chown(filename, uid, gid); err != nil {
		return err
	}
	octal, err := strconv.ParseInt(permissions, 8, 32)
	if err != nil {
		return err
	}
	if err := os.Chmod(filename, os.FileMode(octal)); err != nil {
		return err
	}
	return nil
}

// Start a service instance and update the CP with the state.
func (a *HostAgent) startService(conn coordclient.Connection, procFinished chan<- int, service *dao.Service, serviceState *dao.ServiceState) (bool, error) {
	glog.V(2).Infof("About to start service %s with name %s", service.Id, service.Name)
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return false, err
	}
	defer client.Close()

	//get this service's tenantId for volume mapping
	var tenantId string
	err = client.GetTenantId(service.Id, &tenantId)
	if err != nil {
		glog.Errorf("Failed getting tenantId for service: %s, %s", service.Id, err)
	}

	// get the system user
	unused := 0
	systemUser := dao.User{}
	err = client.GetSystemUser(unused, &systemUser)
	if err != nil {
		glog.Errorf("Unable to get system user account for agent %s", err)
	}
	glog.V(0).Infof("System User %v", systemUser)

	// get the points
	portOps := ""
	if service.Endpoints != nil {
		glog.V(1).Info("Endpoints for service: ", service.Endpoints)
		for _, endpoint := range service.Endpoints {
			if endpoint.Purpose == "export" { // only expose remote endpoints
				portOps += fmt.Sprintf(" -p %d", endpoint.PortNumber)
				if endpoint.Protocol == commons.UDP {
					portOps += "/udp"
				}
			}
		}
	}

	volumeOpts := ""
	if len(tenantId) == 0 && len(service.Volumes) > 0 {
		// FIXME: find a better way of handling this error condition
		glog.Fatalf("Could not get tenant ID and need to mount a volume, service state: %s, service id: %s", serviceState.Id, service.Id)
	}
	for _, volume := range service.Volumes {

		sv, err := getSubvolume(a.varPath, service.PoolId, tenantId, a.vfs)
		if err != nil {
			glog.Fatalf("Could not create subvolume: %s", err)
		} else {

			glog.Infof("sv: %v", sv)
			glog.Infof("Path: %s", sv.Path())
			glog.Infof("RP: %s", volume.ResourcePath)

			resourcePath := path.Join(sv.Path(), volume.ResourcePath)
			if err = os.MkdirAll(resourcePath, 0770); err != nil {
				glog.Fatalf("Could not create resource path: %s, %s", resourcePath, err)
			}

			if err := createVolumeDir(resourcePath, volume.ContainerPath, service.ImageId, volume.Owner, volume.Permission); err != nil {
				glog.Errorf("Error populating resource path: %s with container path: %s, %v", resourcePath, volume.ContainerPath, err)
			}
			volumeOpts += fmt.Sprintf(" -v %s:%s", resourcePath, volume.ContainerPath)
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
		f, err := writeConfFile(prefix, service.Id, filename, config.Content)
		if err != nil {
			return false, err
		}

		if err := chownConfFile(f.Name(), config.Owner, config.Permissions, service.ImageId); err != nil {
			glog.Errorf("Could not chown config file for %s, %s: %s", service.Id, filename, err)
		}

		// everything worked!
		configFiles += fmt.Sprintf(" -v %s:%s ", f.Name(), filename)
	}

	// if this container is going to produce any logs, create the config and get the bind mounts
	logstashForwarderMount := ""
	if len(service.LogConfigs) != 0 {

		// write out the log file config
		configFileName, err := writeLogstashAgentConfig(service)
		if err != nil {
			return false, err
		}

		// bind mount the conf file and everything we need for logstash-forwarder
		logstashForwarderMount = getLogstashBindMounts(configFileName)
	}

	// add arguments to mount requested directory (if requested)
	requestedMount := ""
	for _, bindMountString := range a.mount {
		splitMount := strings.Split(bindMountString, ",")
		numMountArgs := len(splitMount)

		if numMountArgs == 2 || numMountArgs == 3 {
			requestedImage := splitMount[0]
			hostPath := splitMount[1]

			// assume the container path is going to be the same as the host path
			containerPath := hostPath

			// if the container path is provided, use it
			if numMountArgs > 2 {
				containerPath = splitMount[2]
			}

			// insert tenantId into requestedImage - see dao.DeployService
			matchedRequestedImage := false
			if requestedImage == "*" {
				matchedRequestedImage = true
			} else {
				path := strings.SplitN(requestedImage, "/", 3)
				path[len(path)-1] = tenantId + "_" + path[len(path)-1]
				requestedImage = strings.Join(path, "/")
				matchedRequestedImage = (requestedImage == service.ImageId)
			}

			if matchedRequestedImage {
				requestedMount += " -v " + hostPath + ":" + containerPath
			}
		} else {
			glog.Warningf("Could not bind mount the following: %s", bindMountString)
		}
	}

	// add arguments for environment variables
	environmentVariables := "-e CONTROLPLANE=1 "
	environmentVariables += "-e CONTROLPLANE_CONSUMER_URL=http://localhost:22350/api/metrics/store "
	environmentVariables += fmt.Sprintf("-e CONTROLPLANE_SYSTEM_USER=%s ", systemUser.Name)
	environmentVariables += fmt.Sprintf("-e CONTROLPLANE_SYSTEM_PASSWORD=%s ", systemUser.Password)

	// add dns values to setup
	dns := ""
	for _, addr := range a.dockerDns {
		if len(dns) > 0 {
			dns += " "
		}

		_addr := strings.TrimSpace(addr)
		if len(_addr) > 0 {
			dns += "--dns " + addr
		}
	}

	// Add hostname if set
	hostname := ""
	if service.Hostname != "" {
		hostname = fmt.Sprintf("-h %s", service.Hostname)
	}

	proxyCmd := fmt.Sprintf("/serviced/%s proxy %s '%s'", binary, service.Id, service.Startup)
	cmdString := fmt.Sprintf("docker run %s %s %s -d --name=%s %s -v %s %s %s %s %s %s %s", dns, portOps, hostname, serviceState.Id, environmentVariables, volumeBinding, requestedMount, logstashForwarderMount, volumeOpts, configFiles, service.ImageId, proxyCmd)
	glog.V(0).Infof("Starting: %s", cmdString)

	a.dockerTerminate(serviceState.Id)
	a.dockerRemove(serviceState.Id)

	cmd := exec.Command("bash", "-c", cmdString)
	bytes, err := cmd.Output()
	if err != nil {
		glog.Errorf("Could not start container for service %s", serviceState.Id)
		return false, err
	}

	dockerId := strings.TrimSpace(string(bytes))
	go a.waitForProcessToDie(conn, dockerId, procFinished, serviceState)

	glog.V(2).Info("Process started in goroutine")
	return true, nil
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	glog.V(1).Info("Starting HostAgent")
	for {
		// create a wrapping function so that client.Close() can be handled via defer
		keepGoing := func() bool {

			connc := make(chan coordclient.Connection)
			var conn coordclient.Connection
			done := make(chan struct{}, 1)
			defer func() {
				done <- struct{}{}
			}()
			go func() {
				for {
					c, err := a.zkClient.GetConnection()
					if err == nil {
						connc <- c
						return
					}
					// exit when our parent exits
					select {
					case <-done:
						return
					case <-time.After(time.Second):
					}
				}
				close(connc)
			}()
			select {
			case errc := <-a.closing:
				glog.V(0).Info("Received shutdown notice")
				a.zkClient.Close()
				errc <- errors.New("Unable to connect to zookeeper")
				return false

			case conn = <-connc:
				glog.V(1).Info("Got a connected client")
			}
			defer conn.Close()
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

// startMissingChildren accepts a zookeeper connection (conn) and a slice of service instance ids (children),
// a map of channels to signal running children stop, and a stateResult channel for children to signal when
// they shutdown
func (a *HostAgent) startMissingChildren(conn coordclient.Connection, children []string, processing map[string]chan int, ssDone chan stateResult) {
	glog.V(1).Infof("Agent for %s processing %d children", a.hostId, len(children))
	for _, childName := range children {
		if processing[childName] == nil {
			glog.V(2).Info("Agent starting goroutine to watch ", childName)
			childChannel := make(chan int, 1)
			processing[childName] = childChannel
			go a.processServiceState(conn, childChannel, ssDone, childName)
		}
	}
	return
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

func (a *HostAgent) processChildrenAndWait(conn coordclient.Connection) bool {
	processing := make(map[string]chan int)
	ssDone := make(chan stateResult, 25)

	hostPath := zzk.HostPath(a.hostId)

	for {

		conn.CreateDir(hostPath)

		glog.V(3).Infof("getting children of %s", hostPath)
		children, zkEvent, err := conn.ChildrenW(hostPath)
		if err != nil {
			glog.V(0).Infof("Unable to read children, retrying: %s", err)
			select {
			case <-time.After(3 * time.Second):
				return true
			case errc := <-a.closing:
				glog.V(1).Info("Agent received interrupt")
				err = waitForSsNodes(processing, ssDone)
				errc <- err
				return false
			}
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

func (a *HostAgent) processServiceState(conn coordclient.Connection, shutdown <-chan int, done chan<- stateResult, ssId string) {
	procFinished := make(chan int, 1)
	var attached bool

	for {
		var hss zzk.HostServiceState
		zkEvent, err := zzk.LoadHostServiceStateW(conn, a.hostId, ssId, &hss)
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
		if err := zzk.LoadServiceState(conn, hss.ServiceId, hss.ServiceStateId, &ss); err != nil {
			errS := fmt.Sprintf("Host service state unable to load service state %s", ssId)
			glog.Error(errS)
			// This goroutine is watching a node for a service state that does not
			// exist or could not be loaded. We should *probably* delete this node.
			hssPath := zzk.HostServiceStatePath(a.hostId, ssId)
			if err := conn.Delete(hssPath); err != nil {
				glog.Warningf("Unable to delete host service state %s", hssPath)
			}
			done <- stateResult{ssId, errors.New(errS)}
			return
		}

		var service dao.Service
		if err := zzk.LoadService(conn, ss.ServiceId, &service); err != nil {
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
			attached, err = a.startService(conn, procFinished, &service, &ss)

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
			if evt.Type == coordclient.EventNodeDeleted {
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

// GetInfo creates a Host object from the host this function is running on.
func (a *HostAgent) GetInfo(ips []string, host *dao.Host) error {
	hostInfo, err := CurrentContextAsHost("UNKNOWN")
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		// use the default IP of the host if specific IPs have not been requested
		ips = append(ips, hostInfo.IpAddr)
	}
	hostIPs, err := getIPResources(hostInfo.Id, ips...)
	if err != nil {
		return err
	}
	hostInfo.IPs = hostIPs
	*host = *hostInfo
	return nil
}

// getIPResources does the actual work of determining the IPs on the host. Parameters are the IPs to filter on
func getIPResources(hostId string, ipaddress ...string) ([]dao.HostIPResource, error) {

	interfaces, err := net.Interfaces()
	if err != nil {
		glog.Error("Problem reading interfaces: ", err)
		return []dao.HostIPResource{}, err
	}
	//make a  of all ipaddresses to interface
	ips := make(map[string]net.Interface)
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			glog.Error("Problem reading interfaces: ", err)
			return []dao.HostIPResource{}, err
		}
		for _, ip := range addrs {
			normalIP := strings.SplitN(ip.String(), "/", 2)[0]
			normalIP = strings.Trim(strings.ToLower(normalIP), " ")

			ips[normalIP] = iface
		}
	}

	glog.V(4).Infof("Interfaces on this host %v", ips)

	hostIPResources := make([]dao.HostIPResource, 0, len(interfaces))

	validate := func(iface net.Interface, ip string) error {
		if (uint(iface.Flags) & (1 << uint(net.FlagLoopback))) == 0 {
			return fmt.Errorf("Loopback address %v cannot be used to register a host", ip)
		}
		return nil
	}

	for _, ipaddr := range ipaddress {
		normalIP := strings.Trim(strings.ToLower(ipaddr), " ")
		iface, found := ips[normalIP]
		if !found {
			return []dao.HostIPResource{}, fmt.Errorf("IP address %v not valid for this host", ipaddr)
		}
		err = validate(iface, normalIP)
		if err != nil {
			return []dao.HostIPResource{}, err
		}
		hostIp := dao.HostIPResource{}
		hostIp.HostId = hostId
		hostIp.IPAddress = ipaddr
		hostIp.InterfaceName = iface.Name
		hostIPResources = append(hostIPResources, hostIp)
	}
	return hostIPResources, nil
}
