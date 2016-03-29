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

package api

import (
	"bytes"

	commonsdocker "github.com/control-center/serviced/commons/docker"
	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/coordinator/storage"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dao/elasticsearch"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/dfs/nfs"
	"github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/scheduler"
	"github.com/control-center/serviced/shell"
	"github.com/control-center/serviced/stats"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"

	"github.com/control-center/serviced/web"
	"github.com/control-center/serviced/zzk"

	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	// Needed for profiling
	"net/http/httputil"
	_ "net/http/pprof"
)

var minDockerVersion = version{1, 9, 0}
var dockerRegistry = "localhost:5000"

const (
	localhost = "127.0.0.1"
)

type daemon struct {
	servicedEndpoint string
	staticIPs        []string
	cpDao            dao.ControlPlane
	dsDriver         datastore.Driver
	dsContext        datastore.Context
	hostID           string
	zClient          *coordclient.Client
	storageHandler   *storage.Server
	masterPoolID     string
	hostAgent        *node.HostAgent
	shutdown         chan interface{}
	waitGroup        *sync.WaitGroup
	rpcServer        *rpc.Server

	facade *facade.Facade
	hcache *health.HealthStatusCache
	docker docker.Docker
	reg    *registry.RegistryListener
	disk   volume.Driver
	net    storage.StorageDriver
}

func init() {
	commonsdocker.StartKernel()
}

func newDaemon(servicedEndpoint string, staticIPs []string, masterPoolID string) (*daemon, error) {
	d := &daemon{
		servicedEndpoint: servicedEndpoint,
		staticIPs:        staticIPs,
		masterPoolID:     masterPoolID,
		shutdown:         make(chan interface{}),
		waitGroup:        &sync.WaitGroup{},
		rpcServer:        rpc.NewServer(),
	}
	return d, nil
}

func (d *daemon) getEsClusterName(name string) string {
	var (
		clusterName string
		err         error
	)
	filename := path.Join(options.IsvcsPath, name+".clustername")
	data, _ := ioutil.ReadFile(filename)
	clusterName = string(bytes.TrimSpace(data))
	if clusterName == "" {
		clusterName, err = utils.NewUUID36()
		if err != nil {
			glog.Fatalf("Could not generate uuid: %s", err)
		}
		if err = os.MkdirAll(filepath.Dir(filename), 0770); err != nil && !os.IsExist(err) {
			glog.Fatalf("Could not create path to file %s: %s", filename, err)
		}
		if err = ioutil.WriteFile(filename, []byte(clusterName), 0600); err != nil {
			glog.Fatalf("Could not write clustername to file %s: %s", filename, err)
		}
	}
	return clusterName
}

func (d *daemon) startISVCS() {
	isvcs.Init(options.ESStartupTimeout, options.DockerLogDriver, convertStringSliceToMap(options.DockerLogConfigList), d.docker)
	isvcs.Mgr.SetVolumesDir(options.IsvcsPath)
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-serviced", "cluster", d.getEsClusterName("elasticsearch-serviced")); err != nil {
		glog.Fatalf("Could not set es-serviced option: %s", err)
	}
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-logstash", "cluster", d.getEsClusterName("elasticsearch-logstash")); err != nil {
		glog.Fatalf("Could not set es-logstash option: %s", err)
	}
	if err := isvcs.Mgr.Start(); err != nil {
		glog.Fatalf("Could not start isvcs: %s", err)
	}
	go d.startLogstashPurger(10*time.Minute, time.Duration(options.LogstashCycleTime)*time.Hour)
}

func (d *daemon) startAgentISVCS(serviceNames []string) {
	isvcs.InitServices(serviceNames, options.DockerLogDriver, convertStringSliceToMap(options.DockerLogConfigList), d.docker)
	isvcs.Mgr.SetVolumesDir(options.IsvcsPath)
	if err := isvcs.Mgr.Start(); err != nil {
		glog.Fatalf("Could not start isvcs: %s", err)
	}
}

