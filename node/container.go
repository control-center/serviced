// Copyright 2016 The Serviced Authors.
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

package node

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/iptables"
	"github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	dockerclient "github.com/fsouza/go-dockerclient"
)

// StopContainer stops running container or returns nil if the container does
// not exist or has already stopped.
func (a *HostAgent) StopContainer(serviceID string, instanceID int) error {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	// find the container by name
	ctrName := fmt.Sprintf("%s-%d", serviceID, instanceID)
	ctr, err := docker.FindContainer(ctrName)
	if err == docker.ErrNoSuchContainer {
		logger.Debug("Could not stop, container not found")
		return nil
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up container")
		return err
	}

	err = ctr.Stop(45 * time.Second)
	if _, ok := err.(*dockerclient.ContainerNotRunning); ok {
		logger.Debug("Container already stopped")
		return nil
	} else if err != nil {
		logger.WithError(err).Debug("Could not stop container")
		return err
	}

	return nil
}

// AttachContainer returns a channel that monitors the run state of a given
// container.
func (a *HostAgent) AttachContainer(state *zkservice.ServiceState, serviceID string, instanceID int) (<-chan time.Time, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":   serviceID,
		"instanceid":  instanceID,
		"containerid": state.ContainerID,
	})

	// find the container by name
	ctrName := fmt.Sprintf("%s-%d", serviceID, instanceID)
	ctr, err := docker.FindContainer(ctrName)
	if err == docker.ErrNoSuchContainer {
		return nil, nil
	} else if err != nil {
		logger.WithError(err).Debug("Could not look up container")
		return nil, err
	}

	// verify that the container ids match, otherwise delete
	// the container.
	if ctr.ID != state.ContainerID {
		ctr.Kill()
		if err := ctr.Delete(true); err != nil {
			logger.WithError(err).Debug("Could not delete orphaned container")
			return nil, err
		}
		logger.WithField("currentcontainerid", ctr.ID).Warn("Removed orphaned container")
		return nil, nil
	}

	// monitor the container
	ev := a.monitorContainer(logger, ctr)

	// make sure the container is running at the time this event is set
	if !ctr.IsRunning() {
		logger.Debug("Could not capture event, container not running")
		ctr.CancelOnEvent(docker.Die)
		return nil, nil
	}
	go a.exposeAssignedIPs(state, ctr)
	return ev, nil
}

// StartContainer creates a new container and starts.  It returns info about
// the container, and an event monitor to track the running state of the
// service.
func (a *HostAgent) StartContainer(cancel <-chan interface{}, serviceID string, instanceID int) (*zkservice.ServiceState, <-chan time.Time, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	evaluatedService, tenantID, err := a.serviceCache.GetEvaluatedService(serviceID, instanceID)
	if err != nil {
		logger.WithError(err).Error("Failed to get service")
		return nil, nil, err
	}

	// pull the service image
	imageUUID, imageName, err := a.pullImage(logger, cancel, evaluatedService.ImageID)
	if err != nil {
		logger.WithError(err).Debug("Could not pull the service image")
		return nil, nil, err
	}
	// Update the service with the complete image name
	evaluatedService.ImageID = imageName

       // Establish a connection to the master
       masterClient, err := master.NewClient(a.master)
       if err != nil {
               logger.WithField("master", a.master).WithError(err).Debug("Could not connect to the master")
               return nil, nil, err
       }
       defer masterClient.Close()

	// get the system user
	systemUser, err := masterClient.GetSystemUser()
	if err != nil {
		logger.WithError(err).Error("Failed to get the system user account")
		return nil, nil, err
	}
	logger.WithField("systemuser", systemUser.Name).Debug("Got system user")

	// get the container configs
	ctr, state, err := a.setupContainer(tenantID, evaluatedService, instanceID, systemUser, imageUUID)
	if err != nil {
		logger.WithError(err).Debug("Could not setup container")
		return nil, nil, err
	}

	// start the container
	ev := a.monitorContainer(logger, ctr)

	if err := ctr.Start(); err != nil {
		logger.WithError(err).Debug("Could not start container")
		ctr.CancelOnEvent(docker.Die)
		return nil, nil, err
	}
	logger.Debug("Started container")

	dctr, err := ctr.Inspect()
	if err != nil {
		logger.WithError(err).Debug("Could not inspect container")
		ctr.CancelOnEvent(docker.Die)
		return nil, nil, err
	}

	state.HostIP = a.ipaddress
	state.PrivateIP = ctr.NetworkSettings.IPAddress
	state.Started = dctr.State.StartedAt

	go a.exposeAssignedIPs(state, ctr)
	return state, ev, nil
}

