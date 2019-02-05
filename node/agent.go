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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/iptables"
	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/zzk"
	zkdocker "github.com/control-center/serviced/zzk/docker"
	zkservice "github.com/control-center/serviced/zzk/service"
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
	ipaddress            string
	poolID               string
	master               string               // the connection string to the master agent
	uiport               string               // the port to the ui (legacy was port 8787, now default 443)
	rpcport              string               // the rpc port to serviced (default is 4979)
	rpcDisableTLS        bool                 // true of TLS should be disabled for RPC
	hostID               string               // the hostID of the current host
	dockerDNS            []string             // docker dns addresses
	storage              volume.Driver        // driver supporting the application data
	storageTenants       []string             // tenants we have mounted
	mount                []string             // each element is in the form: dockerImage,hostPath,containerPath
	currentServices      map[string]*exec.Cmd // the current running services
	mux                  *proxy.TCPMux
	muxport              string // the mux port to serviced (default is 22250)
	useTLS               bool   // true if TLS should be enabled for MUX
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
	zkSessionTimeout     int
	delegateKeyFile      string
	tokenFile            string
	conntrackFlush       bool
	serviceCache         *ServiceCache
	vip                  VIP
}

func getZkDSN(zookeepers []string,
	sessionTimeout int,
	connectTimeout int,
	perHostConnectDelay int,
	reconnectStartDelay int,
	reconnectMaxDelay int) string {
	if len(zookeepers) == 0 {
		zookeepers = []string{"127.0.0.1:2181"}
	}
	dsn := coordzk.DSN{
		Servers: zookeepers,
		SessionTimeout: time.Duration(sessionTimeout) * time.Second,
		ConnectTimeout: time.Duration(connectTimeout) * time.Second,
		PerHostConnectDelay: time.Duration(perHostConnectDelay) * time.Second,
		ReconnectStartDelay: time.Duration(reconnectStartDelay) * time.Second,
		ReconnectMaxDelay: time.Duration(reconnectMaxDelay) * time.Second,
	}
	return dsn.String()
}

type AgentOptions struct {
	IPAddress            string
	PoolID               string
	Master               string
	UIPort               string
	RPCPort              string
	RPCDisableTLS        bool // true if TLS should be disabled for RPC
	DockerDNS            []string
	VolumesPath          string
	Mount                []string
	FSType               volume.DriverType
	Zookeepers           []string
	Mux                  *proxy.TCPMux
	MuxPort              string
	UseTLS               bool
	DockerRegistry       string
	MaxContainerAge      time.Duration // Maximum container age for a stopped container before being removed
	VirtualAddressSubnet string
	ControllerBinary     string
	LogstashURL          string
	DockerLogDriver      string
	DockerLogConfig      map[string]string
	ZKSessionTimeout     int
	ZKConnectTimeout     int
	ZKPerHostConnectDelay int
	ZKReconnectStartDelay int
	ZKReconnectMaxDelay  int
	DelegateKeyFile      string
	TokenFile            string
	ConntrackFlush       bool
}

// NewHostAgent creates a new HostAgent given a connection string
func NewHostAgent(options AgentOptions, reg registry.Registry) (*HostAgent, error) {
	// save off the arguments
	agent := &HostAgent{}
	agent.ipaddress = options.IPAddress
	agent.poolID = options.PoolID
	agent.master = options.Master
	agent.uiport = options.UIPort
	agent.rpcport = options.RPCPort
	agent.rpcDisableTLS = options.RPCDisableTLS
	agent.dockerDNS = options.DockerDNS
	agent.mount = options.Mount
	agent.mux = options.Mux
	agent.muxport = options.MuxPort
	agent.useTLS = options.UseTLS
	agent.maxContainerAge = options.MaxContainerAge
	agent.virtualAddressSubnet = options.VirtualAddressSubnet
	agent.servicedChain = iptables.NewChain("SERVICED")
	agent.controllerBinary = options.ControllerBinary
	agent.logstashURL = options.LogstashURL
	agent.dockerLogDriver = options.DockerLogDriver
	agent.dockerLogConfig = options.DockerLogConfig
	agent.zkSessionTimeout = options.ZKSessionTimeout
	agent.delegateKeyFile = options.DelegateKeyFile
	agent.tokenFile = options.TokenFile
	agent.conntrackFlush = options.ConntrackFlush
	agent.serviceCache = NewServiceCache(options.Master)

	var err error
	dsn := getZkDSN(options.Zookeepers,
		agent.zkSessionTimeout,
		options.ZKConnectTimeout,
		options.ZKPerHostConnectDelay,
		options.ZKReconnectStartDelay,
		options.ZKReconnectMaxDelay)
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
	agent.vip = NewVirtualIPManager("cc")
	return agent, err
}

