// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.
package serviced

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/circular"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/zzk"

	"encoding/json"
	"errors"
	"fmt"
	"github.com/zenoss/glog"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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
	master          string   // the connection string to the master agent
	hostId          string   // the hostID of the current host
	varPath         string   // directory to store serviced  data
	mount           []string // each element is in the form: container_image:host_path:container_path
	vfs             string   // driver for container volumes
	zookeepers      []string
	currentServices map[string]*exec.Cmd // the current running services
	mux             TCPMux
	closing         chan chan error
}

// assert that this implemenents the Agent interface
var _ Agent = &HostAgent{}

// Create a new HostAgent given the connection string to the

func NewHostAgent(master string, varPath string, mount []string, vfs string, zookeepers []string, mux TCPMux) (*HostAgent, error) {
	// save off the arguments
	agent := &HostAgent{}
	agent.master = master
	agent.varPath = varPath
	agent.mount = mount
	agent.vfs = vfs
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
		glog.V(2).Infof("problem getting docker state: %s", dockerId)
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

func dumpOut(stdout, stderr io.Reader, size int) {
	dumpBuffer(stdout, size, "stdout")
	dumpBuffer(stderr, size, "stderr")
}

func dumpBuffer(reader io.Reader, size int, name string) {

	buffer := make([]byte, size)
	if n, err := reader.Read(buffer); err != nil {
		glog.V(1).Infof("Unable to read %s of dump", name)
	} else {
		glog.V(0).Infof("Process %s:\n%s", name, string(buffer[:n]))
	}

}

func (a *HostAgent) waitForProcessToDie(conn *zk.Conn, cmd *exec.Cmd, procFinished chan<- int, serviceState *dao.ServiceState) {
	a.dockerRemove(serviceState.Id)

	defer func() {
		procFinished <- 1
	}()

	// save the last circularBufferSize bytes from each container run
	lastStdout := circular.NewBuffer(circularBufferSize)
	lastStderr := circular.NewBuffer(circularBufferSize)

	if stdout, err := cmd.StdoutPipe(); err != nil {
		glog.Errorf("Unable to read standard out for service state %s: %v", serviceState.Id, err)
		return
	} else {
		go io.Copy(lastStdout, stdout)
	}
	if stderr, err := cmd.StderrPipe(); err != nil {
		glog.Errorf("Unable to read standard error for service state %s: %v", serviceState.Id, err)
		return
	} else {
		go io.Copy(lastStderr, stderr)
	}

	if err := cmd.Start(); err != nil {
		glog.Errorf("Problem starting command '%s %s': %v", cmd.Path, cmd.Args, err)
		dumpOut(lastStdout, lastStderr, circularBufferSize)
		return
	}

	// We are name the container the same as its service state ID, so use that as an alias
	dockerId := serviceState.Id
	serviceState.DockerId = dockerId

	time.Sleep(1 * time.Second) // Sleep to give docker a chance to start

	var containerState ContainerState
	var err error
	for i := 0; i < 30; i++ {
		if containerState, err = getDockerState(dockerId); err != nil {
			time.Sleep(3 * time.Second) // Sleep to give docker a chance to start
			glog.V(2).Infof("Problem getting service state for %s :%v", serviceState.Id, err)
			a.dockerTerminate(dockerId)
			dumpOut(lastStdout, lastStderr, circularBufferSize)
		} else {
			break
		}
	}

	if err != nil {
		return
	}
	if err = zzk.LoadAndUpdateServiceState(conn, serviceState.ServiceId, serviceState.Id, func(ss *dao.ServiceState) {
		ss.DockerId = containerState.ID
		ss.Started = time.Now()
		ss.Terminated = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
		ss.PrivateIp = containerState.NetworkSettings.IPAddress
		ss.PortMapping = containerState.NetworkSettings.Ports
	}); err != nil {
		glog.Warningf("Unable to update service state %s: %v", serviceState.Id, err)
	} else {

		glog.V(1).Infof("SSPath: %s, PortMapping: %v", zzk.ServiceStatePath(serviceState.ServiceId, serviceState.Id), serviceState.PortMapping)

		if err := cmd.Wait(); err != nil {
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
						dumpOut(lastStdout, lastStderr, circularBufferSize)
					}
				}
			} else {
				glog.V(1).Info("Unable to determine exit code for %s", serviceState.Id)
			}
		} else {
			glog.V(0).Infof("Process for service state %s finished", serviceState.Id)
		}

		if err = zzk.ResetServiceState(conn, serviceState.ServiceId, serviceState.Id); err != nil {
			glog.Errorf("Caught error marking process termination time for %s: %v", serviceState.Id, err)
		}

	}
}