func (d *daemon) stopISVCS() {
	glog.Infof("Shutting down isvcs")
	if err := isvcs.Mgr.Stop(); err != nil {
		glog.Errorf("Error while stopping isvcs: %s", err)
	}
	glog.Infof("isvcs shut down")
}

func (d *daemon) startRPC() {
	if options.DebugPort > 0 {
		go func() {
			if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", options.DebugPort), nil); err != nil {
				glog.Errorf("Unable to bind to debug port %s. Is another instance running?", err)
				return
			}
		}()
	}

	var listener net.Listener
	var err error
	if rpcutils.RPCDisableTLS {
		listener, err = net.Listen("tcp", options.Listen)
	} else {
		var tlsConfig *tls.Config
		tlsConfig, err = getTLSConfig()
		if err != nil {
			glog.Fatalf("Unable to get TLS config: %v", err)
		}

		listener, err = tls.Listen("tcp", options.Listen, tlsConfig)
	}
	if err != nil {
		glog.Fatalf("Unable to bind to port %s. Is another instance running?", options.Listen)
	}

	rpcutils.SetDialTimeout(options.RPCDialTimeout)
	d.rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	glog.V(0).Infof("Listening on %s", listener.Addr().String())
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				glog.Fatalf("Error accepting connections: %s", err)
			}
			go d.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()
}

func (d *daemon) startDockerRegistryProxy() {
	host, port, err := net.SplitHostPort(options.DockerRegistry)
	if err != nil {
		glog.Fatalf("Could not parse docker registry: %s", err)
	}

	if isLocalAddress := func(host string) bool {
		addrs, err := net.LookupIP(host)
		if err != nil {
			glog.Fatalf("Could not resolve ips for docker registry host %s: %s", host, err)
		}
		for _, addr := range addrs {
			if addr.IsLoopback() {
				glog.Infof("Docker registry host %s is a loopback address at %s", host, addr)
				return true
			}
		}

		iaddrs, err := net.InterfaceAddrs()
		if err != nil {
			glog.Fatalf("Could not look up interface address: %s", err)
		}
		for _, iaddr := range iaddrs {
			var ip net.IP
			switch iaddr.(type) {
			case *net.IPNet:
				ip = iaddr.(*net.IPNet).IP
			case *net.IPAddr:
				ip = iaddr.(*net.IPAddr).IP
			default:
				continue
			}

			if !ip.IsLoopback() {
				glog.Infof("Checking interface address at %s", iaddr)
				for _, addr := range addrs {
					if addr.Equal(ip) {
						glog.Infof("Host %s is a local address at %s", host, ip)
						return true
					}
				}
			}
		}

		glog.Infof("Host %s is not a local address", host)
		return false
	}(host); isLocalAddress && port == "5000" {
		return
	}

	if options.Master {
		glog.Infof("Not creating a reverse proxy for docker registry when running as a master")
		return
	}

	glog.Infof("Creating a reverse proxy for docker registry %s at %s", options.DockerRegistry, dockerRegistry)
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   options.DockerRegistry,
	})
	proxy.Director = func(r *http.Request) {
		r.Host = options.DockerRegistry
		r.URL.Host = r.Host
		r.URL.Scheme = "http"
	}
	http.Handle("/", proxy)
	go func() {
		if err := http.ListenAndServe(dockerRegistry, nil); err != nil {
			glog.Fatalf("Unable to bind to docker registry port (:5000) %s. Is another instance already running?", err)
		}
	}()
}

