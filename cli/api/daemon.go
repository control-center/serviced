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

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/auth"
	commonsdocker "github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/config"
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
	"github.com/control-center/serviced/logging"
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
	"github.com/control-center/serviced/volume/devicemapper"

	"github.com/control-center/serviced/web"
	"github.com/control-center/serviced/zzk"

	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
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

	_ "net/http/pprof"
)

var (
	minDockerVersion = version{1, 9, 0}
	log              = logging.PackageLogger()
)

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
	tokenExpiration  time.Duration

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

func newDaemon(servicedEndpoint string, staticIPs []string, masterPoolID string, tokenExpiration time.Duration) (*daemon, error) {
	d := &daemon{
		servicedEndpoint: servicedEndpoint,
		staticIPs:        staticIPs,
		masterPoolID:     masterPoolID,
		shutdown:         make(chan interface{}),
		waitGroup:        &sync.WaitGroup{},
		rpcServer:        rpc.NewServer(),
		tokenExpiration:  tokenExpiration,
	}
	return d, nil
}

func (d *daemon) getEsClusterName(name string) string {
	var (
		clusterName string
		err         error
	)
	options := config.GetOptions()
	filename := path.Join(options.IsvcsPath, name+".clustername")
	data, _ := ioutil.ReadFile(filename)
	clusterName = string(bytes.TrimSpace(data))
	if clusterName == "" {
		clusterName, err = utils.NewUUID36()
		if err != nil {
			log.WithError(err).Fatal("Unable to generate UUID")
		}
		if err = os.MkdirAll(filepath.Dir(filename), 0770); err != nil && !os.IsExist(err) {
			log.WithError(err).WithFields(logrus.Fields{"file": filename}).Fatal("Unable to create path to file")
		}
		if err = ioutil.WriteFile(filename, []byte(clusterName), 0600); err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"file":        filename,
				"clustername": clusterName,
			}).Fatal("Unable to write cluster name to file")
		}
	}
	return clusterName
}

func (d *daemon) startISVCS() {
	options := config.GetOptions()
	isvcs.Init(options.ESStartupTimeout, options.DockerLogDriver, convertStringSliceToMap(options.DockerLogConfigList), d.docker)
	isvcs.Mgr.SetVolumesDir(options.IsvcsPath)
	servicedClusterName := d.getEsClusterName("elasticsearch-serviced")
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-serviced", "cluster", servicedClusterName); err != nil {
		log.WithFields(logrus.Fields{
			"clustername": servicedClusterName,
		}).WithError(err).Fatal("Could not set Elastic configuration")
	}
	logstashClusterName := d.getEsClusterName("elasticsearch-logstash")
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-logstash", "cluster", logstashClusterName); err != nil {
		log.WithFields(logrus.Fields{
			"clustername": logstashClusterName,
		}).WithError(err).Fatal("Could not set Elastic configuration")
	}
	if err := isvcs.Mgr.Start(); err != nil {
		log.WithError(err).Fatal("Unable to start internal services")
	}
	log.Info("Started internal services")
	go d.startLogstashPurger(10*time.Minute, time.Duration(options.LogstashCycleTime)*time.Hour)
}

func (d *daemon) startAgentISVCS(serviceNames []string) {
	options := config.GetOptions()
	log := log.WithFields(logrus.Fields{"services": serviceNames})
	isvcs.InitServices(serviceNames, options.DockerLogDriver, convertStringSliceToMap(options.DockerLogConfigList), d.docker)
	isvcs.Mgr.SetVolumesDir(options.IsvcsPath)
	if err := isvcs.Mgr.Start(); err != nil {
		log.WithError(err).Fatal("Unable to start internal services")
	}
	log.Info("Started internal services")
}

func (d *daemon) stopISVCS() {
	log.Debug("Beginning shutdown of internal services")
	if err := isvcs.Mgr.Stop(); err != nil {
		log.WithError(err).Error("Error while stopping internal services")
	}
	log.Info("Shut down internal services")
}

