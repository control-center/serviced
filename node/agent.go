// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package serviced - agent implements a service that runs on a serviced node.
// It is responsible for ensuring that a particular node is running the correct
// services and reporting the state and health of those services back to the
// master serviced.
package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/iptables"
	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/zzk"
	zkdocker "github.com/control-center/serviced/zzk/docker"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/virtualips"
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

// HostAgent is an instance of the control center Agent.
type HostAgent struct {
	poolID               string
	master               string               // the connection string to the master agent
	uiport               string               // the port to the ui (legacy was port 8787, now default 443)
	rpcport              string               // the rpc port to serviced (default is 4979)
	hostID               string               // the hostID of the current host
	dockerDNS            []string             // docker dns addresses
	storage              volume.Driver        // driver supporting the application data
	storageTenants       []string             // tenants we have mounted
	mount                []string             // each element is in the form: dockerImage,hostPath,containerPath
	currentServices      map[string]*exec.Cmd // the current running services
	mux                  *proxy.TCPMux
	useTLS               bool // Whether the mux uses TLS
	proxyRegistry        proxy.ProxyRegistry
	zkClient             *coordclient.Client
	maxContainerAge      time.Duration   // maximum age for a stopped container before it is removed
	virtualAddressSubnet string          // subnet for virtual addresses
	servicedChain        *iptables.Chain // Assigned IP rule chain
	controllerBinary     string          // Path to the controller binary
	logstashURL          string
	dockerLogDriver      string
	dockerLogConfig      map[string]string
	pullreg              registry.Registry
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

type AgentOptions struct {
	PoolID               string
	Master               string
	UIPort               string
	RPCPort              string
	DockerDNS            []string
	VolumesPath          string
	Mount                []string
	FSType               volume.DriverType
	Zookeepers           []string
	Mux                  *proxy.TCPMux
	UseTLS               bool
	DockerRegistry       string
	MaxContainerAge      time.Duration // Maximum container age for a stopped container before being removed
	VirtualAddressSubnet string
	ControllerBinary     string
	LogstashURL          string
	DockerLogDriver      string
	DockerLogConfig      map[string]string
}

// NewHostAgent creates a new HostAgent given a connection string
func NewHostAgent(options AgentOptions, reg registry.Registry) (*HostAgent, error) {
	// save off the arguments
	agent := &HostAgent{}
	agent.poolID = options.PoolID
	agent.master = options.Master
	agent.uiport = options.UIPort
	agent.rpcport = options.RPCPort
	agent.dockerDNS = options.DockerDNS
	agent.mount = options.Mount
	agent.mux = options.Mux
	agent.useTLS = options.UseTLS
	agent.maxContainerAge = options.MaxContainerAge
	agent.virtualAddressSubnet = options.VirtualAddressSubnet
	agent.servicedChain = iptables.NewChain("SERVICED")
	agent.controllerBinary = options.ControllerBinary
	agent.logstashURL = options.LogstashURL
	agent.dockerLogDriver = options.DockerLogDriver
	agent.dockerLogConfig = options.DockerLogConfig

	var err error
	dsn := getZkDSN(options.Zookeepers)
	if agent.zkClient, err = coordclient.New("zookeeper", dsn, "", nil); err != nil {
		return nil, err
	}
	if agent.storage, err = volume.GetDriver(options.VolumesPath); err != nil {
		glog.Errorf("Could not load storage driver at %s: %s", options.VolumesPath, err)
		return nil, err
	}
	if agent.hostID, err = utils.HostID(); err != nil {
		panic("Could not get hostid")
	}
	agent.currentServices = make(map[string]*exec.Cmd)
	agent.proxyRegistry = proxy.NewDefaultProxyRegistry()
	agent.pullreg = reg
	return agent, err
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

// AttachService attempts to attach to a running container
func (a *HostAgent) AttachService(svc *service.Service, state *servicestate.ServiceState, exited func(string)) error {
	ctr, err := docker.FindContainer(state.DockerID)
	if err != nil {
		glog.Infof("Can not find docker container %s for service %s (%s) ServiceStateID=%s", state.DockerID, svc.Name, svc.ID, state.ID)
		state.DockerID = "" // CC-1341 - don't try to find this container again.
		return err
	}

	if !ctr.IsRunning() {
		defer exited(state.ID)
		return nil
	}

	ctr.OnEvent(docker.Die, func(cid string) {
		defer exited(state.ID)
		glog.Infof("Instance %s (%s) for %s (%s) has died", state.ID, ctr.ID, svc.Name, svc.ID)
		state.DockerID = cid
		a.removeInstance(state.ID, ctr)
	})

	go a.setProxy(svc, ctr)
	return nil
}

// PauseService pauses a running service
func (a *HostAgent) PauseService(service *service.Service, state *servicestate.ServiceState) error {
	return attachAndRun(state.DockerID, service.Snapshot.Pause)
}

// ResumeService resumes a paused service
func (a *HostAgent) ResumeService(service *service.Service, state *servicestate.ServiceState) error {
	return attachAndRun(state.DockerID, service.Snapshot.Resume)
}

func attachAndRun(dockerID, command string) error {
	if dockerID == "" {
		return errors.New("missing docker ID")
	} else if command == "" {
		return nil
	}

	output, err := utils.AttachAndRun(dockerID, []string{command})
	if err != nil {
		err = fmt.Errorf("%s (%s)", string(output), err)
		glog.Errorf("Could not pause container %s: %s", dockerID, err)
	}
	return err
}

// StopService terminates a particular service instance (serviceState) on the localhost.
func (a *HostAgent) StopService(state *servicestate.ServiceState) error {
	if state == nil || state.DockerID == "" {
		return errors.New("missing Docker ID")
	}

	ctr, err := docker.FindContainer(state.DockerID)
	if err != nil {
		return err
	}

	return ctr.Stop(45 * time.Second)
}

// Get the state of the docker container given the dockerId
func getDockerState(dockerID string) (*docker.Container, error) {
	glog.V(1).Infof("Inspecting container: %s", dockerID)
	return docker.FindContainer(dockerID)
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

// PullImage pulls the image into the local repository.  Returns the current image UUID string
func (a *HostAgent) PullImage(cancel <-chan time.Time, imageID string) (string, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("Could not get zk connection: %s", err)
		return "", err
	}
	a.pullreg.SetConnection(conn)
	if err = a.pullreg.PullImage(cancel, imageID); err != nil {
		return "", err
	}
	uuid, err := registry.GetImageUUID(conn, imageID)
	if err != nil {
		return "", fmt.Errorf("unable to get image UUID: %s", err)
	}
	return uuid, nil
}

// StartService starts a new instance of the specified service and updates the control center state accordingly.
func (a *HostAgent) StartService(svc *service.Service, state *servicestate.ServiceState, exited func(string)) error {
	handlerInstalled := false
	defer func() {
		if !handlerInstalled {
			exited(state.ID)
		}
	}()

	glog.V(2).Infof("About to start service %s with name %s", svc.ID, svc.Name)
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer client.Close()

	// start from a known good state
	if state.DockerID != "" {
		if ctr, err := docker.FindContainer(state.DockerID); err != nil {
			glog.Errorf("Could not find container %s for %s", state.DockerID, state.ID)
		} else if err := ctr.Delete(true); err != nil {
			glog.Errorf("Could not delete container %s for %s", state.DockerID, state.ID)
		}
	}

	// create the docker client Config and HostConfig structures necessary to create and start the service
	config, hostconfig, err := configureContainer(a, client, svc, state, a.virtualAddressSubnet)
	if err != nil {
		glog.Errorf("can't configure container: %v", err)
		return err
	}

	cjson, _ := json.MarshalIndent(config, "", "     ")
	glog.V(3).Infof(">>> CreateContainerOptions:\n%s", string(cjson))

	hcjson, _ := json.MarshalIndent(hostconfig, "", "     ")
	glog.V(3).Infof(">>> HostConfigOptions:\n%s", string(hcjson))

	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{Name: state.ID, Config: config},
		*hostconfig,
	}

	ctr, err := docker.NewContainer(cd, false, 10*time.Second, nil, nil)
	if err != nil {
		glog.Errorf("Error trying to create container %v: %v", config, err)
		return err
	}

	startLock := &sync.Mutex{}
	startLock.Lock()
	ctr.OnEvent(docker.Start, func(cid string) {
		glog.Infof("Instance %s (%s) for %s (%s) has started", state.ID, ctr.ID, svc.Name, svc.ID)
		startLock.Unlock()
	})

	ctr.OnEvent(docker.Die, func(cid string) {
		defer exited(state.ID)
		glog.Infof("Instance %s (%s) for %s (%s) has died", state.ID, ctr.ID, svc.Name, svc.ID)
		state.DockerID = cid
		a.removeInstance(state.ID, ctr)
	})
	handlerInstalled = true

	if err := ctr.Start(); err != nil {
		glog.Errorf("Could not start service state %s (%s) for service %s (%s): %s", state.ID, ctr.ID, svc.Name, svc.ID, err)
		a.removeInstance(state.ID, ctr)
		return err
	}

	startLock.Lock()
	if err := updateInstance(state, ctr); err != nil {
		glog.Errorf("Could not update instance %s (%s) for service %s (%s): %s", state.ID, ctr.ID, svc.Name, svc.ID, err)
		ctr.Stop(45 * time.Second)
		return err
	}

	go a.setProxy(svc, ctr)
	return nil
}

func manageTransparentProxy(endpoint *service.ServiceEndpoint, addressConfig *addressassignment.AddressAssignment, ctr *docker.Container, isDelete bool) error {
	var appendOrDeleteFlag string
	if isDelete {
		appendOrDeleteFlag = "-D"
	} else {
		appendOrDeleteFlag = "-A"
	}
	return exec.Command(
		"iptables",
		"-t", "nat",
		appendOrDeleteFlag, "PREROUTING",
		"-d", fmt.Sprintf("%s", addressConfig.IPAddr),
		"-p", endpoint.Protocol,
		"--dport", fmt.Sprintf("%d", addressConfig.Port),
		"-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", ctr.NetworkSettings.IPAddress, endpoint.PortNumber),
	).Run()
}