// SetVIP allows you to update the default vip (primarily for testing)
func (a *HostAgent) SetVIP(v VIP) {
	a.vip = v
}

func attachAndRun(dockerID, command string) ([]byte, error) {
	if dockerID == "" {
		return nil, errors.New("missing docker ID")
	} else if command == "" {
		return nil, nil
	}

	output, err := utils.AttachAndRun(dockerID, []string{command})
	if err != nil {
		err = fmt.Errorf("%s (%s)", string(output), err)
		glog.Errorf("Could not pause container %s: %s", dockerID, err)
		return nil, err
	}
	return output, nil
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
	if !IsValidOwnerSpec(owner) {
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
	if err := CreateVolumeDir(conn, resourcePath, containerPath, image, volume.Owner, volume.Permission); err != nil {
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

	// run the virtual ip listener
	wg.Add(1)
	go func() {
		glog.Infof("Starting virtual ip listener")

		var addrs []string
		for _, ip := range a.vip.GetAll() {
			addrs = append(addrs, ip.Addr)
		}

		l := zkservice.NewHostIPListener(a.hostID, a, addrs)
		zzk.Manage(shutdown, zzk.GeneratePoolPath(a.poolID), l)
		glog.Infof("Virtual ip listener done")
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

	unregister := make(chan interface{})
	stop := make(chan interface{})

	// Create the host state listener here, so it keeps its state
	hsListener := zkservice.NewHostStateListener(a, a.hostID, shutdown)

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

		rwg := &sync.WaitGroup{}
		rwg.Add(1)
		go func() {
			defer rwg.Done()
			t := time.NewTimer(time.Second)
			defer t.Stop()
			for {
				err := zkservice.RegisterHost(unregister, conn, a.hostID)
				if err != nil {
					t.Stop()
					t = time.NewTimer(time.Second)
					select {
					case <-t.C:
					case <-unregister:
						return
					}
				} else {
					return
				}
			}
		}()

		// watch docker action nodes
		actionListener := zkdocker.NewActionListener(a, a.hostID)

		// watch the host state nodes
		// this blocks until
		// 1) has a connection
		// 2) its node is registered
		// 3) receives signal to shutdown or breaks
		startExit := make(chan struct{})
		go func() {
			defer close(startExit)
			glog.Infof("Host Agent successfully started")
			zzk.Start(stop, conn, hsListener, actionListener)
		}()

		select {
		case <-startExit:
			glog.Infof("Host Agent restarting")
			close(unregister)
			rwg.Wait()

			// The unregister var is used directly in the goroutine that calls
			// RegisterHost above. Always wait for rwg before re-assigning it.
			unregister = make(chan interface{})

		case <-shutdown:
			glog.Infof("Host Agent shutting down")

			lockpth := path.Join("/hosts", a.hostID, "locked")
			err := conn.CreateIfExists(lockpth, &coordclient.Dir{})
			if err == nil || err == coordclient.ErrNodeExists {
				mu, _ := conn.NewLock(lockpth)
				if mu != nil {
					mu.Lock()
					defer mu.Unlock()
				}
			}

			close(stop)
			<-startExit
			close(unregister)
			rwg.Wait()
			conn.Delete(path.Join("/hosts", a.hostID, "online"))
			return
		}
	}

}

// AttachAndRun implements zkdocker.ActionHandler; it attaches to a running
// container and performs a command as specified by the container's service
// definition
func (a *HostAgent) AttachAndRun(dockerID string, command []string) ([]byte, error) {
	return utils.AttachAndRun(dockerID, command)
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
