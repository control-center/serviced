// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package serviced - agent implements a service that runs on a serviced node.
// It is responsible for ensuring that a particular node is running the correct
// services and reporting the state and health of those services back to the
// master serviced.
package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"

	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/commons/docker"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	coordzk "github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/domain/user"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/rpc/master"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/zzk"
	zkdocker "github.com/zenoss/serviced/zzk/docker"
	zkservice "github.com/zenoss/serviced/zzk/service"
	"github.com/zenoss/serviced/zzk/virtualips"
)

/*
 glog levels:
 0: important info that should always be shown
 1: info that might be important for debugging
 2: very verbose debug info
 3: trace level info
*/

const (
	dockerEndpoint     = "unix:///var/run/docker.sock"
	circularBufferSize = 1000
)

// HostAgent is an instance of the control plane Agent.
type HostAgent struct {
	poolID               string
	master               string               // the connection string to the master agent
	uiport               string               // the port to the ui (legacy was port 8787, now default 443)
	hostID               string               // the hostID of the current host
	dockerDNS            []string             // docker dns addresses
	varPath              string               // directory to store serviced	 data
	mount                []string             // each element is in the form: dockerImage,hostPath,containerPath
	vfs                  string               // driver for container volumes
	currentServices      map[string]*exec.Cmd // the current running services
	mux                  *proxy.TCPMux
	closing              chan interface{}
	proxyRegistry        proxy.ProxyRegistry
	zkClient             *coordclient.Client
	dockerRegistry       string           // the docker registry to use
	periodicTasks        chan interface{} // signal for periodic tasks to stop
	maxContainerAge      time.Duration    // maximum age for a stopped container before it is removed
	virtualAddressSubnet string           // subnet for virtual addresses
}

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

// funcmap provides template functions for evaluating PortTemplate
var funcmap = template.FuncMap{
	"plus": func(a, b int) int {
		return a + b
	},
}

type AgentOptions struct {
	PoolID               string
	Master               string
	UIPort               string
	DockerDNS            []string
	VarPath              string
	Mount                []string
	VFS                  string
	Zookeepers           []string
	Mux                  *proxy.TCPMux
	DockerRegistry       string
	MaxContainerAge      time.Duration // Maximum container age for a stopped container before being removed
	VirtualAddressSubnet string
}

// NewHostAgent creates a new HostAgent given a connection string
func NewHostAgent(options AgentOptions) (*HostAgent, error) {
	// save off the arguments
	agent := &HostAgent{}
	agent.dockerRegistry = options.DockerRegistry
	agent.poolID = options.PoolID
	agent.master = options.Master
	agent.uiport = options.UIPort
	agent.dockerDNS = options.DockerDNS
	agent.varPath = options.VarPath
	agent.mount = options.Mount
	agent.vfs = options.VFS
	agent.mux = options.Mux
	agent.periodicTasks = make(chan interface{})
	agent.maxContainerAge = options.MaxContainerAge
	agent.virtualAddressSubnet = options.VirtualAddressSubnet

	dsn := getZkDSN(options.Zookeepers)
	basePath := ""
	zkClient, err := coordclient.New("zookeeper", dsn, basePath, nil)
	if err != nil {
		return nil, err
	}
	agent.zkClient = zkClient

	agent.closing = make(chan interface{})
	hostID, err := utils.HostID()
	if err != nil {
		panic("Could not get hostid")
	}
	agent.hostID = hostID
	agent.currentServices = make(map[string]*exec.Cmd)

	agent.proxyRegistry = proxy.NewDefaultProxyRegistry()
	go agent.start()
	go agent.reapOldContainersLoop(time.Minute)
	return agent, err

	/* FIXME: this should work here

	addr, err := net.ResolveTCPAddr("tcp", processForwarderAddr)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	sio := shell.NewProcessForwarderServer(proxyOptions.servicedEndpoint)
	sio.Handle("/", http.FileServer(http.Dir("/serviced/www/")))
	go http.Serve(listener, sio)
	c := &ControllerP{
		processForwarderListener: listener,
	}
	*/

}