func (d *daemon) startRPC() {
	options := config.GetOptions()
	if options.DebugPort > 0 {
		address := fmt.Sprintf("127.0.0.1:%d", options.DebugPort)
		logger := log.WithFields(logrus.Fields{
			"server":  "debug",
			"address": address,
		})
		go func() {
			if err := http.ListenAndServe(address, nil); err != nil {
				logger.Warning("Unable to bind to debug port. Is another instance running?")
				return
			}
			logger.Info("Listening for incoming debug connections")
		}()
	}

	logger := log.WithFields(logrus.Fields{
		"tls":     !rpcutils.RPCDisableTLS,
		"server":  "rpc",
		"address": options.Listen,
	})

	var (
		listener net.Listener
		err      error
	)
	if rpcutils.RPCDisableTLS {
		listener, err = net.Listen("tcp", options.Listen)
	} else {
		var tlsConfig *tls.Config
		tlsConfig, err = getTLSConfig("rpc")
		if err != nil {
			logger.WithError(err).Fatal("Unable to retrieve TLS configuration")
		}
		logger = logger.WithFields(logrus.Fields{
			"ciphersuite": strings.Join(utils.CipherSuitesByName(tlsConfig), ","),
		})
		listener, err = tls.Listen("tcp", options.Listen, tlsConfig)
	}
	if err != nil {
		logger.Fatal("Unable to bind to RPC server address. Is another instance running?")
	}

	rpcutils.SetDialTimeout(options.RPCDialTimeout)
	d.rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	logger.Info("Listening for incoming RPC requests")

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.WithError(err).Fatal("Error accepting RPC connection")
			}
			go d.rpcServer.ServeCodec(rpcutils.NewDefaultAuthServerCodec(conn))
		}
	}()
}

func (d *daemon) run() (err error) {
	options := config.GetOptions()

	// Get the ID of this host
	if d.hostID, err = utils.HostID(); err != nil {
		log.WithError(err).Fatal("Unable to get host ID")
	} else if err := validation.ValidHostID(d.hostID); err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"hostid": d.hostID,
		}).Fatal("Invalid host ID")
	}

	// Validate that we have an acceptable version of Docker
	currentDockerVersion, err := node.GetDockerVersion()
	if err != nil {
		log.WithError(err).Fatal("Unable to get Docker version")
	} else if minDockerVersion.Compare(currentDockerVersion) < 0 {
		log.WithError(err).WithFields(logrus.Fields{
			"minversion": minDockerVersion,
			"version":    currentDockerVersion,
		}).Fatal("Incompatible Docker version")
	}

	// Establish a connection to Docker
	dockerlogger := log.WithFields(logrus.Fields{
		"version": currentDockerVersion,
		"address": docker.DefaultSocket,
	})
	if d.docker, err = docker.NewDockerClient(); err != nil {
		dockerlogger.WithError(err).Fatal("Unable to connect to Docker")
	}
	dockerlogger.Info("Established connection to Docker")

	// Set up the Docker registry
	d.reg = registry.NewRegistryListener(d.docker, options.DockerRegistry, d.hostID)

	// Initialize the application storage
	storagelogger := log.WithFields(logrus.Fields{
		"driver":  options.FSType,
		"path":    options.VolumesPath,
		"args":    options.StorageArgs,
		"options": options.StorageOptions,
	})
	storagelogger.Debug("Initializing application storage")
	if !volume.Registered(options.FSType) {
		storagelogger.Fatal("Invalid storage driver")
	}
	if !filepath.IsAbs(options.VolumesPath) {
		storagelogger.Fatal("Volume path is not absolute")
	}
	if err := volume.InitDriver(options.FSType, options.VolumesPath, options.StorageArgs); err != nil {
		storagelogger.WithError(err).Fatal("Unable to initialize application storage")
	}
	storagelogger.Info("Initialized application storage")

	// Start the RPC server
	d.startRPC()

	//Start the zookeeper client
	localClient, err := d.initZK(options.Zookeepers)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"ensemble": options.Zookeepers,
		}).Fatal("Unable to create a local ZooKeeper client")
	}
	zzk.InitializeLocalClient(localClient)
	log.Info("Established ZooKeeper connection")

	if options.Master {
		d.startISVCS()
		if err := d.startMaster(); err != nil {
			log.WithError(err).Fatal("Unable to start as a serviced master")
		}
	} else {
		d.startAgentISVCS(options.StartISVCS)
	}

	if options.Agent {
		if err := d.startAgent(); err != nil {
			log.WithError(err).Fatal("Unable to start as a serviced delegate")
		}
	}

	signalC := make(chan os.Signal, 10)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-signalC

	log.WithFields(logrus.Fields{
		"signal": sig,
	}).Info("Received interrupt")
	log.Debug("Beginning shutdown")
	close(d.shutdown)

	done := make(chan struct{})
	go func() {
		defer close(done)
		d.waitGroup.Wait()
		log.Info("Shut down subprocesses")
	}()

	select {
	case <-done:
		defer log.Info("Shut down serviced")
	case <-time.After(60 * time.Second):
		defer log.WithFields(logrus.Fields{"timeout": "60s"}).Warning("Timed out waiting for subprocess to shut down")
	}

	zzk.ShutdownConnections()
	log.Info("Disconnected from ZooKeeper")

	switch sig {
	case syscall.SIGHUP:
		command := os.Args
		log.WithFields(logrus.Fields{
			"command": command,
		}).Info("Restarting serviced process without shutting down internal services")
		syscall.Exec(command[0], command[0:], os.Environ())
	default:
		d.stopISVCS()
	}
	if d.hcache != nil {
		d.hcache.SetPurgeFrequency(0)
	}
	return nil
}