func (d *daemon) run() (err error) {
	if d.hostID, err = utils.HostID(); err != nil {
		glog.Fatalf("Could not get host ID: %s", err)
	} else if err := validation.ValidHostID(d.hostID); err != nil {
		glog.Errorf("invalid hostid: %s", d.hostID)
	}

	if currentDockerVersion, err := node.GetDockerVersion(); err != nil {
		glog.Fatalf("Could not get docker version: %s", err)
	} else if minDockerVersion.Compare(currentDockerVersion) < 0 {
		glog.Fatalf("serviced requires docker >= %s", minDockerVersion)
	}

	if !volume.Registered(options.FSType) {
		glog.Fatalf("no driver registered for %s", options.FSType)
	}

	// set up docker
	d.docker, err = docker.NewDockerClient()
	if err != nil {
		glog.Fatalf("Could not connect to docker client: %s", err)
	}

	// set up the registry
	d.reg = registry.NewRegistryListener(d.docker, dockerRegistry, d.hostID)

	// Initialize the storage driver
	if !filepath.IsAbs(options.VolumesPath) {
		glog.Fatalf("volumes path %s must be absolute", options.VolumesPath)
	}
	if err := volume.InitDriver(options.FSType, options.VolumesPath, options.StorageArgs); err != nil {
		glog.Fatalf("Could not initialize storage driver type=%s root=%s args=%v options=%+v: %s", options.FSType, options.VolumesPath, options.StorageArgs, options.StorageOptions, err)
	}
	d.startRPC()
	d.startDockerRegistryProxy()

	//Start the zookeeper client
	localClient, err := d.initZK(options.Zookeepers)
	if err != nil {
		glog.Errorf("failed to create a local coordclient: %v", err)
		return err
	}
	zzk.InitializeLocalClient(localClient)

	if options.Master {
		d.startISVCS()
		if err := d.startMaster(); err != nil {
			glog.Fatal(err)
		}
	} else {
		d.startAgentISVCS(options.StartISVCS)
	}

	if options.Agent {
		if err := d.startAgent(); err != nil {
			glog.Fatal(err)
		}
	}

	signalC := make(chan os.Signal, 10)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-signalC
	glog.Info("Shutting down due to interrupt")
	close(d.shutdown)

	done := make(chan struct{})
	go func() {
		defer close(done)
		glog.Info("Stopping sub-processes")
		d.waitGroup.Wait()
		glog.Info("Sub-processes have stopped")
	}()

	select {
	case <-done:
		defer glog.Info("Shutdown")
	case <-time.After(60 * time.Second):
		defer glog.Infof("Timeout waiting for shutdown")
	}

	zzk.ShutdownConnections()
	switch sig {
	case syscall.SIGHUP:
		glog.Infof("Not shutting down isvcs")
		command := os.Args
		glog.Infof("Reloading by calling syscall.exec for command: %+v\n", command)
		syscall.Exec(command[0], command[0:], os.Environ())
	default:
		d.stopISVCS()
	}
	d.hcache.SetPurgeFrequency(0)
	return nil
}

func (d *daemon) initContext() (datastore.Context, error) {
	datastore.Register(d.dsDriver)
	ctx := datastore.Get()
	if ctx == nil {
		return nil, errors.New("context not available")
	}
	return ctx, nil
}

