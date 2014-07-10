// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package api

import (
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	coordzk "github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/coordinator/storage"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	"github.com/zenoss/serviced/dfs/nfs"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/serviceconfigfile"
	"github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/domain/user"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/node"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/rpc/agent"
	"github.com/zenoss/serviced/rpc/master"
	"github.com/zenoss/serviced/scheduler"
	"github.com/zenoss/serviced/shell"
	"github.com/zenoss/serviced/stats"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/volume"
	// Need to do btrfs driver initializations
	_ "github.com/zenoss/serviced/volume/btrfs"
	// Need to do rsync driver initializations
	_ "github.com/zenoss/serviced/volume/rsync"
	"github.com/zenoss/serviced/web"
	"github.com/zenoss/serviced/zzk"

	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"
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
}

func newDaemon(servicedEndpoint string, staticIPs []string, masterPoolID string) (*daemon, error) {
	d := &daemon{
		servicedEndpoint: servicedEndpoint,
		staticIPs:        staticIPs,
		masterPoolID:     masterPoolID,
	}
	return d, nil
}

func (d *daemon) run() error {
	var err error
	d.hostID, err = utils.HostID()
	if err != nil {
		glog.Fatalf("could not get hostid: %s", err)
	}

	l, err := net.Listen("tcp", options.Listen)
	if err != nil {
		glog.Fatalf("Could not bind to port %v. Is another instance running", err)
	}

	//This asserts isvcs
	//TODO: should this just be in startMaster
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir(path.Join(options.VarPath, "isvcs"))

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

	rpc.HandleHTTP()

	glog.V(0).Infof("Listening on %s", l.Addr().String())
	return http.Serve(l, nil) // start the server
}

func (d *daemon) initContext() (datastore.Context, error) {
	datastore.Register(d.dsDriver)
	ctx := datastore.Get()
	if ctx == nil {
		return nil, errors.New("context not available")
	}
	return ctx, nil
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

	rootBasePath := "/"
	dsn := coordzk.NewDSN(options.Zookeepers, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	zClient, err := coordclient.New("zookeeper", dsn, rootBasePath, nil)
	if err != nil {
		glog.Errorf("failed create a new coordclient: %v", err)
		return err
	}
	zzk.InitializeGlobalCoordClient(zClient)

	d.facade = d.initFacade()

	if d.cpDao, err = d.initDAO(); err != nil {
		return err
	}

	if err = d.facade.CreateDefaultPool(d.dsContext); err != nil {
		return err
	}

	if err = d.registerMasterRPC(); err != nil {
		return err
	}

	d.initWeb()
	d.startScheduler()

	agentIP, err := utils.GetIPAddress()
	if err != nil {
		panic(err)
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

	agentIP, err := utils.GetIPAddress()
	if err != nil {
		panic(err)
	}
	thisHost, err := host.Build(agentIP, "unknown")
	if err != nil {
		panic(err)
	}

	myHostID, err := utils.HostID()
	if err != nil {
		return fmt.Errorf("HostID failed: %v", err)
	}

	sleepRetry := 5
	go func() {
		var poolID string
		for {
			glog.Infof("Trying to discover my pool...")
			masterClient, err := master.NewClient(d.servicedEndpoint)
			if err != nil {
				glog.Errorf("master.NewClient failed (endpoint %+v) : %v", d.servicedEndpoint, err)
				time.Sleep(time.Duration(sleepRetry) * time.Second)
				continue
			}
			myHost, err := masterClient.GetHost(myHostID)
			if err != nil {
				glog.Warningf("masterClient.GetHost %v failed: %v (has this host been added?)", myHostID, err)
				time.Sleep(time.Duration(sleepRetry) * time.Second)
				continue
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
		zzk.InitializeGlobalCoordClient(zClient)

		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(poolID))
		if err != nil {
			glog.Errorf("Error in getting a connection based on pool %v: %v", poolID, err)
		}

		nfsClient, err := storage.NewClient(thisHost, options.VarPath)
		if err != nil {
			glog.Fatalf("could not create an NFS client: %s", err)
		}
		nfsClient.Wait()

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

		// register the API
		glog.V(0).Infoln("registering ControlPlaneAgent service")
		if err = rpc.RegisterName("ControlPlaneAgent", hostAgent); err != nil {
			glog.Fatalf("could not register ControlPlaneAgent RPC server: %v", err)
		}

		go func() {
			if options.ReportStats {
				statsdest := fmt.Sprintf("http://%s/api/metrics/store", options.HostStats)
				statsduration := time.Duration(options.StatsPeriod) * time.Second
				glog.V(1).Infoln("Staring container statistics reporter")
				statsReporter, err := stats.NewStatsReporter(statsdest, statsduration, poolBasedConn)
				if err != nil {
					glog.Errorf("Error kicking off stats reporter %v", err)
				}
				defer statsReporter.Close()
			}
			signalChan := make(chan os.Signal, 10)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			glog.V(0).Info("Shutting down due to interrupt")
			hostAgent.Shutdown()
			glog.Info("Agent shutdown")
			isvcs.Mgr.Stop()
			os.Exit(0)
		}()

	}()

	glog.Infof("agent start staticips: %v [%d]", d.staticIPs, len(d.staticIPs))
	if err = rpc.RegisterName("Agent", agent.NewServer(d.staticIPs)); err != nil {
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

	if err := rpc.RegisterName("Master", master.NewServer(d.facade)); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}

	// register the deprecated rpc servers
	if err := rpc.RegisterName("LoadBalancer", d.cpDao); err != nil {
		return fmt.Errorf("could not register rpc server LoadBalancer: %v", err)
	}

	if err := rpc.RegisterName("ControlPlane", d.cpDao); err != nil {
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
	return isvcs.Mgr.Start()
}

func (d *daemon) initDAO() (dao.ControlPlane, error) {
	return elasticsearch.NewControlSvc("localhost", 9200, d.facade, options.VarPath, options.VFS, options.DockerRegistry)
}

func (d *daemon) initWeb() {
	// TODO: Make bind port for web server optional?
	glog.V(4).Infof("Starting web server: uiport: %v; port: %v; zookeepers: %v", options.UIPort, options.Endpoint, options.Zookeepers)
	cpserver := web.NewServiceConfig(options.UIPort, options.Endpoint, options.ReportStats, options.HostAliases, options.TLS, options.MuxPort)
	go cpserver.ServeUI()
	go cpserver.Serve()
}
func (d *daemon) startScheduler() {
	go d.runScheduler()
}

func (d *daemon) runScheduler() {
	for {
		func() {
			sched, shutdown := scheduler.NewScheduler("", d.hostID, d.cpDao, d.facade)
			sched.Start()
			select {
			case <-shutdown:
			}
		}()
	}
}