func (d *daemon) initContext() datastore.Context {
	log.Debug("Acquiring application context from Elastic")
	datastore.Register(d.dsDriver)
	ctx := datastore.Get()
	if ctx == nil {
		log.Fatal("Unable to acquire application context from Elastic")
	}
	return ctx
}

func (d *daemon) initZK(zks []string) (*coordclient.Client, error) {
	options := config.GetOptions()
	coordzk.RegisterZKLogger()
	dsn := coordzk.NewDSN(zks, time.Duration(options.ZKSessionTimeout)*time.Second).String()
	log.WithFields(logrus.Fields{
		"dsn":            dsn,
		"sessiontimeout": options.ZKSessionTimeout,
		"ensemble":       zks,
	}).Debug("Establishing connection to ZooKeeper")
	return coordclient.New("zookeeper", dsn, "/", nil)
}

func (d *daemon) startMaster() (err error) {
	log.Debug("Starting serviced master")
	options := config.GetOptions()
	agentIP := options.OutboundIP
	if agentIP == "" {
		agentIP, err = utils.GetIPAddress()
		if err != nil {
			log.WithError(err).Fatal("Unable to determine outbound IP address")
		}
	}
	log.WithFields(logrus.Fields{
		"address": agentIP,
	}).Info("Determined outbound IP address")

	rpcPort := strings.TrimLeft(options.Listen, ":")
	thisHost, err := host.Build(agentIP, rpcPort, d.masterPoolID, "")

	if err != nil {
		log.WithFields(logrus.Fields{
			"address": agentIP,
			"rpcport": rpcPort,
		}).WithError(err).Fatal("Unable to register master as host")
	}

	// Load keys if they exist, else generate them
	masterKeyFile := filepath.Join(options.IsvcsPath, auth.MasterKeyFileName)
	keylog := log.WithFields(logrus.Fields{
		"keyfile": masterKeyFile,
	})
	if err = auth.CreateOrLoadMasterKeys(masterKeyFile); err != nil {
		keylog.WithError(err).Fatal("Unable to load or create master keys")
	}

	keylog.Info("Loaded master keys from disk")

	// This is storage related
	storagelogger := log.WithFields(logrus.Fields{
		"path":   options.VolumesPath,
		"driver": options.FSType,
	})
	if options.FSType == "btrfs" {
		if !volume.IsBtrfsFilesystem(options.VolumesPath) {
			storagelogger.Fatal("Volume path does not contain a btrfs filesystem")
		}
	} else if options.FSType == "devicemapper" {
		devicemapper.SetStorageStatsUpdateInterval(options.StorageStatsUpdateInterval)
	}
	if d.disk, err = volume.GetDriver(options.VolumesPath); err != nil {
		storagelogger.WithError(err).Fatal("Unable to access application storage")
	}
	if d.net, err = nfs.NewServer(options.VolumesPath, "serviced_volumes_v2", "0.0.0.0/0"); err != nil {
		storagelogger.WithError(err).Fatal("Unable to initialize NFS server")
	}

	//set tenant volumes on nfs storagedriver
	log.Debug("Exporting tenant volumes via NFS")
	tenantVolumes := make(map[string]struct{})
	for _, vol := range d.disk.List() {
		tenantlogger := storagelogger.WithFields(logrus.Fields{"tenant": vol})
		tenantlogger.Debug("Exporting tenant volume")
		if tVol, err := d.disk.GetTenant(vol); err == nil {
			if _, found := tenantVolumes[tVol.Path()]; !found {
				tenantVolumes[tVol.Path()] = struct{}{}
				d.net.AddVolume(tVol.Path())
				tenantlogger.Info("Exported tenant volume via NFS")
			}
		} else {
			tenantlogger.WithError(err).Error("Unable to export tenant volume via NFS. Application data will not be available on remote hosts")
		}
	}

	if d.storageHandler, err = storage.NewServer(d.net, thisHost, options.VolumesPath); err != nil {
		log.WithError(err).Fatal("Unable to create internal NFS server manager")
	}

	d.dsDriver = d.initDriver()
	d.dsContext = d.initContext()
	d.facade = d.initFacade()
	d.cpDao = d.initDAO()

	if err = d.facade.CreateDefaultPool(d.dsContext, d.masterPoolID); err != nil {
		log.WithError(err).Fatal("Unable to create default pool")
	}

	if err = d.facade.UpgradeRegistry(d.dsContext, "", false); err != nil {
		log.WithError(err).Fatal("Unable to upgrade internal Docker image registry")
	}

	if err = d.registerMasterRPC(); err != nil {
		log.WithError(err).Fatal("Unable to register RPC services")
	}

	nfsServer, ok := d.net.(*nfs.Server)
	if ok {
		nfsServer.SetClientValidator(facade.NewDfsClientValidator(d.facade, d.dsContext))
	}

	d.initWeb()
	d.addTemplates()
	d.startScheduler()

	log.Info("Started serviced master")

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

func getTLSConfig(connectionType string) (*tls.Config, error) {
	options := config.GetOptions()
	proxyCertPEM, proxyKeyPEM, err := getKeyPairs(options.CertPEMFile, options.KeyPEMFile)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               utils.MinTLS(connectionType),
		PreferServerCipherSuites: true,
		CipherSuites:             utils.CipherSuites(connectionType),
	}
	return &tlsConfig, nil

}