func (d *daemon) initZK(zks []string) (*coordclient.Client, error) {
	coordzk.RegisterZKLogger()
	dsn := coordzk.NewDSN(zks, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	return coordclient.New("zookeeper", dsn, "/", nil)
}

func (d *daemon) startMaster() (err error) {
	agentIP := options.OutboundIP
	if agentIP == "" {
		agentIP, err = utils.GetIPAddress()
		if err != nil {
			glog.Fatalf("Failed to acquire ip address: %s", err)
		}
	}

	// This is storage related
	rpcPort := strings.TrimLeft(options.Listen, ":")
	thisHost, err := host.Build(agentIP, rpcPort, d.masterPoolID, "")
	if err != nil {
		glog.Errorf("could not build host for agent IP %s: %v", agentIP, err)
		return err
	}

	if options.FSType == "btrfs" {
		if !volume.IsBtrfsFilesystem(options.VolumesPath) {
			return fmt.Errorf("path %s is not btrfs", options.VolumesPath)
		}
	}
	if d.disk, err = volume.GetDriver(options.VolumesPath); err != nil {
		glog.Errorf("Could not get volume driver at %s: %s", options.VolumesPath, err)
		return err
	}
	if d.net, err = nfs.NewServer(options.VolumesPath, "serviced_volumes_v2", "0.0.0.0/0"); err != nil {
		glog.Errorf("Could not initialize network driver: %s", err)
		return err
	}
	//set tenant volumes on nfs storagedriver
	glog.Infoln("Finding volumes")
	tenantVolumes := make(map[string]struct{})
	for _, vol := range d.disk.List() {
		glog.V(2).Infof("Getting tenant volume for %s", vol)
		if tVol, err := d.disk.GetTenant(vol); err == nil {
			if _, found := tenantVolumes[tVol.Path()]; !found {
				tenantVolumes[tVol.Path()] = struct{}{}
				glog.Infof("tenant volume %s found for export", tVol.Path())
				d.net.AddVolume(tVol.Path())
			}
		} else {
			glog.Warningf("Could not get Tenant for volume %s: %v", vol, err)
		}
	}

	if d.storageHandler, err = storage.NewServer(d.net, thisHost, options.VolumesPath); err != nil {
		glog.Errorf("Could not start network server: %s", err)
		return err
	}

	if d.dsDriver, err = d.initDriver(); err != nil {
		glog.Errorf("Could not initialize driver: %s", err)
		return err
	}

	if d.dsContext, err = d.initContext(); err != nil {
		glog.Errorf("Could not initialize context: %s", err)
		return err
	}

	d.facade = d.initFacade()

	if d.cpDao, err = d.initDAO(); err != nil {
		glog.Errorf("Could not initialize DAO: %s", err)
		return err
	}

	if err = d.facade.CreateDefaultPool(d.dsContext, d.masterPoolID); err != nil {
		glog.Errorf("Could not create default pool: %s", err)
		return err
	}

	if err = d.facade.UpgradeRegistry(d.dsContext, "", false); err != nil {
		glog.Errorf("Could not upgrade registry: %s", err)
		return err
	}

	if err = d.registerMasterRPC(); err != nil {
		glog.Errorf("Could not register master RPCs: %s", err)
		return err
	}

	d.initWeb()
	d.addTemplates()
	d.startScheduler()

	return nil
}

func getKeyPairs(certPEMFile, keyPEMFile string) (certPEM, keyPEM []byte, err error) {
	if len(certPEMFile) > 0 {
		certPEM, err = ioutil.ReadFile(certPEMFile)
		if err != nil {
			return
		}
	} else {
		certPEM = []byte(proxy.InsecureCertPEM)
	}
	if len(keyPEMFile) > 0 {
		keyPEM, err = ioutil.ReadFile(keyPEMFile)
		if err != nil {
			return
		}
	} else {
		keyPEM = []byte(proxy.InsecureKeyPEM)
	}
	return
}

func getTLSConfig() (*tls.Config, error) {
	proxyCertPEM, proxyKeyPEM, err := getKeyPairs(options.CertPEMFile, options.KeyPEMFile)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
	if err != nil {
		glog.Error("Could not parse public/private key pair (tls.X509KeyPair): ", err)
		return nil, err
	}

	tlsConfig := tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               utils.MinTLS(),
		PreferServerCipherSuites: true,
		CipherSuites:             utils.CipherSuites(),
	}
	return &tlsConfig, nil

}

func createMuxListener() (net.Listener, error) {
	if options.TLS {
		glog.V(1).Info("using TLS on mux")

		tlsConfig, err := getTLSConfig()
		if err != nil {
			return nil, err
		}
		glog.V(1).Infof("TLS enabled tcp mux listening on %d", options.MuxPort)
		return tls.Listen("tcp", fmt.Sprintf(":%d", options.MuxPort), tlsConfig)

	}
	return net.Listen("tcp", fmt.Sprintf(":%d", options.MuxPort))
}

