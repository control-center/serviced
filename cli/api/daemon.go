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
	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/coordinator/storage"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dao/elasticsearch"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/dfs/nfs"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/scheduler"
	"github.com/control-center/serviced/shell"
	"github.com/control-center/serviced/stats"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
	// Need to do btrfs driver initializations
	_ "github.com/control-center/serviced/volume/btrfs"
	// Need to do rsync driver initializations
	_ "github.com/control-center/serviced/volume/rsync"
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
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	// Needed for profiling
	_ "net/http/pprof"
)

var minDockerVersion = version{0, 11, 1}

type daemon struct {
	servicedEndpoint string
	staticIPs        []string
	cpDao            dao.ControlPlane
	dsDriver         datastore.Driver
	dsContext        datastore.Context
	facade           *facade.Facade
	hostID           string
	zClient          *coordclient.Client
	storageHandler   *storage.Server
	masterPoolID     string
	hostAgent        *node.HostAgent
	shutdown         chan interface{}
	waitGroup        *sync.WaitGroup
	rpcServer        *rpc.Server
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

func (d *daemon) getEsClusterName(Type string) string {

	filename := path.Join(options.VarPath, "isvcs", Type+".clustername")
	clusterName := ""
	data, err := ioutil.ReadFile(filename)
	if err != nil || len(data) <= 0 {
		clusterName, err = utils.NewUUID36()
		if err != nil {
			glog.Fatalf("could not generate uuid: %s", err)
		}
		if err := os.MkdirAll(path.Dir(filename), 0770); err != nil {
			glog.Fatalf("could not create dir %s: %s", path.Dir(filename), err)
		}
		if err := ioutil.WriteFile(filename, []byte(clusterName), 0600); err != nil {
			glog.Fatalf("could not write clustername to %s: %s", filename, err)
		}
	} else {
		clusterName = strings.TrimSpace(string(data))
	}
	return clusterName
}

func (d *daemon) run() error {
	var err error
	d.hostID, err = utils.HostID()
	if err != nil {
		glog.Fatalf("could not get hostid: %s", err)
	}

	// Start the debug port listener, if configured
	if options.DebugPort > 0 {
		go func() {
			if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", options.DebugPort), nil); err != nil {
				glog.Errorf("Unable to bind debug port to %v. Is another instance running?", err)
				return
			}
			glog.Infof("Started debug listener on port %d", options.DebugPort)
		}()
	}

	l, err := net.Listen("tcp", options.Listen)
	if err != nil {
		glog.Fatalf("Could not bind to port %v. Is another instance running", err)
	}

	//This asserts isvcs
	//TODO: should this just be in startMaster
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir(path.Join(options.VarPath, "isvcs"))
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-serviced", "cluster", d.getEsClusterName("elasticsearch-serviced")); err != nil {
		glog.Fatalf("could not set es-serviced option: %s", err)
	}
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-logstash", "cluster", d.getEsClusterName("elasticsearch-logstash")); err != nil {
		glog.Fatalf("could not set es-logstash option: %s", err)
	}

	dockerVersion, err := node.GetDockerVersion()
	if err != nil {
		glog.Fatalf("could not determine docker version: %s", err)
	}

	if minDockerVersion.Compare(dockerVersion.Client) < 0 {
		glog.Fatalf("serviced needs at least docker >= %s", minDockerVersion)
	}

	//TODO: is this needed for both agent and master?
	if _, ok := volume.Registered(options.VFS); !ok {
		glog.Fatalf("no driver registered for %s", options.VFS)
	}

	if options.Master {
		if err = d.startMaster(); err != nil {
			glog.Fatalf("%v", err)
		}
	}
	if options.Agent {
		if err = d.startAgent(); err != nil {
			glog.Fatalf("%v", err)
		}
	}

	d.rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	glog.V(0).Infof("Listening on %s", l.Addr().String())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				glog.Fatalf("Error accepting connections: %s", err)
			}
			go d.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()

	signalChan := make(chan os.Signal, 10)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	glog.V(0).Info("Shutting down due to interrupt")
	close(d.shutdown)

	doneWaiting := make(chan struct{})
	go func() {
		glog.Info("Waiting for wait group")
		d.waitGroup.Wait()
		glog.Info("wait group is done")
		close(doneWaiting)
	}()

	select {
	case <-doneWaiting:
		glog.Info("Shutdown")
	case <-time.After(60 * time.Second):
		glog.Info("Timed out waiting for shutdown")
		//Return error???
	}
	// finally, close all connections to zookeeper
	zzk.ShutdownConnections()
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
	dsn := coordzk.NewDSN(zks, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	return coordclient.New("zookeeper", dsn, "/", nil)
}