// Use the Context field of the given template to fill in all the templates in
// the Command fields of the template's ServiceDefinitions
func injectContext(s *service.Service, svcState *servicestate.ServiceState, cp dao.ControlPlane) error {
	getSvc := func(svcID string) (service.Service, error) {
		svc := service.Service{}
		err := cp.GetService(svcID, &svc)
		return svc, err
	}
	findChild := func(svcID, childName string) (service.Service, error) {
		svc := service.Service{}
		err := cp.FindChildService(dao.FindChildRequest{svcID, childName}, &svc)
		return svc, err
	}

	return s.Evaluate(getSvc, findChild, svcState.InstanceID)
}

// Shutdown stops the agent
func (a *HostAgent) Shutdown() {
	glog.V(2).Info("Issuing shutdown signal")
	close(a.periodicTasks) // shut down period tasks
	close(a.closing)
	glog.Info("exiting shutdown")
}

// AttachService attempts to attach to a running container
func (a *HostAgent) AttachService(done chan<- interface{}, service *service.Service, serviceState *servicestate.ServiceState) error {
	dc, err := dockerclient.NewClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("can't create docker client: %v", err)
		return err
	}

	for i := 0; i < 5; i++ {
		// get docker status
		containerState, err := getDockerState(serviceState.DockerID)
		glog.V(2).Infof("Agent.updateCurrentState got container state for docker ID %s: %v", serviceState.DockerID, containerState)

		switch {
		case err == nil && !containerState.State.Running:
			glog.V(1).Infof("Container does not appear to be running: %s", serviceState.ID)
			return errors.New("Container not running for " + serviceState.ID)

		case err != nil && strings.HasPrefix(err.Error(), "no container"):
			glog.Warningf("Error retrieving container state: %s", serviceState.ID)
			return err

		}

		if containerState == nil {
			time.Sleep(time.Second)
			continue
		}
		updateInstance(serviceState, containerState)
		go a.waitInstance(dc, done, service, serviceState)
		return nil
	}
	return fmt.Errorf("could not update container state")
}

// StopService terminates a particular service instance (serviceState) on the localhost.
func (a *HostAgent) StopService(serviceState *servicestate.ServiceState) error {
	return a.dockerTerminate(serviceState.ID)
}

func reapContainers(client *dockerclient.Client, maxAge time.Duration) error {
	containers, lastErr := client.ListContainers(dockerclient.ListContainersOptions{All: true})
	if lastErr != nil {
		return lastErr
	}
	cutoff := time.Now().Add(-maxAge).Unix()
	for _, container := range containers {
		if !strings.HasPrefix(container.Status, "Exited") {
			continue
		}
		if container.Created > cutoff {
			continue
		}
		// attempt to delete the container
		glog.Infof("About to remove container %s", container.ID)
		if err := client.RemoveContainer(dockerclient.RemoveContainerOptions{ID: container.ID}); err != nil {
			lastErr = err
			glog.Errorf("Could not remove container %s: %s", container.ID, err)
		}
	}
	return lastErr
}

func (a *HostAgent) reapOldContainersLoop(interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			dc, err := dockerclient.NewClient(dockerEndpoint)
			if err != nil {
				glog.Errorf("can't create docker client: %v", err)
				continue
			}
			reapContainers(dc, a.maxContainerAge)
		case _, ok := <-a.periodicTasks:
			if !ok {
				return // we are shutting down
			}
		}
	}
}

func (a *HostAgent) dockerRemove(dockerID string) error {
	glog.V(1).Infof("Ensuring that container %s does not exist", dockerID)

	dc, err := dockerclient.NewClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("can't create docker client: %v", err)
		return err
	}

	if err = dc.RemoveContainer(dockerclient.RemoveContainerOptions{ID: dockerID, RemoveVolumes: true}); err != nil {
		glog.Errorf("unable to remove container %s: %v", dockerID, err)
		return err
	}

	glog.V(2).Infof("Successfully removed %s", dockerID)
	return nil
}

func (a *HostAgent) dockerTerminate(dockerID string) error {
	glog.V(1).Infof("Killing container %s", dockerID)

	dc, err := dockerclient.NewClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("can't create docker client: %v", err)
		return err
	}

	if err = dc.KillContainer(dockerclient.KillContainerOptions{dockerID, dockerclient.SIGTERM}); err != nil && !strings.Contains(err.Error(), "No such container") {
		glog.Errorf("unable to kill container %s: %v", dockerID, err)
		return err
	}

	glog.V(2).Infof("Successfully killed %s", dockerID)
	return nil
}