func createMuxListener() net.Listener {
	options := config.GetOptions()
	var (
		listener net.Listener
		err      error
	)

	log := log.WithFields(logrus.Fields{
		"tls":  !options.MuxDisableTLS,
		"port": options.MuxPort,
	})
	log.Debug("Starting traffic multiplexer")

	if !options.MuxDisableTLS {
		tlsConfig, err := getTLSConfig("mux")
		if err != nil {
			log.WithError(err).Fatal("Invalid TLS configuration")
		}
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%d", options.MuxPort), tlsConfig)
		log = log.WithFields(logrus.Fields{
			"ciphersuite": strings.Join(utils.CipherSuitesByName(tlsConfig), ","),
		})
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", options.MuxPort))
	}
	if err != nil {
		log.WithError(err).Fatal("Unable to start traffic multiplexer")
	}
	log.Debug("Created TCP multiplexer")
	return listener
}

// Check if the pool the agent belongs to is allowed to access the DFS
func delegateHasDFSAccess() bool {
	identity := auth.CurrentIdentity()
	if identity == nil {
		return false
	}
	return identity.HasDFSAccess()
}

func (d *daemon) startAgent() error {
	options := config.GetOptions()
	muxListener := createMuxListener()
	mux, err := proxy.NewTCPMux(muxListener)
	if err != nil {
		log.WithError(err).Fatal("Could not start TCP multiplexer")
	}

	// Determine the delegate's IP address
	agentIP := options.OutboundIP
	if agentIP == "" {
		var err error
		agentIP, err = utils.GetIPAddress()
		if err != nil {
			log.WithError(err).Fatal("Unable to acquire outbound IP address")
		}
	}
	log.WithFields(logrus.Fields{
		"address": agentIP,
	}).Info("Determined delegate's outbound IP address")

	rpcPort := strings.TrimLeft(options.Listen, ":")
	thisHost, err := host.Build(agentIP, rpcPort, "unknown", "", options.StaticIPs...)
	if err != nil {
		log.WithFields(logrus.Fields{
			"address": agentIP,
			"rpcport": rpcPort,
		}).WithError(err).Fatal("Unable to register master as host")
	}

	myHostID, err := utils.HostID()
	if err != nil {
		log.WithError(err).Fatal("Unable to get host ID")
	} else if err := validation.ValidHostID(myHostID); err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"hostid": d.hostID,
		}).Fatal("Invalid host ID")
	}

	log := log.WithFields(logrus.Fields{
		"master": d.servicedEndpoint,
		"hostid": myHostID,
	})

	// Load delegate keys if they exist
	delegateKeyFile := filepath.Join(options.EtcPath, auth.DelegateKeyFileName)
	tokenFile := filepath.Join(options.EtcPath, auth.TokenFileName)

	// Start watching for delegate keys to be loaded
	go auth.WatchDelegateKeyFile(delegateKeyFile, d.shutdown)

	go func() {
		// Wait for delegate keys to exist before trying to authenticate
		select {
		case <-auth.WaitForDelegateKeys(d.shutdown):
		case <-d.shutdown:
			return
		}

		// Authenticate against the master
		getToken := func() (string, int64, error) {
			masterClient, err := master.NewClient(d.servicedEndpoint)
			if err != nil {
				return "", 0, err
			}
			defer masterClient.Close()
			token, expires, err := masterClient.AuthenticateHost(myHostID)
			if err != nil {
				return "", 0, err
			}
			return token, expires, nil
		}

		// Start authenticating
		auth.TokenLoop(getToken, tokenFile, d.shutdown)
	}()

	// Flag so we only log that a host hasn't been added yet once
	var loggedNoHost bool

	go func() {
		var poolID string
		for {
			poolID = func() string {
				log.Debug("Attempting to determine pool assignment for this delegate")
				var myHost *host.Host
				masterClient, err := master.NewClient(d.servicedEndpoint)
				if err != nil {
					log.WithError(err).Fatal("Unable to make RPC connection")
				}
				defer masterClient.Close()
				myHost, err = masterClient.GetHost(myHostID)
				if err != nil {
					if !loggedNoHost {
						log.Warn("Unable to find pool assignment for this delegate. Has it been added via `serviced host add`? Will continue to retry silently")
						loggedNoHost = true
					}
					return ""
				}
				poolID = myHost.PoolID
				log := log.WithFields(logrus.Fields{
					"poolid": poolID,
				})
				log.Info("Determined pool assignment for this delegate")
				//send updated host info
				updatedHost, err := host.UpdateHostInfo(*myHost)
				if err != nil {
					log.WithError(err).Warn("Unable to acquire delegate host information")
					return poolID
				}
				err = masterClient.UpdateHost(updatedHost)
				if err != nil {
					log.WithError(err).Warn("Unable to update master with delegate host information")
					return poolID
				}
				log.Info("Updated master with delegate host information")
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

		log = log.WithFields(logrus.Fields{
			"poolid": poolID,
		})

		poolPath := zzk.GeneratePoolPath(poolID)
		poolBasedConn, err := zzk.GetLocalConnection(poolPath)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"zkpath": poolPath,
			}).Fatal("Unable to establish pool-based connection to ZooKeeper")
		}
		log.WithFields(logrus.Fields{
			"zkpath": poolPath,
		}).Info("Established pool-based connection to ZooKeeper")

		if !delegateHasDFSAccess() {
			log.Debug("Did not mount the distributed filesystem. Delegate does not have DFS permissions")
		} else if options.NFSClient == "0" {
			log.Debug("Did not mount the distributed filesystem, since SERVICED_NFS_CLIENT is disabled on this host")
		} else {
			log := log.WithFields(logrus.Fields{
				"path": options.VolumesPath,
			})
			nfsClient, err := storage.NewClient(thisHost, options.VolumesPath)
			if err != nil {
				log.WithError(err).Fatal("Unable to connect to NFS server on the master")
			}

			go func() {
				<-d.shutdown
				log.Debug("Disconnecting from NFS server on the master")
				nfsClient.Close()
				log.Info("Disconnected from NFS server on the master")
			}()

			// loop and log waiting for Storage Leader
			nfsDone := make(chan struct{})
			go func() {
				defer close(nfsDone)
				nfsClient.Wait()
			}()

			// wait indefinitely(?) for storage to work before starting
			log.Debug("Waiting for a master to report in as storage leader")
			loggedTimeout := false
			nfsUp := false
			for !nfsUp {
				select {
				case <-nfsDone:
					nfsUp = true
					log.Info("Distributed filesystem is ready")
					break
				case <-time.After(time.Second * 30):
					if !loggedTimeout {
						log.Warn("No master has reported in as storage leader yet, so unable to run services. Will retry silently")
						loggedTimeout = true
					}
					continue
				}
			}
		}

		agentOptions := node.AgentOptions{
			IPAddress:            agentIP,
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
			UseTLS:               !options.MuxDisableTLS,
			DockerRegistry:       options.DockerRegistry,
			MaxContainerAge:      time.Duration(int(time.Second) * options.MaxContainerAge),
			VirtualAddressSubnet: options.VirtualAddressSubnet,
			ControllerBinary:     options.ControllerBinary,
			LogstashURL:          options.LogstashURL,
			DockerLogDriver:      options.DockerLogDriver,
			DockerLogConfig:      convertStringSliceToMap(options.DockerLogConfigList),
			ZKSessionTimeout:     options.ZKSessionTimeout,
			DelegateKeyFile:      delegateKeyFile,
			TokenFile:            tokenFile,
		}
		// creates a zClient that is not pool based!
		hostAgent, err := node.NewHostAgent(agentOptions, d.reg)
		d.hostAgent = hostAgent

		d.waitGroup.Add(1)
		go func() {
			hostAgent.Start(d.shutdown)
			log.Info("Shut down delegate")
			d.waitGroup.Done()
		}()

		// register the API
		log.Debug("Registering ControlCenterAgent RPC service")
		if err = d.rpcServer.RegisterName("ControlCenterAgent", hostAgent); err != nil {
			log.WithError(err).Fatal("Unable to register ControlCenterAgent RPC service")
		}

		if options.Master {
			rpcutils.RegisterLocal("ControlCenterAgent", hostAgent)
			log.Debug("Registered local ControlCenterAgent RPC service")
		}

		if options.ReportStats {
			statsdest := fmt.Sprintf("http://%s/api/metrics/store", options.HostStats)
			statsduration := time.Duration(options.StatsPeriod) * time.Second
			log := log.WithFields(logrus.Fields{
				"statsurl": statsdest,
				"interval": options.StatsPeriod,
			})
			log.Debug("Starting container statistics reporting")
			statsReporter, err := stats.NewStatsReporter(statsdest, statsduration, poolBasedConn, options.Master, d.docker)
			if err != nil {
				log.WithError(err).Error("Unable to start reporting stats")
			} else {
				go func() {
					defer statsReporter.Close()
					<-d.shutdown
					log.Info("Stopping stats reporting")
				}()
			}
		}
	}()

	agentServer := agent.NewServer(d.staticIPs)
	if err = d.rpcServer.RegisterName("Agent", agentServer); err != nil {
		log.WithError(err).Fatal("Unable to register Agent RPC service")
	}

	if options.Master {
		rpcutils.RegisterLocal("Agent", agentServer)
		log.Debug("Registered local Agent RPC service")
	}

	// TODO: Integrate this server into the rpc server, or something.
	// Currently its only use is for command execution.
	go func() {
		agentEndpoint := "localhost:" + options.RPCPort
		sio := shell.NewProcessExecutorServer(options.Endpoint, agentEndpoint, options.DockerRegistry, options.ControllerBinary)
		http.ListenAndServe(":50000", sio)
	}()

	return nil
}