func (d *daemon) startAgent() error {
	muxListener, err := createMuxListener()
	if err != nil {
		glog.Errorf("Could not create mux listener: %s", err)
		return err
	}
	mux, err := proxy.NewTCPMux(muxListener)
	if err != nil {
		glog.Errorf("Could not create TCP mux listener: %s", err)
		return err
	}

	agentIP := options.OutboundIP
	if agentIP == "" {
		var err error
		agentIP, err = utils.GetIPAddress()
		if err != nil {
			glog.Fatalf("Failed to acquire ip address: %s", err)
		}
	}

	rpcPort := strings.TrimLeft(options.Listen, ":")
	thisHost, err := host.Build(agentIP, rpcPort, "unknown", "", options.StaticIPs...)
	if err != nil {
		glog.Fatalf("Failed to acquire all host info: %s", err)
	}

	myHostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("HostID failed: %v", err)
		return err
	} else if err := validation.ValidHostID(myHostID); err != nil {
		glog.Errorf("invalid hostid: %s", myHostID)
	}

	go func() {
		var poolID string
		for {
			poolID = func() string {
				glog.Infof("Trying to discover my pool...")
				var myHost *host.Host
				masterClient, err := master.NewClient(d.servicedEndpoint)
				if err != nil {
					glog.Errorf("master.NewClient failed (endpoint %+v) : %v", d.servicedEndpoint, err)
					return ""
				}
				defer masterClient.Close()
				myHost, err = masterClient.GetHost(myHostID)
				if err != nil {
					glog.Warningf("masterClient.GetHost %v failed: %v (has this host been added?)", myHostID, err)
					return ""
				}
				poolID = myHost.PoolID
				glog.Infof(" My PoolID: %v", poolID)
				//send updated host info
				updatedHost, err := host.UpdateHostInfo(*myHost)
				if err != nil {
					glog.Infof("Could not send updated host information: %v", err)
					return poolID
				}
				err = masterClient.UpdateHost(updatedHost)
				if err != nil {
					glog.Warningf("Could not update host information: %v", err)
					return poolID
				}
				glog.V(2).Infof("Sent updated host info %#v", updatedHost)
				return poolID
			}()
			if poolID != "" {
				break
			}
			select {
			case <-d.shutdown:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		thisHost.PoolID = poolID

		poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
		if err != nil {
			glog.Errorf("Error in getting a connection based on pool %v: %v", poolID, err)
		}

		if options.NFSClient != "0" {
			nfsClient, err := storage.NewClient(thisHost, options.VolumesPath)
			if err != nil {
				glog.Fatalf("could not create an NFS client: %s", err)
			}

			go func() {
				<-d.shutdown
				glog.Infof("shutting down storage client")
				nfsClient.Close()
			}()

			//loop and log waiting for Storage Leader
			nfsDone := make(chan struct{})
			go func() {
				defer close(nfsDone)
				nfsClient.Wait()
			}()
			//wait indefinitely(?) for storage to work before starting
			glog.Info("Waiting for Storage Leader")
			nfsUp := false
			for !nfsUp {
				select {
				case <-nfsDone:
					nfsUp = true
					glog.Info("Found Storage Leader")
					break
				case <-time.After(time.Second * 30):
					glog.Info("Waiting for Storage Leader, will not be available for running services. ")
					continue
				}
			}
		} else {
			glog.Info("NFS Client disabled")
		}

		agentOptions := node.AgentOptions{
			PoolID:               thisHost.PoolID,
			Master:               options.Endpoint,
			UIPort:               options.UIPort,
			RPCPort:              options.RPCPort,
			DockerDNS:            options.DockerDNS,
			VolumesPath:          options.VolumesPath,
			Mount:                options.Mount,
			FSType:               options.FSType,
			Zookeepers:           options.Zookeepers,
			Mux:                  mux,
			UseTLS:               options.TLS,
			DockerRegistry:       dockerRegistry,
			MaxContainerAge:      time.Duration(int(time.Second) * options.MaxContainerAge),
			VirtualAddressSubnet: options.VirtualAddressSubnet,
			ControllerBinary:     options.ControllerBinary,
			LogstashURL:          options.LogstashURL,
			DockerLogDriver:      options.DockerLogDriver,
			DockerLogConfig:      convertStringSliceToMap(options.DockerLogConfigList),
		}
		// creates a zClient that is not pool based!
		hostAgent, err := node.NewHostAgent(agentOptions, d.reg)
		d.hostAgent = hostAgent

		d.waitGroup.Add(1)
		go func() {
			hostAgent.Start(d.shutdown)
			glog.Info("Host Agent has shutdown")
			d.waitGroup.Done()
		}()

		// register the API
		glog.V(0).Infoln("registering ControlPlaneAgent service")
		if err = d.rpcServer.RegisterName("ControlPlaneAgent", hostAgent); err != nil {
			glog.Fatalf("could not register ControlPlaneAgent RPC server: %v", err)
		}

		if options.Master {
			rpcutils.RegisterLocal("ControlPlaneAgent", hostAgent)
		}
		if options.ReportStats {
			statsdest := fmt.Sprintf("http://%s/api/metrics/store", options.HostStats)
			statsduration := time.Duration(options.StatsPeriod) * time.Second
			glog.V(1).Infoln("Staring container statistics reporter")
			statsReporter, err := stats.NewStatsReporter(statsdest, statsduration, poolBasedConn, options.Master, d.docker)
			if err != nil {
				glog.Errorf("Error kicking off stats reporter %v", err)
			} else {
				go func() {
					defer statsReporter.Close()
					<-d.shutdown
				}()
			}
		}
	}()

	agentServer := agent.NewServer(d.staticIPs)
	if err = d.rpcServer.RegisterName("Agent", agentServer); err != nil {
		glog.Fatalf("could not register Agent RPC server: %v", err)
	}
	if err != nil {
		glog.Fatalf("Could not start ControlPlane agent: %v", err)
	}
	if options.Master {
		rpcutils.RegisterLocal("Agent", agentServer)
	}

	// TODO: Integrate this server into the rpc server, or something.
	// Currently its only use is for command execution.
	go func() {
		sio := shell.NewProcessExecutorServer(options.Endpoint, dockerRegistry, options.ControllerBinary, options.UIPort)
		http.ListenAndServe(":50000", sio)
	}()

	return nil
}

func (d *daemon) registerMasterRPC() error {
	glog.V(0).Infoln("registering Master RPC services")

	server := master.NewServer(d.facade)
	disableLocal := os.Getenv("DISABLE_RPC_BYPASS")
	if disableLocal == "" {
		rpcutils.RegisterLocalAddress(options.Endpoint, fmt.Sprintf("localhost:%s", options.RPCPort),
			fmt.Sprintf("127.0.0.1:%s", options.RPCPort))
	} else {
		glog.V(0).Infoln("Enabling RPC for local calls; disabling reflection lookup")
	}
	rpcutils.RegisterLocal("Master", server)
	if err := d.rpcServer.RegisterName("Master", server); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}

	// register the deprecated rpc servers
	rpcutils.RegisterLocal("LoadBalancer", d.cpDao)
	if err := d.rpcServer.RegisterName("LoadBalancer", d.cpDao); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}
	rpcutils.RegisterLocal("ControlPlane", d.cpDao)
	if err := d.rpcServer.RegisterName("ControlPlane", d.cpDao); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}
	return nil
}