// Get the state of the docker container given the dockerId
func getDockerState(dockerID string) (*dockerclient.Container, error) {
	glog.V(1).Infof("Inspecting container: %s", dockerID)

	dc, err := dockerclient.NewClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("can't create docker client: %v", err)
		return nil, err
	}

	return dc.InspectContainer(dockerID)
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

func (a *HostAgent) waitInstance(dc *dockerclient.Client, procFinished chan<- interface{}, svc *service.Service, state *servicestate.ServiceState) {
	exited := make(chan error)

	go func() {
		defer close(procFinished)
		rc, err := dc.WaitContainer(state.DockerID)
		if err != nil || rc != 0 || glog.GetVerbosity() > 0 {
			// TODO: output of docker logs is potentially very large
			// this should be implemented another way, perhaps a docker attach
			// or extend docker to give last N seconds
			if output, err := exec.Command("docker", "logs", state.DockerID).CombinedOutput(); err != nil {
				glog.Errorf("Could not get logs for container %s", state.DockerID)
			} else {
				var buffersize = 1000
				if index := len(output) - buffersize; index > 0 {
					output = output[index:]
				}
				glog.Warningf("Last %d bytes of container %s: %s", buffersize, state.DockerID, string(output))
			}
		}
		glog.Infof("Docker wait %s exited", state.DockerID)
		// remove the container
		if err := dc.RemoveContainer(dockerclient.RemoveContainerOptions{ID: state.DockerID, RemoveVolumes: true}); err != nil {
			glog.Errorf("Could not remove container %s: %s", state.DockerID, err)
		}
		exited <- err
	}()

	glog.V(4).Infof("Looking for address assignment in service %s (%s)", svc.Name, svc.ID)
	for _, endpoint := range svc.Endpoints {
		if addressConfig := endpoint.GetAssignment(); addressConfig != nil {
			glog.V(4).Infof("Found address assignment for %s:%s endpoint %s", svc.Name, svc.ID, endpoint.Name)
			proxyID := fmt.Sprintf("%v:%v", state.ServiceID, endpoint.Name)

			frontEnd := proxy.ProxyAddress{IP: addressConfig.IPAddr, Port: addressConfig.Port}
			backEnd := proxy.ProxyAddress{IP: state.PrivateIP, Port: endpoint.PortNumber}

			if err := a.proxyRegistry.CreateProxy(proxyID, endpoint.Protocol, frontEnd, backEnd); err != nil {
				glog.Warningf("Could not starte External address proxy for %v; error: proxyId", proxyID, err)
			}
			defer a.proxyRegistry.RemoveProxy(proxyID)
		}
	}

	exitcode, ok := utils.GetExitStatus(<-exited)
	if !ok {
		glog.V(1).Infof("Unable to determine exit code for %s", state.ID)
		return
	}

	switch exitcode {
	case 137:
		glog.V(1).Infof("Docker process killed: %s", state.ID)
	case 2:
		glog.V(1).Infof("Docker process stopped: %s", state.ID)
	case 0:
		glog.V(0).Infof("Process for service state %s finished", state.ID)
	}
}

func updateInstance(state *servicestate.ServiceState, ctr *dockerclient.Container) {
	state.DockerID = ctr.ID
	state.Started = ctr.Created
	state.PrivateIP = ctr.NetworkSettings.IPAddress
	state.PortMapping = make(map[string][]domain.HostIPAndPort)
	for k, v := range ctr.NetworkSettings.Ports {
		pm := []domain.HostIPAndPort{}
		for _, pb := range v {
			pm = append(pm, domain.HostIPAndPort{HostIP: pb.HostIp, HostPort: pb.HostPort})
		}
		state.PortMapping[string(k)] = pm
	}
}

func getSubvolume(varPath, poolID, tenantID, fs string) (*volume.Volume, error) {
	baseDir, _ := filepath.Abs(path.Join(varPath, "volumes"))
	if _, err := volume.Mount(fs, poolID, baseDir); err != nil {
		return nil, err
	}
	baseDir, _ = filepath.Abs(path.Join(varPath, "volumes", poolID))
	return volume.Mount(fs, tenantID, baseDir)
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
		return fmt.Errorf("unsupported owner specification: %s", owner)
	}

	uid, gid, err := getInternalImageIDs(owner, dockerImage)
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