func (a *HostAgent) setProxy(svc *service.Service, ctr *docker.Container) {
	glog.V(4).Infof("Looking for address assignment in service %s (%s)", svc.Name, svc.ID)
	for _, endpoint := range svc.Endpoints {
		if addressConfig := endpoint.GetAssignment(); addressConfig != nil {
			glog.V(4).Infof("Found address assignment for %s: %s endpoint %s", svc.Name, svc.ID, endpoint.Name)
			frontendAddress := iptables.NewAddress(addressConfig.IPAddr, int(addressConfig.Port))
			backendAddress := iptables.NewAddress(ctr.NetworkSettings.IPAddress, int(endpoint.PortNumber))

			if err := a.servicedChain.Forward(iptables.Add, endpoint.Protocol, frontendAddress, backendAddress); err != nil {
				glog.Warningf("Could not start external address proxy for %s:%s: %s", svc.ID, endpoint.Name, err)
			}

			//local variables, not loop var, for the defer func
			protocol := endpoint.Protocol
			endpointName := endpoint.Name
			serviceID := svc.ID
			defer func() {
				if err := a.servicedChain.Forward(iptables.Delete, protocol, frontendAddress, backendAddress); err != nil {
					glog.Warningf("Could not remove external address proxy for %s:%s: %s", serviceID, endpointName, err)
				}
			}()
		}
	}
	ctr.Wait(time.Hour * 24 * 365)
}