// ResumeContainer resumes a paused container
func (a *HostAgent) ResumeContainer(serviceID string, instanceID int) error {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	svc, err := a.getService(serviceID)
	if err != nil {
		logger.WithError(err).Debug("Unable to retrieve service")
		return nil
	}

	ctrName := fmt.Sprintf("%s-%d", svc.ID, instanceID)

	// check to see if the container exists and is running
	ctr, err := docker.FindContainer(ctrName)
	if err == docker.ErrNoSuchContainer {
		// container has been deleted and the event monitor should catch this
		logger.Debug("Container not found")
		return nil
	}
	if !ctr.IsRunning() {
		// container has stopped and the event monitor should catch this
		logger.Debug("Container stopped")
		return nil
	}

	// resume the paused container
	if err := attachAndRun(ctrName, svc.Snapshot.Resume); err != nil {
		logger.WithError(err).Debug("Could not resume paused container")
		return err
	}
	logger.Debug("Resumed paused container")

	return nil
}

// PauseContainer pauses a running container
func (a *HostAgent) PauseContainer(serviceID string, instanceID int) error {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})
	ctrName := fmt.Sprintf("%s-%d", serviceID, instanceID)

	// check to see if the container exists and is running
	ctr, err := docker.FindContainer(ctrName)
	if err == docker.ErrNoSuchContainer {
		// container has been deleted and the event monitor should catch this
		logger.Debug("Container not found")
		return nil
	}
	if !ctr.IsRunning() {
		// container has stopped and the event monitor should catch this
		logger.Debug("Container stopped")
		return nil
	}

	// Get the service from the client
	svc, err := a.getService(serviceID)
	if err != nil {
		logger.WithError(err).Debug("Unable to get service")
		return nil
	}

	// pause the running container
	if err := attachAndRun(ctrName, svc.Snapshot.Pause); err != nil {
		logger.WithError(err).Debug("Could not pause running container")
		return err
	}
	logger.Debug("Paused running container")
	return nil
}

// pullImage pulls the service image and returns the uuid string
// of the image and the fully qualified image name.
func (a *HostAgent) pullImage(logger *log.Entry, cancel <-chan interface{}, imageID string) (string, string, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		logger.WithError(err).Debug("Could not connect to coordinator")

		// TODO: wrap error?
		return "", "", err
	}

	timeoutC := make(chan time.Time)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-done:
		case <-cancel:
			select {
			case <-done:
			case timeoutC <- time.Now():
			}
		}
	}()

	a.pullreg.SetConnection(conn)
	if err := a.pullreg.PullImage(timeoutC, imageID); err != nil {
		logger.WithError(err).Debug("Could not pull image")

		// TODO: wrap error?
		return "", "", err
	}
	logger.Debug("Pulled image")

	uuid, err := registry.GetImageUUID(conn, imageID)
	if err != nil {
		logger.WithError(err).Debug("Could not load image id")

		// TODO: wrap error?
		return "", "", err
	}
	logger.Debug("Found image uuid")

	name, err := a.pullreg.ImagePath(imageID)
	if err != nil {
		logger.WithError(err).Debug("Could not get full image name")

		// TODO: wrap error?
		return "", "", err
	}

	return uuid, name, nil
}

// monitorContainer tracks the running state of the container.
func (a *HostAgent) monitorContainer(logger *log.Entry, ctr *docker.Container) <-chan time.Time {
	ev := make(chan time.Time, 1)
	ctr.OnEvent(docker.Die, func(_ string) {
		defer close(ev)
		dctr, err := ctr.Inspect()
		if err != nil {
			logger.WithError(err).Error("Could not look up container")
			ev <- time.Now()
			return
		}

		logger.WithFields(log.Fields{
			"terminated": dctr.State.FinishedAt,
			"exitcode":   dctr.State.ExitCode,
		}).Debug("Container exited")

		if dctr.State.ExitCode != 0 || log.GetLevel() == log.DebugLevel {
			dockerLogsToFile(ctr.ID, 1000)
		}

		if err := ctr.Delete(true); err != nil {
			logger.WithError(err).Warn("Could not delete container")
		}

		// just in case something unusual happened
		if !dctr.State.FinishedAt.IsZero() {
			ev <- dctr.State.FinishedAt
		} else {
			ev <- time.Now()
		}
		return
	})
	return ev
}