// StartService starts a new instance of the specified service and updates the control plane state accordingly.
func (a *HostAgent) StartService(done chan<- interface{}, service *service.Service, serviceState *servicestate.ServiceState) error {
	glog.V(2).Infof("About to start service %s with name %s", service.ID, service.Name)
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer client.Close()

	// start from a known good state
	a.dockerTerminate(serviceState.ID)
	a.dockerRemove(serviceState.ID)

	dc, err := dockerclient.NewClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("can't create Docker client: %v ", err)
		return err
	}

	em, err := dc.MonitorEvents()
	if err != nil {
		glog.Errorf("can't monitor Docker events: %v", err)
		return err
	}
	defer em.Close()

	// create the docker client Config and HostConfig structures necessary to create and start the service
	config, hostconfig, err := configureContainer(a, client, service, serviceState, a.virtualAddressSubnet)
	if err != nil {
		glog.Errorf("can't configure container: %v", err)
		return err
	}

	cjson, _ := json.MarshalIndent(config, "", "     ")
	glog.V(3).Infof(">>> CreateContainerOptions:\n%s", string(cjson))

	hcjson, _ := json.MarshalIndent(hostconfig, "", "     ")
	glog.V(2).Infof(">>> HostConfigOptions:\n%s", string(hcjson))

	// pull the image from the registry first if necessary, then attempt to create the container.
	registry, err := docker.NewDockerRegistry(a.dockerRegistry)
	if err != nil {
		glog.Errorf("can't use docker registry %s: %s", a.dockerRegistry, err)
		return err
	}
	ctr, err := docker.CreateContainer(*registry, dc, dockerclient.CreateContainerOptions{Name: serviceState.ID, Config: config})
	if err != nil {
		glog.Errorf("can't create container %v: %v", config, err)
		return err
	}

	glog.V(2).Infof("container %s created  Name:%s for service Name:%s ID:%s Cmd:%+v", ctr.ID, serviceState.ID, service.Name, service.ID, config.Cmd)

	// use the docker client EventMonitor to listen for events from this container
	s, err := em.Subscribe(ctr.ID)
	if err != nil {
		glog.Errorf("can't subscribe to Docker events on container %s: %v", ctr.ID, err)
		return err
	}

	emc := make(chan struct{})

	s.Handle(dockerclient.Start, func(e dockerclient.Event) error {
		glog.V(2).Infof("container %s starting Name:%s for service Name:%s ID:%s Cmd:%+v", e["id"], serviceState.ID, service.Name, service.ID, config.Cmd)
		emc <- struct{}{}
		return nil
	})

	err = dc.StartContainer(ctr.ID, hostconfig)
	if err != nil {
		glog.Errorf("can't start container %s for service Name:%s ID:%s error: %v", ctr.ID, service.Name, service.ID, err)
		return err
	}

	// wait until we get notified that the container is started, or ten seconds, whichever comes first.
	// TODO: make the timeout configurable
	timeout := 10 * time.Second
	tout := time.After(timeout)
	select {
	case <-emc:
		glog.V(0).Infof("container %s started  Name:%s for service Name:%s ID:%s", ctr.ID, serviceState.ID, service.Name, service.ID)
	case <-tout:
		glog.Warningf("container %s start timed out after %v Name:%s for service Name:%s ID:%s Cmd:%+v", ctr.ID, timeout, serviceState.ID, service.Name, service.ID, config.Cmd)
		// FIXME: WORKAROUND for issue where dockerclient.Start event doesn't always notify
		if container, err := dc.InspectContainer(ctr.ID); err != nil {
			glog.Warning("container %s could not be inspected error:%v\n\n", ctr.ID, err)
		} else {
			glog.Warningf("container %s inspected State:%+v", ctr.ID, container.State)
			if container.State.Running == true {
				glog.Infof("container %s start event timed out, but is running - will not return start timed out", ctr.ID)
				break
			}
		}
		return fmt.Errorf("start timed out")
	}

	ctr, err = getDockerState(ctr.ID)
	if err != nil {
		glog.Errorf("Problem getting service state for %s: %v", serviceState.ID, err)
		return err
	}

	glog.V(2).Infof("container %s a.waitForProcessToDie", ctr.ID)
	updateInstance(serviceState, ctr)
	go a.waitInstance(dc, done, service, serviceState)
	return nil
}