func (a *HostAgent) removeInstance(stateID string, ctr *docker.Container) {
	rc, err := ctr.Wait(time.Second)
	if err != nil || rc != 0 || glog.GetVerbosity() > 0 {
		// TODO: output of docker logs is potentially very large
		// this should be implemented another way, perhaps a docker attach
		// or extend docker to give last N seconds
		if output, err := exec.Command("docker", "logs", "--tail", "10000", ctr.ID).CombinedOutput(); err != nil {
			glog.Errorf("Could not get logs for container %s", ctr.ID)
		} else {
			prefix := fmt.Sprintf("ctr-%s: ", ctr.ID[0:5])
			split := strings.Split(string(output), "\n")
			for i, s := range split {
				split[i] = prefix + s
			}
			final := strings.Join(split, "\n")
			glog.Warningf("Last 10000 lines of container %s:\n %s", ctr.ID, string(final))
		}
	}
	if ctr.IsRunning() {
		glog.Errorf("Instance %s (%s) is still running, killing container", stateID, ctr.ID)
		ctr.Kill()
	}
	if err := ctr.Delete(true); err != nil {
		glog.Errorf("Could not remove instance %s (%s): %s", stateID, ctr.ID, err)
	}
	glog.Infof("Service state %s (%s) received exit code %d", stateID, ctr.ID, rc)

}