func (d *daemon) registerMasterRPC() error {
	log.Debug("Registering master RPC services")
	options := config.GetOptions()

	server := master.NewServer(d.facade, d.tokenExpiration)
	disableLocal := os.Getenv("DISABLE_RPC_BYPASS")
	if disableLocal == "" {
		rpcutils.RegisterLocalAddress(options.Endpoint, fmt.Sprintf("localhost:%s", options.RPCPort),
			fmt.Sprintf("127.0.0.1:%s", options.RPCPort))
	} else {
		log.Debug("Enabled RPC for local calls")
	}
	rpcutils.RegisterLocal("Master", server)
	if err := d.rpcServer.RegisterName("Master", server); err != nil {
		return fmt.Errorf("could not register RPC server named Master: %v", err)
	}

	// register the deprecated rpc servers
	rpcutils.RegisterLocal("LoadBalancer", d.cpDao)
	if err := d.rpcServer.RegisterName("LoadBalancer", d.cpDao); err != nil {
		return fmt.Errorf("could not register RPC server named LoadBalancer: %v", err)
	}
	rpcutils.RegisterLocal("ControlCenter", d.cpDao)
	if err := d.rpcServer.RegisterName("ControlCenter", d.cpDao); err != nil {
		return fmt.Errorf("could not register RPC server named ControlCenter: %v", err)
	}
	return nil
}