// configureContainer creates and populates two structures, a docker client Config and a docker client HostConfig structure
// that are used to create and start a container respectively. The information used to populate the structures is pulled from
// the service, serviceState, and conn values that are passed into configureContainer.
func configureContainer(a *HostAgent, client *ControlClient,
	svc *service.Service, serviceState *servicestate.ServiceState,
	virtualAddressSubnet string) (*dockerclient.Config, *dockerclient.HostConfig, error) {
	cfg := &dockerclient.Config{}
	hcfg := &dockerclient.HostConfig{}

	//get this service's tenantId for volume mapping
	var tenantID string
	err := client.GetTenantId(svc.ID, &tenantID)
	if err != nil {
		glog.Errorf("Failed getting tenantID for service: %s, %s", svc.ID, err)
	}

	// get the system user
	unused := 0
	systemUser := user.User{}
	err = client.GetSystemUser(unused, &systemUser)
	if err != nil {
		glog.Errorf("Unable to get system user account for agent %s", err)
	}
	glog.V(1).Infof("System User %v", systemUser)

	cfg.Image = svc.ImageID

	// get the endpoints
	cfg.ExposedPorts = make(map[dockerclient.Port]struct{})
	hcfg.PortBindings = make(map[dockerclient.Port][]dockerclient.PortBinding)

	if svc.Endpoints != nil {
		glog.V(1).Info("Endpoints for service: ", svc.Endpoints)
		for _, endpoint := range svc.Endpoints {
			if endpoint.Purpose == "export" { // only expose remote endpoints
				var port uint16
				port = endpoint.PortNumber
				if endpoint.PortTemplate != "" {
					t := template.Must(template.New("PortTemplate").Funcs(funcmap).Parse(endpoint.PortTemplate))
					b := bytes.Buffer{}
					err := t.Execute(&b, serviceState)
					if err == nil {
						j, err := strconv.Atoi(b.String())
						if err != nil {
							glog.Errorf("%+v", err)
						} else if j > 0 {
							port = uint16(j)
						}
					}
				}
				var p string
				switch endpoint.Protocol {
				case commons.UDP:
					p = fmt.Sprintf("%d/%s", port, "udp")
				default:
					p = fmt.Sprintf("%d/%s", port, "tcp")
				}
				cfg.ExposedPorts[dockerclient.Port(p)] = struct{}{}
				hcfg.PortBindings[dockerclient.Port(p)] = append(hcfg.PortBindings[dockerclient.Port(p)], dockerclient.PortBinding{})
			}
		}
	}

	if len(tenantID) == 0 && len(svc.Volumes) > 0 {
		// FIXME: find a better way of handling this error condition
		glog.Fatalf("Could not get tenant ID and need to mount a volume, service state: %s, service id: %s", serviceState.ID, svc.ID)
	}

	// Make sure the image exists locally.
	registry, err := docker.NewDockerRegistry(a.dockerRegistry)
	if err != nil {
		glog.Errorf("Error using docker registry %s: %s", a.dockerRegistry, err)
		return nil, nil, err
	}
	dc, err := dockerclient.NewClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("can't create docker client: %v", err)
		return nil, nil, err
	}
	if _, err = docker.InspectImage(*registry, dc, svc.ImageID); err != nil {
		glog.Errorf("can't inspect docker image %s: %s", svc.ImageID, err)
		return nil, nil, err
	}

	cfg.Volumes = make(map[string]struct{})
	hcfg.Binds = []string{}

	if err := injectContext(svc, serviceState, client); err != nil {
		glog.Errorf("Error injecting context: %s", err)
		return nil, nil, err
	}

	for _, volume := range svc.Volumes {
		if volume.Type != "" && volume.Type != "dfs" {
			continue
		}

		resourcePath, err := a.setupVolume(tenantID, svc, volume)
		if err != nil {
			glog.Fatalf("%s", err)
		}

		binding := fmt.Sprintf("%s:%s", resourcePath, volume.ContainerPath)
		cfg.Volumes[strings.Split(binding, ":")[1]] = struct{}{}
		hcfg.Binds = append(hcfg.Binds, strings.TrimSpace(binding))
	}

	dir, binary, err := ExecPath()
	if err != nil {
		glog.Errorf("Error getting exec path: %v", err)
		return nil, nil, err
	}
	volumeBinding := fmt.Sprintf("%s:/serviced", dir)
	cfg.Volumes[strings.Split(volumeBinding, ":")[1]] = struct{}{}
	hcfg.Binds = append(hcfg.Binds, strings.TrimSpace(volumeBinding))

	// bind mount everything we need for logstash-forwarder
	if len(svc.LogConfigs) != 0 {
		const LOGSTASH_CONTAINER_DIRECTORY = "/usr/local/serviced/resources/logstash"
		logstashPath := utils.ResourcesDir() + "/logstash"
		binding := fmt.Sprintf("%s:%s", logstashPath, LOGSTASH_CONTAINER_DIRECTORY)
		cfg.Volumes[LOGSTASH_CONTAINER_DIRECTORY] = struct{}{}
		hcfg.Binds = append(hcfg.Binds, binding)
		glog.V(1).Infof("added logstash bind mount: %s", binding)
	}

	// specify temporary volume paths for docker to create
	tmpVolumes := []string{"/tmp"}
	for _, volume := range svc.Volumes {
		if volume.Type == "tmp" {
			tmpVolumes = append(tmpVolumes, volume.ContainerPath)
		}
	}
	for _, path := range tmpVolumes {
		cfg.Volumes[path] = struct{}{}
		glog.V(0).Infof("added temporary docker container path: %s", path)
	}

	// add arguments to mount requested directory (if requested)
	glog.V(2).Infof("Checking Mount options for service %#v", svc)
	for _, bindMountString := range a.mount {
		glog.V(2).Infof("bindmount is  %#v", bindMountString)
		splitMount := strings.Split(bindMountString, ",")
		numMountArgs := len(splitMount)

		if numMountArgs == 2 || numMountArgs == 3 {

			requestedImage := splitMount[0]
			glog.V(2).Infof("mount requestedImage %#v", requestedImage)
			hostPath := splitMount[1]
			glog.V(2).Infof("mount hostPath %#v", hostPath)
			// assume the container path is going to be the same as the host path
			containerPath := hostPath

			// if the container path is provided, use it
			if numMountArgs > 2 {
				containerPath = splitMount[2]
			}
			glog.V(2).Infof("mount containerPath %#v", containerPath)

			// insert tenantId into requestedImage - see facade.DeployService
			matchedRequestedImage := false
			if requestedImage == "*" {
				matchedRequestedImage = true
			} else {
				imageID, err := commons.ParseImageID(requestedImage)
				if err != nil {
					glog.Errorf("error parsing imageid %v: %v", requestedImage, err)
					continue
				}
				svcImageID, err := commons.ParseImageID(svc.ImageID)
				if err != nil {
					glog.Errorf("error parsing service imageid %v; %v", svc.ImageID, err)
					continue
				}
				glog.V(2).Infof("mount checking %#v and %#v ", imageID, svcImageID)
				matchedRequestedImage = (imageID.Repo == svcImageID.Repo)
			}

			if matchedRequestedImage {
				binding := fmt.Sprintf("%s:%s", hostPath, containerPath)
				cfg.Volumes[strings.Split(binding, ":")[1]] = struct{}{}
				hcfg.Binds = append(hcfg.Binds, strings.TrimSpace(binding))
			}
		} else {
			glog.Warningf("Could not bind mount the following: %s", bindMountString)
		}
	}

	// Get host IP
	ips, err := utils.GetIPv4Addresses()
	if err != nil {
		glog.Errorf("Error getting host IP addresses: %v", err)
		return nil, nil, err
	}

	// add arguments for environment variables
	cfg.Env = append([]string{},
		fmt.Sprintf("CONTROLPLANE_SYSTEM_USER=%s", systemUser.Name),
		fmt.Sprintf("CONTROLPLANE_SYSTEM_PASSWORD=%s", systemUser.Password),
		fmt.Sprintf("CONTROLPLANE_HOST_IPS='%s'", strings.Join(ips, " ")),
		fmt.Sprintf("SERVICED_VIRTUAL_ADDRESS_SUBNET=%s", virtualAddressSubnet),
		fmt.Sprintf("SERVICED_IS_SERVICE_SHELL=false"),
		fmt.Sprintf("SERVICED_NOREGISTRY=%s", os.Getenv("SERVICED_NOREGISTRY")),
		fmt.Sprintf("SERVICED_SERVICE_IMAGE=%s", svc.ImageID))

	// add dns values to setup
	for _, addr := range a.dockerDNS {
		_addr := strings.TrimSpace(addr)
		if len(_addr) > 0 {
			cfg.Dns = append(cfg.Dns, addr)
		}
	}

	// Add hostname if set
	if svc.Hostname != "" {
		cfg.Hostname = svc.Hostname
	}

	cfg.Cmd = append([]string{},
		fmt.Sprintf("/serviced/%s", binary),
		"service",
		"proxy",
		svc.ID,
		strconv.Itoa(serviceState.InstanceID),
		svc.Startup)

	if svc.Privileged {
		hcfg.Privileged = true
	}

	return cfg, hcfg, nil
}