func updateInstance(state *servicestate.ServiceState, ctr *docker.Container) error {
	if _, err := ctr.Inspect(); err != nil {
		return err
	}
	state.DockerID = ctr.ID
	state.Started = ctr.Created
	state.PrivateIP = ctr.NetworkSettings.IPAddress
	state.PortMapping = make(map[string][]domain.HostIPAndPort)
	for k, v := range ctr.NetworkSettings.Ports {
		pm := []domain.HostIPAndPort{}
		for _, pb := range v {
			pm = append(pm, domain.HostIPAndPort{HostIP: pb.HostIP, HostPort: pb.HostPort})
			state.PortMapping[string(k)] = pm
		}
	}
	return nil
}

// configureContainer creates and populates two structures, a docker client Config and a docker client HostConfig structure
// that are used to create and start a container respectively. The information used to populate the structures is pulled from
// the service, serviceState, and conn values that are passed into configureContainer.
func configureContainer(a *HostAgent, client dao.ControlPlane,
	svc *service.Service, serviceState *servicestate.ServiceState,
	virtualAddressSubnet string) (*dockerclient.Config, *dockerclient.HostConfig, error) {
	cfg := &dockerclient.Config{}
	hcfg := &dockerclient.HostConfig{}

	//get this service's tenantId for volume mapping
	var tenantID string
	err := client.GetTenantId(svc.ID, &tenantID)
	if err != nil {
		glog.Errorf("Failed getting tenantID for service: %s, %s", svc.ID, err)
		return nil, nil, err
	}

	// get the system user
	unused := 0
	systemUser := user.User{}
	err = client.GetSystemUser(unused, &systemUser)
	if err != nil {
		glog.Errorf("Unable to get system user account for agent %s", err)
		return nil, nil, err
	}
	glog.V(1).Infof("System User %v", systemUser)

	cfg.User = "root"
	cfg.WorkingDir = "/"
	if cfg.Image, err = a.pullreg.ImagePath(svc.ImageID); err != nil {
		glog.Errorf("Could not parse image %s: %s", svc.ImageID, err)
		return nil, nil, err
	}

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
					port, err = serviceState.EvalPortTemplate(endpoint.PortTemplate)
					if err != nil {
						glog.Errorf("Unable to interpret ContainerPort: %s", err)
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

	bindsMap := make(map[string]string) // map to prevent duplicate path assignments. Use to populate hcfg.Binds later.

	if err := injectContext(svc, serviceState, client); err != nil {
		glog.Errorf("Error injecting context: %s", err)
		return nil, nil, err
	}

	// iterate svc.Volumes - create bindings for non-dfs volumes
	for _, volume := range svc.Volumes {
		if volume.Type != "" && volume.Type != "dfs" {
			continue
		}

		resourcePath, err := a.setupVolume(tenantID, svc, volume)
		if err != nil {
			return nil, nil, err
		}

		resourcePath = strings.TrimSpace(resourcePath)
		containerPath := strings.TrimSpace(volume.ContainerPath)
		bindsMap[containerPath] = resourcePath
	}

	// mount serviced path
	dir, _, err := ExecPath()
	if err != nil {
		glog.Errorf("Error getting exec path: %v", err)
		return nil, nil, err
	}

	dir, binary := filepath.Split(a.controllerBinary)
	resourcePath := strings.TrimSpace(dir)
	containerPath := strings.TrimSpace("/serviced")
	bindsMap[containerPath] = resourcePath

	// bind mount everything we need for logstash-forwarder
	if len(svc.LogConfigs) != 0 {
		const LOGSTASH_CONTAINER_DIRECTORY = "/usr/local/serviced/resources/logstash"
		logstashPath := utils.ResourcesDir() + "/logstash"
		resourcePath := strings.TrimSpace(logstashPath)
		containerPath := strings.TrimSpace(LOGSTASH_CONTAINER_DIRECTORY)
		bindsMap[containerPath] = resourcePath
		glog.V(1).Infof("added logstash bind mount: %s", fmt.Sprintf("%s:%s", resourcePath, containerPath))
	}

	// specify temporary volume paths for docker to create
	tmpVolumes := []string{"/tmp"}
	for _, volume := range svc.Volumes {
		if volume.Type == "tmp" {
			tmpVolumes = append(tmpVolumes, volume.ContainerPath)
		}
	}
	for _, path := range tmpVolumes {
		glog.V(4).Infof("added temporary docker container path: %s", path)
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
				hostPath = strings.TrimSpace(hostPath)
				containerPath = strings.TrimSpace(containerPath)
				bindsMap[containerPath] = hostPath
			}
		} else {
			glog.Warningf("Could not bind mount the following: %s", bindMountString)
		}
	}

	// transfer bindsMap to hcfg.Binds
	hcfg.Binds = []string{}
	for containerPath, hostPath := range bindsMap {
		binding := fmt.Sprintf("%s:%s", hostPath, containerPath)
		hcfg.Binds = append(hcfg.Binds, binding)
	}

	// Get host IP
	ips, err := utils.GetIPv4Addresses()
	if err != nil {
		glog.Errorf("Error getting host IP addresses: %v", err)
		return nil, nil, err
	}

	// XXX: Hopefully temp fix for CC-1384 & CC-1631 (docker/docker issue 14203).
	count := rand.Intn(128)
	fix := ""
	for i := 0; i < count; i++ {
		fix += "."
	}
	// End temp fix part 1. See immediately below for part 2.

	// add arguments for environment variables
	cfg.Env = append([]string{},
		fmt.Sprintf("CONTROLPLANE_SYSTEM_USER=%s", systemUser.Name),
		fmt.Sprintf("CONTROLPLANE_SYSTEM_PASSWORD=%s", systemUser.Password),
		fmt.Sprintf("CONTROLPLANE_HOST_IPS='%s'", strings.Join(ips, " ")),
		fmt.Sprintf("SERVICED_VIRTUAL_ADDRESS_SUBNET=%s", virtualAddressSubnet),
		fmt.Sprintf("SERVICED_IS_SERVICE_SHELL=false"),
		fmt.Sprintf("SERVICED_NOREGISTRY=%s", os.Getenv("SERVICED_NOREGISTRY")),
		fmt.Sprintf("SERVICED_SERVICE_IMAGE=%s", svc.ImageID),
		fmt.Sprintf("SERVICED_MAX_RPC_CLIENTS=1"),
		fmt.Sprintf("SERVICED_RPC_PORT=%s", a.rpcport),
		fmt.Sprintf("SERVICED_LOG_ADDRESS=%s", a.logstashURL),
		fmt.Sprintf("SERVICED_UI_PORT=%s", strings.Split(a.uiport, ":")[1]),
		fmt.Sprintf("SERVICED_MASTER_IP=%s", strings.Split(a.master, ":")[0]),
		fmt.Sprintf("TZ=%s", os.Getenv("TZ")),
		// XXX: Hopefully temp fix for CC-1384 & CC-1631 (docker/docker issue 14203).
		fmt.Sprintf("DOCKER_14203_FIX='%s'", fix),
		// End temp fix part 2. See immediately above for part 1.
	)

	// add dns values to setup
	for _, addr := range a.dockerDNS {
		_addr := strings.TrimSpace(addr)
		if len(_addr) > 0 {
			cfg.DNS = append(cfg.DNS, addr)
		}
	}

	// Add hostname if set
	if svc.Hostname != "" {
		cfg.Hostname = svc.Hostname
	}

	cfg.Cmd = append([]string{},
		filepath.Join("/serviced", binary),
		svc.ID,
		strconv.Itoa(serviceState.InstanceID),
		svc.Startup)

	if svc.Privileged {
		hcfg.Privileged = true
	}

	// Memory and CpuShares should never be negative
	if svc.MemoryLimit < 0 {
		cfg.Memory = 0
	} else {
		cfg.Memory = int64(svc.MemoryLimit)
	}

	if svc.CPUShares < 0 {
		cfg.CPUShares = 0
	} else {
		cfg.CPUShares = svc.CPUShares
	}

	hcfg.LogConfig.Type = a.dockerLogDriver
	hcfg.LogConfig.Config = a.dockerLogConfig

	// CC-1848: set core ulimit to 0
	hcfg.Ulimits = []dockerclient.ULimit{
		{
			Name: "core",
			Soft: 0,
			Hard: 0,
		},
	}

	return cfg, hcfg, nil
}