func (d *daemon) initDriver() (datastore.Driver, error) {

	eDriver := elastic.New("localhost", 9200, "controlplane")
	eDriver.AddMapping(host.MAPPING)
	eDriver.AddMapping(pool.MAPPING)
	eDriver.AddMapping(servicetemplate.MAPPING)
	eDriver.AddMapping(service.MAPPING)
	eDriver.AddMapping(addressassignment.MAPPING)
	eDriver.AddMapping(serviceconfigfile.MAPPING)
	eDriver.AddMapping(user.MAPPING)
	err := eDriver.Initialize(10 * time.Second)
	if err != nil {
		return nil, err
	}
	return eDriver, nil
}

func initMetricsClient() *metrics.Client {
	addr := fmt.Sprintf("http://%s:8888", localhost)
	client, err := metrics.NewClient(addr)
	if err != nil {
		glog.Errorf("Unable to initiate metrics client to %s", addr)
		return nil
	}
	return client
}

func (d *daemon) initFacade() *facade.Facade {
	f := facade.New()
	zzk := facade.GetFacadeZZK(f)
	f.SetZZK(zzk)
	index := registry.NewRegistryIndexClient(f)
	dfs := dfs.NewDistributedFilesystem(d.docker, index, d.reg, d.disk, d.net, time.Duration(options.MaxDFSTimeout)*time.Second)
	dfs.SetTmp(os.Getenv("TMP"))
	f.SetDFS(dfs)
	f.SetIsvcsPath(options.IsvcsPath)
	d.hcache = health.New()
	d.hcache.SetPurgeFrequency(5 * time.Second)
	f.SetHealthCache(d.hcache)
	client := initMetricsClient()
	f.SetMetricsClient(client)
	return f
}