// setupVolume
func (a *HostAgent) setupVolume(tenantID string, service *service.Service, volume servicedefinition.Volume) (string, error) {
	glog.V(4).Infof("setupVolume for service Name:%s ID:%s", service.Name, service.ID)
	sv, err := getSubvolume(a.varPath, service.PoolID, tenantID, a.vfs)
	if err != nil {
		return "", fmt.Errorf("Could not create subvolume: %s", err)
	}

	resourcePath := path.Join(sv.Path(), volume.ResourcePath)
	if err = os.MkdirAll(resourcePath, 0770); err != nil {
		return "", fmt.Errorf("Could not create resource path: %s, %s", resourcePath, err)
	}

	if err := createVolumeDir(resourcePath, volume.ContainerPath, service.ImageID, volume.Owner, volume.Permission); err != nil {
		glog.Errorf("Error populating resource path: %s with container path: %s, %v", resourcePath, volume.ContainerPath, err)
	}

	glog.V(4).Infof("resourcePath: %s  containerPath: %s", resourcePath, volume.ContainerPath)
	return resourcePath, nil
}

func (a *HostAgent) GetHost(hostID string) (*host.Host, error) {
	rpcMaster, err := master.NewClient(a.master)
	if err != nil {
		glog.Errorf("Failed to get RPC master: %v", err)
		return nil, err
	}
	defer rpcMaster.Close()
	myHost, err := rpcMaster.GetHost(hostID)
	if err != nil {
		glog.Errorf("Could not get host %s: %s", hostID, err)
		return nil, err
	}
	return myHost, nil
}