// setupVolume
func (a *HostAgent) setupVolume(tenantID string, service *service.Service, volume servicedefinition.Volume) (string, error) {
	glog.V(4).Infof("setupVolume for service Name:%s ID:%s", service.Name, service.ID)
	vol, err := a.storage.Get(tenantID)
	if err != nil {
		return "", fmt.Errorf("could not get subvolume %s: %s", tenantID, err)
	}
	a.addStorageTenant(tenantID)

	resourcePath := filepath.Join(vol.Path(), volume.ResourcePath)
	if err = os.MkdirAll(resourcePath, 0770); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("Could not create resource path: %s, %s", resourcePath, err)
	}

	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return "", fmt.Errorf("Could not get zk connection for resource path: %s, %s", resourcePath, err)
	}

	containerPath := volume.InitContainerPath
	if len(strings.TrimSpace(containerPath)) == 0 {
		containerPath = volume.ContainerPath
	}
	image, err := a.pullreg.ImagePath(service.ImageID)
	if err != nil {
		glog.Errorf("Could not get registry image for %s: %s", service.ImageID, err)
		return "", err
	}
	if err := createVolumeDir(conn, resourcePath, containerPath, image, volume.Owner, volume.Permission); err != nil {
		glog.Errorf("Error populating resource path: %s with container path: %s, %v", resourcePath, containerPath, err)
		return "", err
	}

	glog.V(4).Infof("resourcePath: %s  containerPath: %s", resourcePath, containerPath)
	return resourcePath, nil
}