// exposeAssignedIPs sets up iptables forwarding rules for endpoints with
// assigned ips.
func (a *HostAgent) exposeAssignedIPs(state *zkservice.ServiceState, ctr *docker.Container) {
	logger := plog.WithFields(log.Fields{
		"containerid":   state.ContainerID,
		"containername": ctr.Name,
	})

	if ip := state.AssignedIP; ip != "" {
		for _, exp := range state.Exports {
			if port := exp.AssignedPortNumber; port > 0 {
				explog := logger.WithFields(log.Fields{
					"application": exp.Application,
					"ipaddress":   ip,
					"portnumber":  port,
				})
				explog.Debug("Starting proxy for endpoint")
				public := iptables.NewAddress(ip, int(port))
				private := iptables.NewAddress(state.PrivateIP, int(exp.PortNumber))

				a.servicedChain.Forward(iptables.Add, exp.Protocol, public, private)
				defer a.servicedChain.Forward(iptables.Delete, exp.Protocol, public, private)
			}
		}

		ctr.Wait(time.Hour * 24 * 365)
	}
}

func (a *HostAgent) getService(serviceID string) (*service.Service, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
	})

	// Establish a connection to the master
	masterClient, err := master.NewClient(a.master)
	if err != nil {
		logger.WithField("master", a.master).WithError(err).Debug("Could not connect to the master")
		return nil, err
	}
	defer masterClient.Close()

	// Get the service from the master
	svc, err := masterClient.GetService(serviceID)
	if err != nil {
		logger.WithError(err).Debug("Unable to get the service")
		return nil, err
	}

	return svc, nil
}

// dockerLogsToFile dumps container logs to file
func dockerLogsToFile(containerid string, numlines int) {
	// TODO: need to get logs from api

	fname := filepath.Join(os.TempDir(), fmt.Sprintf("%s.container.log", containerid))
	f, e := os.Create(fname)
	if e != nil {
		plog.WithError(e).WithFields(log.Fields{
			"file": fname,
		}).Debug("Unable to create container log file")
		return
	}
	defer f.Close()
	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", numlines), containerid)
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Run(); err != nil {
		plog.WithError(err).Debug("Unable to get logs for container")
		return
	}
	plog.WithFields(log.Fields{
		"file":  fname,
		"lines": numlines,
	}).Infof("Container log dumped to file")
}

// setupContainer creates and populates two structures, a docker client Config and a docker client HostConfig structure
// that are used to create and start a container respectively. The information used to populate the structures is pulled from
// the service, serviceState, and conn values that are passed into setupContainer.
func (a *HostAgent) setupContainer(tenantID string, svc *service.Service, instanceID int, systemUser user.User, imageUUID string) (*docker.Container, *zkservice.ServiceState, error) {
	logger := plog.WithFields(log.Fields{
		"tenantid":    tenantID,
		"servicename": svc.Name,
		"serviceid":   svc.ID,
		"instanceid":  instanceID,
	})
	cfg, hcfg, state, err := a.createContainerConfig(tenantID, svc, instanceID, systemUser, imageUUID)
	if err != nil {
		logger.WithError(err).Error("Unable to create container configuration")
		return nil, nil, err
	}

	ctr, err := a.createContainer(cfg, hcfg, svc.ID, instanceID)
	if err != nil {
		logger.WithFields(log.Fields{
			"image":      cfg.Image,
			"instanceid": instanceID,
		}).WithError(err).Error("Could not create container")
		return nil, nil, err
	}
	state.ContainerID = ctr.ID

	return ctr, state, nil
}