// main loop of the HostAgent
func (a *HostAgent) start() {
	glog.Info("Starting HostAgent")
	for {
		connc := make(chan coordclient.Connection)
		go func() {
			for {
				c, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(a.poolID))
				if err == nil {
					connc <- c
					return
				}

				select {
				case <-a.closing:
					return
				case <-time.After(time.Second):
				}
			}
		}()

		// create a wrapping function so that client.Close() can be handled via defer
		func(conn coordclient.Connection) {
			glog.Info("Got a connected client")
			// watch virtual IP zookeeper nodes
			virtualIPListener := virtualips.NewVirtualIPListener(conn)

			// watch docker action nodes
			actionListener := zkdocker.NewActionListener(conn, a, a.hostID)

			// watch the host state nodes
			// this blocks until
			// 1) has a connection
			// 2) its node is registered
			// 3) receieves signal to shutdown or breaks
			hsListener := zkservice.NewHostStateListener(conn, a, a.hostID)

			zzk.Start(a.closing, hsListener, virtualIPListener, actionListener)
		}(<-connc)
		select {
		case <-a.closing:
			return
		default:
			// this will not spin infinitely
		}
	}
}

// AttachAndRun implements zkdocker.ActionHandler; it attaches to a running
// container and performs a command as specified by the container's service
// definition
func (a *HostAgent) AttachAndRun(dockerID string, command []string) ([]byte, error) {
	return utils.AttachAndRun(dockerID, command)
}

type stateResult struct {
	id  string
	err error
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