// main loop of the HostAgent
func (a *HostAgent) Start(shutdown <-chan interface{}) {
	glog.Info("Starting HostAgent")

	// CC-1991: Unmount NFS on agent shutdown
	if a.storage.DriverType() == volume.DriverTypeNFS {
		defer a.releaseStorageTenants()
	}

	var wg sync.WaitGroup
	defer func() {
		glog.Info("Waiting for agent routines...")
		wg.Wait()
		glog.Info("Agent routines ended")
	}()

	wg.Add(1)
	go func() {
		glog.Infof("Starting TTL for old Docker containers")
		docker.RunTTL(shutdown, time.Minute, a.maxContainerAge)
		glog.Info("Docker TTL done")
		wg.Done()
	}()

	// Increase the number of maximal tracked connections for iptables
	maxConnections := "655360"
	if cnxns := strings.TrimSpace(os.Getenv("SERVICED_IPTABLES_MAX_CONNECTIONS")); cnxns != "" {
		maxConnections = cnxns
	}
	glog.Infof("Set sysctl maximum tracked connections for iptables to %s", maxConnections)
	utils.SetSysctl("net.netfilter.nf_conntrack_max", maxConnections)

	// Clean up any extant iptables chain, just in case
	a.servicedChain.Remove()
	// Add our chain for assigned IP rules
	if err := a.servicedChain.Inject(); err != nil {
		glog.Errorf("Error creating SERVICED iptables chain (%v)", err)
	}
	// Clean up when we're done
	defer a.servicedChain.Remove()

	for {
		// handle shutdown if we are waiting for a zk connection
		var conn coordclient.Connection
		select {
		case conn = <-zzk.Connect(zzk.GeneratePoolPath(a.poolID), zzk.GetLocalConnection):
		case <-shutdown:
			return
		}
		if conn == nil {
			continue
		}

		glog.Info("Got a connected client")

		// watch virtual IP zookeeper nodes
		virtualIPListener := virtualips.NewVirtualIPListener(a, a.hostID)

		// watch docker action nodes
		actionListener := zkdocker.NewActionListener(a, a.hostID)

		// watch the host state nodes
		// this blocks until
		// 1) has a connection
		// 2) its node is registered
		// 3) receives signal to shutdown or breaks
		hsListener := zkservice.NewHostStateListener(a, a.hostID)

		glog.Infof("Host Agent successfully started")
		zzk.Start(shutdown, conn, hsListener, virtualIPListener, actionListener)

		select {
		case <-shutdown:
			glog.Infof("Host Agent shutting down")
			return
		default:
			glog.Infof("Host Agent restarting")
		}
	}
}

// AttachAndRun implements zkdocker.ActionHandler; it attaches to a running
// container and performs a command as specified by the container's service
// definition
func (a *HostAgent) AttachAndRun(dockerID string, command []string) ([]byte, error) {
	return utils.AttachAndRun(dockerID, command)
}