func (a *HostAgent) createContainerConfig(tenantID string, svc *service.Service, instanceID int, systemUser user.User, imageUUID string) (*dockerclient.Config, *dockerclient.HostConfig, *zkservice.ServiceState, error) {
	logger := plog.WithFields(log.Fields{
		"tenantid":    tenantID,
		"servicename": svc.Name,
		"serviceid":   svc.ID,
		"instanceid":  instanceID,
	})
	cfg := &dockerclient.Config{}
	hcfg := &dockerclient.HostConfig{}

	cfg.User = "root"
	cfg.WorkingDir = "/"
	cfg.Image = svc.ImageID

	// get the endpoints
	cfg.ExposedPorts = make(map[dockerclient.Port]struct{})
	hcfg.PortBindings = make(map[dockerclient.Port][]dockerclient.PortBinding)
	state := &zkservice.ServiceState{
		ImageUUID: imageUUID,
		Paused:    false,
		HostIP:    a.ipaddress,
	}

	var assignedIP string
	var static bool
	if svc.Endpoints != nil {
		logger.WithField("endpoints", svc.Endpoints).Debug("Endpoints for service")
		for _, endpoint := range svc.Endpoints {
			if endpoint.Purpose == "export" { // only expose remote endpoints
				var p string
				switch endpoint.Protocol {
				case commons.UDP:
					p = fmt.Sprintf("%d/%s", endpoint.PortNumber, "udp")
				default:
					p = fmt.Sprintf("%d/%s", endpoint.PortNumber, "tcp")
				}
				cfg.ExposedPorts[dockerclient.Port(p)] = struct{}{}
				hcfg.PortBindings[dockerclient.Port(p)] = append(hcfg.PortBindings[dockerclient.Port(p)], dockerclient.PortBinding{})

				var assignedPortNumber uint16
				if a := endpoint.GetAssignment(); a != nil {
					assignedIP = endpoint.AddressAssignment.IPAddr
					static = endpoint.AddressAssignment.AssignmentType == commons.STATIC
					assignedPortNumber = a.Port
				}

				// set the export data
				state.Exports = append(state.Exports, zkservice.ExportBinding{
					Application:        endpoint.Application,
					Protocol:           endpoint.Protocol,
					PortNumber:         endpoint.PortNumber,
					AssignedPortNumber: assignedPortNumber,
				})
			} else {
				state.Imports = append(state.Imports, zkservice.ImportBinding{
					Application:    endpoint.Application,
					Purpose:        endpoint.Purpose,
					PortNumber:     endpoint.PortNumber,
					PortTemplate:   endpoint.PortTemplate,
					VirtualAddress: endpoint.VirtualAddress,
				})
			}
		}
		state.AssignedIP = assignedIP
		state.Static = static
	}

	bindsMap := make(map[string]string) // map to prevent duplicate path assignments. Use to populate hcfg.Binds later.

	// iterate svc.Volumes - create bindings for non-dfs volumes
	for _, volume := range svc.Volumes {
		if volume.Type != "" && volume.Type != "dfs" {
			continue
		}

		resourcePath, err := a.setupVolume(tenantID, svc, volume)
		if err != nil {
			return nil, nil, nil, err
		}

		addBindingToMap(bindsMap, volume.ContainerPath, resourcePath)
	}

	// mount serviced path
	dir, _, err := ExecPath()
	if err != nil {
		logger.WithError(err).Error("Error getting the path to the serviced executable")
		return nil, nil, nil, err
	}

	dir, binary := filepath.Split(a.controllerBinary)
	addBindingToMap(bindsMap, "/serviced", dir)

	// bind mount everything we need for filebeat
	if len(svc.LogConfigs) != 0 {
		const LOGSTASH_CONTAINER_DIRECTORY = "/usr/local/serviced/resources/logstash"
		logstashPath := utils.ResourcesDir() + "/logstash"
		addBindingToMap(bindsMap, LOGSTASH_CONTAINER_DIRECTORY, logstashPath)
	}

	// Bind mount the keys we need
	addBindingToMap(bindsMap, "/etc/serviced", filepath.Dir(a.delegateKeyFile))

	// specify temporary volume paths for docker to create
	tmpVolumes := []string{"/tmp"}
	for _, volume := range svc.Volumes {
		if volume.Type == "tmp" {
			tmpVolumes = append(tmpVolumes, volume.ContainerPath)
		}
	}

	// add arguments to mount requested directory (if requested)
	logger.Debug("Checking service's mount options")
	for _, bindMountString := range a.mount {
		logger.WithField("bindmount", bindMountString).Debug("Checking bindmount string")
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
			logger.WithFields(log.Fields{
				"requestedimage": requestedImage,
				"hostpath":       hostPath,
				"containerpath":  containerPath,
			}).Debug("Parsed out bind mount information")

			// insert tenantId into requestedImage - see facade.DeployService
			matchedRequestedImage := false
			if requestedImage == "*" {
				matchedRequestedImage = true
			} else {
				imageID, err := commons.ParseImageID(requestedImage)
				if err != nil {
					logger.WithError(err).
						WithField("requestedimageid", requestedImage).
						Error("Unable to parse requested ImageID")
					continue
				}
				svcImageID, err := commons.ParseImageID(svc.ImageID)
				if err != nil {
					logger.WithError(err).Error("Unable to parse service imageID")
					continue
				}
				matchedRequestedImage = (imageID.Repo == svcImageID.Repo)
			}

			logger.WithFields(log.Fields{
				"matchedrequestedimage": matchedRequestedImage,
			}).Debug("Finished evaluation for matchedRequestedImage")

			if matchedRequestedImage {
				addBindingToMap(bindsMap, containerPath, hostPath)
			}
		} else {
			logger.WithField("bindmount", bindMountString).
				Warn("Could not bind mount the requested mount point")
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
		logger.WithError(err).Error("Unable to get host IP addresses")
		return nil, nil, nil, err
	}

	// XXX: Hopefully temp fix for CC-1384 & CC-1631 (docker/docker issue 14203).
	count := rand.Intn(128)
	fix := ""
	for i := 0; i < count; i++ {
		fix += "."
	}
	// End temp fix part 1. See immediately below for part 2.

	// add arguments for environment variables
	cfg.Env = append(svc.Environment,
		fmt.Sprintf("CONTROLPLANE_SYSTEM_USER=%s", systemUser.Name),
		fmt.Sprintf("CONTROLPLANE_SYSTEM_PASSWORD=%s", systemUser.Password),
		fmt.Sprintf("CONTROLPLANE_HOST_IPS='%s'", strings.Join(ips, " ")),
		fmt.Sprintf("SERVICED_VIRTUAL_ADDRESS_SUBNET=%s", a.virtualAddressSubnet),
		fmt.Sprintf("SERVICED_IS_SERVICE_SHELL=false"),
		fmt.Sprintf("SERVICED_NOREGISTRY=%s", os.Getenv("SERVICED_NOREGISTRY")),
		fmt.Sprintf("SERVICED_SERVICE_IMAGE=%s", svc.ImageID),
		fmt.Sprintf("SERVICED_MAX_RPC_CLIENTS=1"),
		fmt.Sprintf("SERVICED_RPC_PORT=%s", a.rpcport),
		fmt.Sprintf("SERVICED_LOG_ADDRESS=%s", a.logstashURL),
		//The SERVICED_UI_PORT environment variable is deprecated and services should always use port 443 to contact serviced from inside a container
		"SERVICED_UI_PORT=443",
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

	cmd := []string{filepath.Join("/serviced", binary)}

	// Flag TLS for the mux if it's disabled
	if !a.useTLS {
		cmd = append(cmd, "--mux-disable-tls")
	}

	cfg.Cmd = append(cmd,
		svc.ID,
		strconv.Itoa(instanceID),
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
	return cfg, hcfg, state, nil
}

func (a *HostAgent) createContainer(conf *dockerclient.Config, hostConf *dockerclient.HostConfig, svcID string, instanceID int) (*docker.Container, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":  svcID,
		"instanceid": instanceID,
	})

	// create the container
	opts := dockerclient.CreateContainerOptions{
		Name:       fmt.Sprintf("%s-%d", svcID, instanceID),
		Config:     conf,
		HostConfig: hostConf,
	}

	ctr, err := docker.NewContainer(&opts, false, 10*time.Second, nil, nil)
	if err != nil {
		logger.WithError(err).Error("Could not create container")
		return nil, err
	}
	logger.WithField("containerid", ctr.ID).Debug("Created a new container")
	return ctr, nil
}

func addBindingToMap(bindsMap map[string]string, cp, rp string) {
	rp = strings.TrimSpace(rp)
	cp = strings.TrimSpace(cp)
	if len(rp) > 0 && len(cp) > 0 {
		log.WithFields(log.Fields{"ContainerPath": cp, "ResourcePath": rp}).Debug("Adding path to bindsMap")
		bindsMap[cp] = rp
	} else {
		log.WithFields(log.Fields{"ContainerPath": cp, "ResourcePath": rp}).Warn("Not adding to map, because at least one argument is empty.")
	}
}