func (d *daemon) startMaster() error {
	if err := d.initISVCS(); err != nil {
		return err
	}

	var err error
	if d.dsDriver, err = d.initDriver(); err != nil {
		return err
	}

	if d.dsContext, err = d.initContext(); err != nil {
		return err
	}

	localClient, err := d.initZK(options.Zookeepers)
	if err != nil {
		glog.Errorf("failed to create a local coordclient: %v", err)
		return err
	}
	zzk.InitializeLocalClient(localClient)

	if len(options.RemoteZookeepers) > 0 {
		remoteClient, err := d.initZK(options.RemoteZookeepers)
		if err != nil {
			glog.Warningf("failed to create a remote coordclient; running in disconnected mode: %v", err)
		} else {
			zzk.InitializeRemoteClient(remoteClient)
		}
	}

	d.facade = d.initFacade()

	if d.cpDao, err = d.initDAO(); err != nil {
		return err
	}

	if err = d.facade.CreateDefaultPool(d.dsContext, d.masterPoolID); err != nil {
		return err
	}

	if err = d.registerMasterRPC(); err != nil {
		return err
	}

	d.initWeb()
	d.startScheduler()
	d.addTemplates()

	agentIP := options.OutboundIP
	if agentIP == "" {
		var err error
		agentIP, err = utils.GetIPAddress()
		if err != nil {
			glog.Fatalf("Failed to acquire ip address: %s", err)
		}
	}

	// This is storage related
	thisHost, err := host.Build(agentIP, d.masterPoolID)
	if err != nil {
		glog.Errorf("could not build host for agent IP %s: %v", agentIP, err)
		return err
	}

	if err := os.MkdirAll(options.VarPath, 0755); err != nil {
		glog.Errorf("could not create varpath %s: %s", options.VarPath, err)
		return err
	}

	if nfsDriver, err := nfs.NewServer(options.VarPath, "serviced_var", "0.0.0.0/0"); err != nil {
		return err
	} else {
		d.storageHandler, err = storage.NewServer(nfsDriver, thisHost, d.zClient)
		if err != nil {
			return err
		}
		d.waitGroup.Add(1)
		go func() {
			defer d.waitGroup.Done()
			<-d.shutdown
			glog.Infof("Shutting down storage handler")
			d.storageHandler.Close()
		}()
	}

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

func createMuxListener() (net.Listener, error) {
	if options.TLS {
		glog.V(1).Info("using TLS on mux")

		proxyCertPEM, proxyKeyPEM, err := getKeyPairs(options.CertPEMFile, options.KeyPEMFile)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
		if err != nil {
			glog.Error("ListenAndMux Error (tls.X509KeyPair): ", err)
			return nil, err
		}

		tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
		glog.V(1).Infof("TLS enabled tcp mux listening on %d", options.MuxPort)
		return tls.Listen("tcp", fmt.Sprintf(":%d", options.MuxPort), &tlsConfig)

	}
	return net.Listen("tcp", fmt.Sprintf(":%d", options.MuxPort))
}

func (d *daemon) startAgent() error {
	muxListener, err := createMuxListener()
	if err != nil {
		return err
	}
	mux, err := proxy.NewTCPMux(muxListener)
	if err != nil {
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
	thisHost, err := host.Build(agentIP, "unknown")
	if err != nil {
		panic(err)
	}

	myHostID, err := utils.HostID()
	if err != nil {
		return fmt.Errorf("HostID failed: %v", err)
	}

	go func() {
		var poolID string
		for {
			glog.Infof("Trying to discover my pool...")
			var myHost *host.Host
			masterClient, err := master.NewClient(d.servicedEndpoint)
			if err != nil {
				glog.Errorf("master.NewClient failed (endpoint %+v) : %v", d.servicedEndpoint, err)
			} else {
				myHost, err = masterClient.GetHost(myHostID)
				if err != nil {
					glog.Warningf("masterClient.GetHost %v failed: %v (has this host been added?)", myHostID, err)
				}
			}
			if err != nil {
				//wait to try getting pool again or for shutdow, whichever comes first
				select {
				case <-d.shutdown:
					return
				case <-time.After(5 * time.Second):
					continue
				}
			}
			poolID = myHost.PoolID
			glog.Infof(" My PoolID: %v", poolID)
			//send updated host info
			updatedHost, err := host.UpdateHostInfo(*myHost)
			if err != nil {
				glog.Infof("Could not send updated host information: %v", err)
				break
			}
			err = masterClient.UpdateHost(updatedHost)
			if err != nil {
				glog.Warningf("Could not update host information: %v", err)
				break
			}
			glog.V(2).Infof("Sent updated host info %#v", updatedHost)
			break
		}

		thisHost.PoolID = poolID

		basePoolPath := "/pools/" + poolID
		dsn := coordzk.NewDSN(options.Zookeepers, time.Second*15).String()
		glog.Infof("zookeeper dsn: %s", dsn)
		zClient, err := coordclient.New("zookeeper", dsn, basePoolPath, nil)
		if err != nil {
			glog.Errorf("failed create a new coordclient: %v", err)
		}
		zzk.InitializeLocalClient(zClient)

		poolBasedConn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
		if err != nil {
			glog.Errorf("Error in getting a connection based on pool %v: %v", poolID, err)
		}

		nfsClient, err := storage.NewClient(thisHost, options.VarPath)
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
		agentOptions := node.AgentOptions{
			PoolID:               thisHost.PoolID,
			Master:               options.Endpoint,
			UIPort:               options.UIPort,
			DockerDNS:            options.DockerDNS,
			VarPath:              options.VarPath,
			Mount:                options.Mount,
			VFS:                  options.VFS,
			Zookeepers:           options.Zookeepers,
			Mux:                  mux,
			DockerRegistry:       options.DockerRegistry,
			MaxContainerAge:      time.Duration(int(time.Second) * options.MaxContainerAge),
			VirtualAddressSubnet: options.VirtualAddressSubnet,
		}
		// creates a zClient that is not pool based!
		hostAgent, err := node.NewHostAgent(agentOptions)
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

		if options.ReportStats {
			statsdest := fmt.Sprintf("http://%s/api/metrics/store", options.HostStats)
			statsduration := time.Duration(options.StatsPeriod) * time.Second
			glog.V(1).Infoln("Staring container statistics reporter")
			statsReporter, err := stats.NewStatsReporter(statsdest, statsduration, poolBasedConn)
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

	glog.Infof("agent start staticips: %v [%d]", d.staticIPs, len(d.staticIPs))
	if err = d.rpcServer.RegisterName("Agent", agent.NewServer(d.staticIPs)); err != nil {
		glog.Fatalf("could not register Agent RPC server: %v", err)
	}
	if err != nil {
		glog.Fatalf("Could not start ControlPlane agent: %v", err)
	}

	// TODO: Integrate this server into the rpc server, or something.
	// Currently its only use is for command execution.
	go func() {
		sio := shell.NewProcessExecutorServer(options.Endpoint, options.DockerRegistry)
		http.ListenAndServe(":50000", sio)
	}()

	return nil
}

func (d *daemon) registerMasterRPC() error {
	glog.V(0).Infoln("registering Master RPC services")

	if err := d.rpcServer.RegisterName("Master", master.NewServer(d.facade)); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}

	// register the deprecated rpc servers
	if err := d.rpcServer.RegisterName("LoadBalancer", d.cpDao); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}

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

func (d *daemon) initFacade() *facade.Facade {
	f := facade.New(options.DockerRegistry)
	return f
}

func (d *daemon) initISVCS() error {
	if err := isvcs.Mgr.Start(); err != nil {
		return err
	}

	// Start the logstash purger
	go func() {
		// Run the first time after 10 minutes
		select {
		case <-d.shutdown:
			return
		case <-time.After(10 * time.Minute):
			isvcs.PurgeLogstashIndices(options.LogstashMaxDays)
		}
		// Now run every 6 hours
		for {
			select {
			case <-d.shutdown:
				return
			case <-time.After(6 * time.Hour):
				isvcs.PurgeLogstashIndices(options.LogstashMaxDays)
			}
		}
	}()

	d.waitGroup.Add(1)
	go func() {
		defer d.waitGroup.Done()
		<-d.shutdown
		//give other things some time to close before killing ZK etc...
		time.Sleep(3 * time.Second)
		glog.Infof("Shutting down isvcs")
		isvcs.Mgr.Stop()
		glog.Infof("isvcs shut down")
	}()
	return nil
}

func (d *daemon) initDAO() (dao.ControlPlane, error) {
	dfsTimeout := time.Duration(options.MaxDFSTimeout) * time.Second
	return elasticsearch.NewControlSvc("localhost", 9200, d.facade, options.VarPath, options.VFS, dfsTimeout, options.DockerRegistry)
}

func (d *daemon) initWeb() {
	// TODO: Make bind port for web server optional?
	glog.V(4).Infof("Starting web server: uiport: %v; port: %v; zookeepers: %v", options.UIPort, options.Endpoint, options.Zookeepers)
	cpserver := web.NewServiceConfig(options.UIPort, options.Endpoint, options.ReportStats, options.HostAliases, options.TLS, options.MuxPort)
	go cpserver.ServeUI()
	go cpserver.Serve(d.shutdown)
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
		sched, err := scheduler.NewScheduler(d.masterPoolID, d.hostID, d.cpDao, d.facade)
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