func (d *daemon) initDriver() datastore.Driver {
	log := log.WithFields(logrus.Fields{
		"address": "localhost:9200",
		"index":   "controlplane",
	})
	log.Debug("Establishing connection with Elastic")
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
		log.WithError(err).Fatal("Unable to establish connection to Elastic database")
	}
	return eDriver
}

func initMetricsClient() *metrics.Client {
	addr := fmt.Sprintf("http://%s:8888", localhost)
	log := log.WithFields(logrus.Fields{
		"metricsaddr": addr,
	})
	client, err := metrics.NewClient(addr)
	if err != nil {
		log.WithError(err).Warn("Unable to connect to metrics server")
		return nil
	}
	log.Info("Established connection to metrics server")
	return client
}

func (d *daemon) initFacade() *facade.Facade {
	options := config.GetOptions()
	f := facade.New()
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
	if err := f.CreateSystemUser(d.dsContext); err != nil {
		log.WithError(err).Fatal("Unable to create system user")
	}
	return f
}

// startLogstashPurger purges logstash based on days and size
func (d *daemon) startLogstashPurger(initialStart, cycleTime time.Duration) {
	options := config.GetOptions()
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

// FIXME: The dao package is deprecated and should be removed.
func (d *daemon) initDAO() dao.ControlPlane {
	options := config.GetOptions()
	// Run the first time after 10 minutes
	rpcPortInt, err := strconv.Atoi(options.RPCPort)
	if err != nil {
		log.WithField("rpcPort", options.RPCPort).WithError(err).Fatal("RPC Port invalid")
	}
	if err := os.MkdirAll(options.BackupsPath, 0777); err != nil && !os.IsExist(err) {
		log.WithFields(logrus.Fields{
			"backupspath": options.BackupsPath,
		}).WithError(err).Fatal("Unable to create backup path")
	}
	cp, err := elasticsearch.NewControlSvc("localhost", 9200, d.facade, options.BackupsPath, rpcPortInt)
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize DAO layer")
	}
	return cp
}

