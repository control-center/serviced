// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package main

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	coordzk "github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/rpc/master"
	"github.com/zenoss/serviced/rpc/agent"
	"github.com/zenoss/serviced/shell"
	"github.com/zenoss/serviced/stats"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/volume"
	_ "github.com/zenoss/serviced/volume/btrfs"
	_ "github.com/zenoss/serviced/volume/rsync"
	"github.com/zenoss/serviced/web"

	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// startDaemon starts the agent or master services on this host.
func startDaemon() {
	d := newDaemon()
	d.start()
}

type daemon struct {
	cpDao     dao.ControlPlane
	dsDriver  datastore.Driver
	dsContext datastore.Context
	facade    *facade.Facade
	hostID    string
	zclient  *coordclient.Client

}

func newDaemon() *daemon {
	return &daemon{}
}

func (d *daemon) start() {
	var err error
	d.hostID, err = utils.HostID()
	if err != nil {
		glog.Fatalf("Could not get hostid", err)
	}

	l, err := net.Listen("tcp", options.listen)
	if err != nil {
		glog.Fatalf("Could not bind to port %v. Is another instance running", err)
	}

	//This asserts isvcs
	//TODO: should this just be in startMaster
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir(options.varPath + "/isvcs")

	dockerVersion, err := serviced.GetDockerVersion()
	if err != nil {
		glog.Fatalf("Could not determine docker version: %s", err)
	}

	atLeast := []int{0, 8, 1}
	if compareVersion(atLeast, dockerVersion.Client) < 0 {
		glog.Fatal("serviced needs at least docker >= 0.8.1")
	}

	//TODO: is this needed for both agent and master?
	if _, ok := volume.Registered(options.vfs); !ok {
		glog.Fatalf("no driver registered for %s", options.vfs)
	}

	if options.master {
		if err = d.startMaster(); err != nil {
			glog.Fatalf("%v", err)
		}
	}
	if options.agent {
		if err = d.startAgent(); err != nil {
			glog.Fatalf("%v", err)
		}
	}

	rpc.HandleHTTP()

	if options.repstats {
		statsdest := fmt.Sprintf("http://%s/api/metrics/store", options.statshost)
		statsduration := time.Duration(options.statsperiod) * time.Second
		glog.V(1).Infoln("Staring container statistics reporter")
		statsReporter := stats.NewStatsReporter(statsdest, statsduration)
		defer statsReporter.Close()
	}

	glog.V(0).Infof("Listening on %s", l.Addr().String())
	http.Serve(l, nil) // start the server
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

	d.facade = d.initFacade()

	if d.zclient, err = d.initZK(); err != nil{
		return err
	}

	if d.cpDao, err = d.initDAO(); err != nil {
		return err
	}

	if err = d.facade.CreateDefaultPool(d.dsContext); err!= nil{
		return err
	}

	if err = d.regiseterMasterRPC(); err != nil {
		return err
	}

	d.initWeb()

	d.startScheduler()
	return nil
}

func (d *daemon) startAgent() error {
	mux := serviced.TCPMux{}

	mux.CertPEMFile = options.certPEMFile
	mux.KeyPEMFile = options.keyPEMFile
	mux.Enabled = true
	mux.Port = options.muxPort
	mux.UseTLS = options.tls

	_dns := strings.Split(options.dockerDns, ",")
	hostAgent, err := serviced.NewHostAgent(options.port, options.uiport, _dns, options.varPath, options.mount, options.vfs, options.zookeepers, mux)
	if err != nil {
		glog.Fatalf("Could not start ControlPlane agent: %v", err)
	}
	// register the API
	glog.V(0).Infoln("registering ControlPlaneAgent service")
	if err = rpc.RegisterName("ControlPlaneAgent", hostAgent); err != nil{
		glog.Fatalf("could not register ControlPlaneAgent RPC server: %v", err)
	}
	if err = rpc.RegisterName("Agent", agent.NewServer()); err != nil{
		glog.Fatalf("could not register Agent RPC server: %v", err)
	}


	go func() {
		signalChan := make(chan os.Signal, 10)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan
		glog.V(0).Info("Shutting down due to interrupt")
		err = hostAgent.Shutdown()
		if err != nil {
			glog.V(1).Infof("Agent shutdown with error: %v", err)
		}
		isvcs.Mgr.Stop()
		os.Exit(0)
	}()

	// TODO: Integrate this server into the rpc server, or something.
	// Currently its only use is for command execution.
	go func() {
		sio := shell.NewProcessExecutorServer(options.port)
		http.ListenAndServe(":50000", sio)
	}()
	return nil
}

func (d *daemon) regiseterMasterRPC() error {
	glog.V(0).Infoln("registering Master RPC services")

	if err := rpc.RegisterName("Master", master.NewServer()); err != nil {
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

	//TODO: figure out elastic mappings
	eDriver := elastic.New("localhost", 9200, "controlplane")
	err := eDriver.Initialize(10 * time.Second)
	if err != nil {
		return nil, err
	}
	return eDriver, nil

}

func (d *daemon) initFacade() *facade.Facade {
	f := facade.New()
	return f
}

func (d *daemon) initISVCS() error {
	return isvcs.Mgr.Start()
}

func (d *daemon)initZK() (*coordclient.Client, error){
	dsn := coordzk.NewDSN(options.zookeepers, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	zclient, err := coordclient.New("zookeeper", dsn, "", nil)
	return zclient, err
}

func (d *daemon) initDAO() (dao.ControlPlane, error) {
	return elasticsearch.NewControlSvc("localhost", 9200, d.facade, d.zclient, options.varPath, options.vfs)
}

func (d *daemon) initWeb() {
	// TODO: Make bind port for web server optional?
	glog.V(4).Infof("Starting web server: uiport: %v; port: %v; zookeepers: %v", options.uiport, options.port, options.zookeepers)
	cpserver := web.NewServiceConfig(options.uiport, options.port, options.zookeepers, options.repstats, options.hostaliases)
	go cpserver.ServeUI()
	go cpserver.Serve()

}
func (d *daemon) startScheduler() {
	go d.runScheduler()
}

func (d *daemon) runScheduler() {
	for {
		func() {
			conn, err := d.zclient.GetConnection()
			if err != nil {
				return
			}
			defer conn.Close()

			sched, shutdown := serviced.NewScheduler("", conn, d.hostID, d.cpDao, d.facade)
			sched.Start()
			select {
			case <-shutdown:
			}
		}()
	}

}