func getSubvolume(varPath, poolId, tenantId, fs string) (*volume.Volume, error) {
	baseDir, _ := filepath.Abs(path.Join(varPath, "volumes"))
	volume.Mount(fs, poolId, baseDir)
	baseDir, _ = filepath.Abs(path.Join(varPath, "volumes", poolId))
	return volume.Mount(fs, tenantId, baseDir)
}

/*
writeConfFile is responsible for writing contents out to a file
Input string prefix  : cp_cd67c62b-e462-5137-2cd8-38732db4abd9_zenmodeler_logstash_forwarder_conf_
Input string id      : Service ID (example cd67c62b-e462-5137-2cd8-38732db4abd9)
Input string filename: zenmodeler_logstash_forwarder_conf
Input string content : the content that you wish to write to a file
Output *os.File  f   : file handler to the file that you've just opened and written the content to
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

/*
chownConfFile is responsible for changing the owner of a file
Input *os.File f     : file handler to a file that has already been opened
Input string id      : Service ID (example cd67c62b-e462-5137-2cd8-38732db4abd9)
Input string filename: zenmodeler_logstash_forwarder_conf
Input string owner   : update the file's owner to this provided string
Output bool          : returns true if: the owner parameter is not present present OR the file has been chowned to the requested owner successfully
*/
func chownConfFile(f *os.File, id string, filename string, owner string) bool {
	if len(owner) != 0 {
		parts := strings.Split(owner, ":")
		if len(parts) != 2 {
			glog.Errorf("Unsupported owner specification: %s, only %%d:%%d supported for now: %s, %s", owner, id, filename)
			return false
		}
		uid, err := strconv.Atoi(parts[0])
		if err != nil {
			glog.Warningf("Malformed UID: %s %s: %s", id, filename, err)
			return false
		}
		gid, err := strconv.Atoi(parts[0])
		if err != nil {
			glog.Warningf("Malformed GID: %s %s: %s", id, filename, err)
			return false
		}
		err = f.Chown(uid, gid)
		if err != nil {
			glog.Warningf("Could not chown config file: %s %s: %s", id, filename, err)
		}
	}
	return true
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

	//get this service's tenantId for env injection
	var tenantId string
	err = client.GetTenantId(service.Id, &tenantId)
	if err != nil {
		glog.Errorf("Failed getting tenantId for service: %s, %s", service.Id, err)
	}

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
	if len(tenantId) == 0 && len(service.Volumes) > 0 {
		// FIXME: find a better way of handling this error condition
		glog.Fatalf("Could not get tenant ID and need to mount a volume, service state: %s, service id: %s", serviceState.Id, service.Id)
	}
	for _, volume := range service.Volumes {

		sv, err := getSubvolume(a.varPath, service.PoolId, tenantId, a.vfs)
		if err != nil {
			glog.Fatal("Could not create subvolume: %s", err)
		} else {

			resourcePath := path.Join(sv.Path(), volume.ResourcePath)
			if err = os.MkdirAll(resourcePath, 0770); err != nil {
				glog.Fatal("Could not create resource path: %s, %s", resourcePath, err)
			}

			if err := createVolumeDir(resourcePath, volume.ContainerPath, service.ImageId, volume.Owner, volume.Permission); err != nil {
				glog.Fatalf("Error creating resource path: %v", err)
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

		fileChowned := chownConfFile(f, service.Id, filename, config.Owner)
		if fileChowned == false {
			continue
		}

		// everything worked!
		configFiles += fmt.Sprintf(" -v %s:%s ", f.Name(), filename)
	}

	// if this container is going to produce any logs, bind mount the following files:
	// logstash-forwarder, sslCertificate, sslKey, logstash-forwarder conf
	// FIX ME: consider moving this functionality to its own function...
	logstashForwarderMount := ""
	if len(service.LogConfigs) > 0 {
		logstashForwarderLogConf := `
        {
        	"paths": [ "%s" ],
        	"fields": { "type": "%s" }
        }`
		logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, service.LogConfigs[0].Path, service.LogConfigs[0].Type)
		for _, logConfig := range service.LogConfigs[1:] {
			logstashForwarderLogConf = logstashForwarderLogConf + `,
				{
					"paths": [ "%s" ],
					"fields": { "type": "%s" }
				}`
			logstashForwarderLogConf = fmt.Sprintf(logstashForwarderLogConf, logConfig.Path, logConfig.Type)
		}

		containerDefaultGatewayAndLogstashForwarderPort := "172.17.42.1:5043"
		// *********************************************************************************************
		// ***** FIX ME the following 3 variables are defined in serviced/proxy.go as well! ************
		containerLogstashForwarderDir := "/usr/local/serviced/resources/logstash"
		containerLogstashForwarderBinaryPath := containerLogstashForwarderDir + "/logstash-forwarder"
		containerLogstashForwarderConfPath := containerLogstashForwarderDir + "/logstash-forwarder.conf"
		// *********************************************************************************************
		containerSSLCertificatePath := containerLogstashForwarderDir + "/logstash-forwarder.crt"
		containerSSLKeyPath := containerLogstashForwarderDir + "/logstash-forwarder.key"

		logstashForwarderShipperConf := `
			{
				"network": {
			    	"servers": [ "%s" ],
					"ssl certificate": "%s",
					"ssl key": "%s",
					"ssl ca": "%s",
			    	"timeout": 15
			   	},
			   	"files": [
					%s
			   	]
			}`
		logstashForwarderShipperConf = fmt.Sprintf(logstashForwarderShipperConf, containerDefaultGatewayAndLogstashForwarderPort, containerSSLCertificatePath, containerSSLKeyPath, containerSSLCertificatePath, logstashForwarderLogConf)

		filename := service.Name + "_logstash_forwarder_conf"
		prefix := fmt.Sprintf("cp_%s_%s_", service.Id, strings.Replace(filename, "/", "__", -1))
		f, err := writeConfFile(prefix, service.Id, filename, logstashForwarderShipperConf)
		if err != nil {
			return false, err
		}

		logstashPath := resourcesDir() + "/logstash"
		hostLogstashForwarderPath := logstashPath + "/logstash-forwarder"
		hostLogstashForwarderConfPath := f.Name()
		hostSSLCertificatePath := logstashPath + "/logstash-forwarder.crt"
		hostSSLKeyPath := logstashPath + "/logstash-forwarder.key"

		logstashForwarderBinaryMount := " -v " + hostLogstashForwarderPath + ":" + containerLogstashForwarderBinaryPath
		logstashForwarderConfFileMount := " -v " + hostLogstashForwarderConfPath + ":" + containerLogstashForwarderConfPath
		sslCertificateMount := " -v " + hostSSLCertificatePath + ":" + containerSSLCertificatePath
		sslKeyMount := " -v " + hostSSLKeyPath + ":" + containerSSLKeyPath

		logstashForwarderMount = logstashForwarderBinaryMount + sslCertificateMount + sslKeyMount + logstashForwarderConfFileMount
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

	// add arguments for environment variables
	environmentVariables := "-e CONTROLPLANE=1"
	environmentVariables = environmentVariables + " -e CONTROLPLANE_SERVICE_ID=" + service.Id
	environmentVariables = environmentVariables + " -e CONTROLPLANE_TENANT_ID=" + tenantId
	environmentVariables = environmentVariables + " -e CONTROLPLANE_CONSUMER_WS=ws://localhost:8444/ws/metrics/store"
	environmentVariables = environmentVariables + " -e CONTROLPLANE_CONSUMER_URL=http://localhost:8444/ws/metrics/store"

	proxyCmd := fmt.Sprintf("/serviced/%s proxy %s '%s'", binary, service.Id, service.Startup)
	//                                   01           02 03    04 05 06 07 08 09 10   01       02               03                    04             05              06                      07          08           09               10
	cmdString := fmt.Sprintf("docker run %s -rm -name=%s %s -v %s %s %s %s %s %s %s", portOps, serviceState.Id, environmentVariables, volumeBinding, requestedMount, logstashForwarderMount, volumeOpts, configFiles, service.ImageId, proxyCmd)
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

// startMissingChildren accepts a zookeeper connection (conn) and a slice of service instance ids (children),
// a map of channels to signal running children stop, and a stateResult channel for children to signal when
// they shutdown
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

// GetInfo creates a Host object from the host this function is running on.
func (a *HostAgent) GetInfo(unused int, host *dao.Host) error {
	hostInfo, err := CurrentContextAsHost("UNKNOWN")
	if err != nil {
		return err
	}
	*host = *hostInfo
	return nil
}

//SendHostIPs handles the details of sending HostIPResources
type SendHostIPs func(ips dao.HostIPs, unused *int) error

/**
RegisterResources registers resources on the host such as IP addresses with the control plane master.
The duration parameter is how often to register with the master
*/
func (a *HostAgent) RegisterIPResources(duration time.Duration) {
	registerFn := func() {
		controlClient, err := NewControlClient(a.master)
		if err != nil {
			glog.Errorf("Could not start ControlPlane client %v", err)
			return
		}
		defer controlClient.Close()
		err = registerIPs(a.hostId, controlClient.RegisterHostIPs)
		if err != nil {
			glog.Errorf("Error registering resources %v", err)
		}
	}
	//do it the first time
	registerFn()
	tc := time.Tick(duration)
	//run in timed loop
	for _ = range tc {
		registerFn()
	}
}

/*
registerIPs does the actual work of determining the IPs on the host. Parameters are the hostId for this host
and the function used to send the found IPs
*/
func registerIPs(hostId string, sendFn SendHostIPs) error {

	interfaces, err := net.Interfaces()
	if err != nil {
		glog.Error("Problem reading interfaces: ", err)
		return err
	}
	hostIPResources := make([]dao.HostIPResource, 0, len(interfaces))
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			glog.Errorf("Problem reading address for interface %s: %s", iface.Name, err)
			return err
		}
		for _, addr := range addrs {
			//send address to Master
			hostIp := dao.HostIPResource{}
			hostIp.IPAddress = addr.String()
			hostIp.InterfaceName = iface.Name
			hostIPResources = append(hostIPResources, hostIp)
		}
	}
	var unused int
	glog.V(4).Infof("Agent registering IPs %v", hostIPResources)
	hostIps := dao.HostIPs{}
	hostIps.HostId = hostId
	hostIps.IPs = hostIPResources
	if err := sendFn(hostIps, &unused); err != nil {
		glog.Errorf("Error registering IPs %v", err)
	}
	return nil
}

// *********************************************************************
// ***** FIXME *********************************************************
// ***** The following three functions are also defined in isvc.go *****
// returns serviced home
func serviceDHome() string {
	return os.Getenv("SERVICED_HOME")
}

func localDir(p string) string {
	homeDir := ServiceDHome()
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = path.Join(path.Dir(filename), "isvcs")
	}
	return path.Join(homeDir, p)
}

func resourcesDir() string {
	return localDir("resources")
}

// *********************************************************************