func (d *daemon) initWeb() {
	options := config.GetOptions()
	// Run the first time after 10 minutes
	// TODO: Make bind port for web server optional?
	log := log.WithFields(logrus.Fields{
		"uiport": options.UIPort,
		"master": options.Endpoint,
	})
	log.Debug("Starting Control Center UI server")
	cpserver := web.NewServiceConfig(
		options.UIPort,
		options.Endpoint,
		options.ReportStats,
		options.HostAliases,
		!options.MuxDisableTLS,
		options.MuxPort,
		options.AdminGroup,
		options.CertPEMFile,
		options.KeyPEMFile,
		options.UIPollFrequency,
		options.SnapshotSpacePercent,
		d.facade)

	web.SetServiceStatsCacheTimeout(options.SvcStatsCacheTimeout)
	log.WithFields(logrus.Fields{
		"cachetimeout": options.SvcStatsCacheTimeout,
	}).Debug("Set service stats cache timeout to configured value")

	go cpserver.Serve(d.shutdown)
	log.Info("Started Control Center UI server")
}

func (d *daemon) startScheduler() {
	go d.runScheduler()
}

func (d *daemon) addTemplates() {
	root := utils.LocalDir("templates")
	log := log.WithFields(logrus.Fields{
		"path": root,
	})
	log.Debug("Loading service templates")
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
			log := log.WithFields(logrus.Fields{
				"template": info.Name(),
			})
			var reader io.ReadCloser
			if reader, err = os.Open(path); err != nil {
				log.Warn("Unable to open template file")
				return nil
			}
			defer reader.Close()
			st := servicetemplate.ServiceTemplate{}
			if err := json.NewDecoder(reader).Decode(&st); err != nil {
				log.Warn("Unable to parse template file")
				return nil
			}
			d.facade.AddServiceTemplate(d.dsContext, st)
			log.Debug("Added service template")
			return nil
		})
		if err != nil {
			log.WithError(err).Warn("Unable to autoload templates from the filesystem")
		} else {
			log.Info("Loaded service templates")
		}
	}()
}

func (d *daemon) runScheduler() {
	log.Debug("Starting service scheduler")
	options := config.GetOptions()
	// Run the first time after 10 minutes
	for {
		sched, err := scheduler.NewScheduler(d.masterPoolID, d.hostID, d.storageHandler, d.cpDao, d.facade, d.reg, options.SnapshotTTL)
		if err != nil {
			log.WithError(err).Fatal("Unable to start service scheduler")
			return
		}

		sched.Start()
		log.Info("Started service scheduler")
		select {
		case <-d.shutdown:
			log.Debug("Shutting down service scheduler")
			sched.Stop()
			log.Info("Stopped service scheduler")
			return
		}
	}
}