// startLogstashPurger purges logstash based on days and size
func (d *daemon) startLogstashPurger(initialStart, cycleTime time.Duration) {
	// Run the first time after 10 minutes
	select {
	case <-d.shutdown:
		return
	case <-time.After(initialStart):
	}
	for {
		isvcs.PurgeLogstashIndices(options.LogstashMaxDays, options.LogstashMaxSize)
		select {
		case <-d.shutdown:
			return
		case <-time.After(cycleTime):
		}
	}
}

func (d *daemon) initDAO() (dao.ControlPlane, error) {
	rpcPortInt, err := strconv.Atoi(options.RPCPort)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(options.BackupsPath, 0777); err != nil && !os.IsExist(err) {
		glog.Fatalf("Could not create default backup path at %s: %s", options.BackupsPath, err)
	}
	return elasticsearch.NewControlSvc("localhost", 9200, d.facade, options.BackupsPath, rpcPortInt)
}

func (d *daemon) initWeb() {
	// TODO: Make bind port for web server optional?
	glog.V(4).Infof("Starting web server: uiport: %v; port: %v; zookeepers: %v", options.UIPort, options.Endpoint, options.Zookeepers)
	cpserver := web.NewServiceConfig(
		options.UIPort,
		options.Endpoint,
		options.ReportStats,
		options.HostAliases,
		options.TLS,
		options.MuxPort,
		options.AdminGroup,
		options.CertPEMFile,
		options.KeyPEMFile,
		options.UIPollFrequency,
		d.facade)
	web.SetServiceStatsCacheTimeout(options.SvcStatsCacheTimeout)
	go cpserver.Serve(d.shutdown)
	go cpserver.ServePublicPorts(d.shutdown, d.cpDao)
}

func (d *daemon) startScheduler() {
	go d.runScheduler()
}

func (d *daemon) addTemplates() {
	root := utils.LocalDir("templates")
	glog.V(1).Infof("Adding templates from %s", root)
	// Don't block startup for this. It's merely a convenience.
	go func() {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info == nil || !strings.HasSuffix(info.Name(), ".json") {
				return nil
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			var reader io.ReadCloser
			if reader, err = os.Open(path); err != nil {
				glog.Warningf("Unable to open template %s", path)
				return nil
			}
			defer reader.Close()
			st := servicetemplate.ServiceTemplate{}
			if err := json.NewDecoder(reader).Decode(&st); err != nil {
				glog.Warningf("Unable to parse template file %s", path)
				return nil
			}
			glog.V(1).Infof("Adding service template %s", path)
			d.facade.AddServiceTemplate(d.dsContext, st)
			return nil
		})
		if err != nil {
			glog.Warningf("Not loading templates from %s: %s", root, err)
		}
	}()
}

func (d *daemon) runScheduler() {
	for {
		sched, err := scheduler.NewScheduler(d.masterPoolID, d.hostID, d.storageHandler, d.cpDao, d.facade, d.reg, options.SnapshotTTL)
		if err != nil {
			glog.Errorf("Could not start scheduler: %s", err)
			return
		}

		sched.Start()
		select {
		case <-d.shutdown:
			glog.Info("Shutting down scheduler")
			sched.Stop()
			glog.Info("Scheduler stopped")
			return
		}
	}
}