// BindVirtualIP implements virtualip.VirtualIPHandler
func (a *HostAgent) BindVirtualIP(virtualIP *pool.VirtualIP, name string) error {
	glog.Infof("Adding: %v", virtualIP)
	// ensure that the Bind Address is reported by ifconfig ... ?
	if err := exec.Command("ifconfig", virtualIP.BindInterface).Run(); err != nil {
		return fmt.Errorf("Problem with BindInterface %s", virtualIP.BindInterface)
	}

	binaryNetmask := net.IPMask(net.ParseIP(virtualIP.Netmask).To4())
	cidr, _ := binaryNetmask.Size()

	// ADD THE VIRTUAL INTERFACE
	// sudo ifconfig eth0:1 inet 192.168.1.136 netmask 255.255.255.0
	// ip addr add IPADDRESS/CIDR dev eth1 label BINDINTERFACE:zvip#
	if err := exec.Command("ip", "addr", "add", virtualIP.IP+"/"+strconv.Itoa(cidr), "dev", virtualIP.BindInterface, "label", name).Run(); err != nil {
		return fmt.Errorf("Problem with creating virtual interface %s", name)
	}

	glog.Infof("Added virtual interface/IP: %v (%+v)", name, virtualIP)
	return nil
}

func (a *HostAgent) UnbindVirtualIP(virtualIP *pool.VirtualIP) error {
	glog.Infof("Removing: %v", virtualIP.IP)

	binaryNetmask := net.IPMask(net.ParseIP(virtualIP.Netmask).To4())
	cidr, _ := binaryNetmask.Size()

	//sudo ip addr del 192.168.0.10/24 dev eth0
	if err := exec.Command("ip", "addr", "del", virtualIP.IP+"/"+strconv.Itoa(cidr), "dev", virtualIP.BindInterface).Run(); err != nil {
		return fmt.Errorf("Problem with removing virtual interface %+v: %v", virtualIP, err)
	}

	glog.Infof("Removed virtual interface: %+v", virtualIP)
	return nil
}

func (a *HostAgent) VirtualInterfaceMap(prefix string) (map[string]*pool.VirtualIP, error) {
	interfaceMap := make(map[string]*pool.VirtualIP)

	//ip addr show | awk '/zvip/{print $NF}'
	virtualInterfaceNames, err := exec.Command("bash", "-c", "ip addr show | awk '/"+prefix+"/{print $NF}'").CombinedOutput()
	if err != nil {
		glog.Warningf("Determining virtual interfaces failed: %v", err)
		return interfaceMap, err
	}
	glog.V(2).Infof("Control center virtual interfaces: %v", string(virtualInterfaceNames))

	for _, virtualInterfaceName := range strings.Fields(string(virtualInterfaceNames)) {
		bindInterfaceAndIndex := strings.Split(virtualInterfaceName, prefix)
		if len(bindInterfaceAndIndex) != 2 {
			err := fmt.Errorf("Unexpected interface format: %v", bindInterfaceAndIndex)
			return interfaceMap, err
		}
		bindInterface := strings.TrimSpace(string(bindInterfaceAndIndex[0]))

		//ip addr show | awk '/virtualInterfaceName/ {print $2}'
		virtualIPAddressAndCIDR, err := exec.Command("bash", "-c", "ip addr show | awk '/"+virtualInterfaceName+"/ {print $2}'").CombinedOutput()
		if err != nil {
			glog.Warningf("Determining IP address of interface %v failed: %v", virtualInterfaceName, err)
			return interfaceMap, err
		}

		virtualIPAddress, network, err := net.ParseCIDR(strings.TrimSpace(string(virtualIPAddressAndCIDR)))
		if err != nil {
			return interfaceMap, err
		}
		netmask := net.IP(network.Mask)

		interfaceMap[virtualIPAddress.String()] = &pool.VirtualIP{PoolID: "", IP: virtualIPAddress.String(), Netmask: netmask.String(), BindInterface: bindInterface}
	}

	return interfaceMap, nil
}

// addStorageTenant remembers a storage tenant we have used
func (a *HostAgent) addStorageTenant(tenantID string) {
	for _, tid := range a.storageTenants {
		if tid == tenantID {
			return
		}
	}
	a.storageTenants = append(a.storageTenants, tenantID)
}

// releaseStorageTenants releases the resources for each tenant we have used
func (a *HostAgent) releaseStorageTenants() {
	for _, tenantID := range a.storageTenants {
		if err := a.storage.Release(tenantID); err != nil {
			glog.Warningf("Could not release tenant %s: %s", tenantID, err)
		}
	}
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
