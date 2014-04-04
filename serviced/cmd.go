// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package main

// This is the main entry point for the application. Here we parse command line
// flags and either start a service or execute command line functions.

//svc "github.com/zenoss/serviced/svc"
import (
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/glog"

	"flag"
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"

)

// Store the command line options
var options struct {
	port             string
	listen           string
	master           bool
	dockerDns        string
	agent            bool
	muxPort          int
	tls              bool
	keyPEMFile       string
	certPEMFile      string
	varPath          string // Directory to store data, eg isvcs & service volumes
	resourcePath     string
	zookeepers       ListOpts
	repstats         bool
	statshost        string
	statsperiod      int
	mcusername       string
	mcpasswd         string
	mount            ListOpts
	resourceperiod   int
	vfs              string
	esStartupTimeout int
	hostaliases      string
}

var agentIP string

// getEnvVarInt() returns the env var as an int value or the defaultValue if env var is unset
func getEnvVarInt(envVar string, defaultValue int) int {
	envVarValue := os.Getenv(envVar)
	if len(envVarValue) > 0 {
		value, err := strconv.Atoi(envVarValue)
		if err != nil {
			glog.Errorf("Could not convert env var %s:%s to integer, error:%s", envVar, envVarValue, err)
			return defaultValue
		}
		return value
	}
	return defaultValue
}

// ensureMinimumInt sets the env var and command line flag to the given minimum if the value is less than the minimum
func ensureMinimumInt(envVar string, flagName string, minimum int) {
	theFlag := flag.Lookup(flagName)
	value, _ := strconv.Atoi(theFlag.Value.String())
	if value < minimum {
		glog.Infof("overriding flag %s:%s with minimum value of %v", flagName, theFlag.Value.String(), minimum)
		valueStr := strconv.Itoa(minimum)
		os.Setenv(envVar, valueStr)
		flag.Set(flagName, valueStr)
	} else {
		os.Setenv(envVar, theFlag.Value.String())
	}
}

// Setup flag options (static block)
func init() {
	var err error
	agentIP, err = utils.GetIPAddress()
	if err != nil {
		panic(err)
	}

	dockerDns := os.Getenv("SERVICED_DOCKER_DNS")
	flag.StringVar(&options.port, "port", agentIP+":4979", "port for remote serviced (example.com:8080)")
	flag.StringVar(&options.listen, "listen", ":4979", "port for local serviced (example.com:8080)")
	flag.StringVar(&options.dockerDns, "dockerDns", dockerDns, "docker dns configuration used for running containers (comma seperated list)")
	flag.BoolVar(&options.master, "master", false, "run in master mode, ie the control plane service")
	flag.BoolVar(&options.agent, "agent", false, "run in agent mode, ie a host in a resource pool")
	flag.IntVar(&options.muxPort, "muxport", 22250, "multiplexing port to use")
	flag.BoolVar(&options.tls, "tls", true, "enable TLS")

	varPathDefault := path.Join(os.TempDir(), "serviced")
	if len(os.Getenv("SERVICED_HOME")) > 0 {
		varPathDefault = path.Join(os.Getenv("SERVICED_HOME"), "var")
	} else {
		if user, err := user.Current(); err == nil {
			varPathDefault = path.Join(os.TempDir(), "serviced-"+user.Username, "var")
		}
	}
	flag.StringVar(&options.varPath, "varPath", varPathDefault, "path to store serviced data")

	flag.StringVar(&options.keyPEMFile, "keyfile", "", "path to private key file (defaults to compiled in private key)")
	flag.StringVar(&options.certPEMFile, "certfile", "", "path to public certificate file (defaults to compiled in public cert)")
	options.zookeepers = make(ListOpts, 0)
	flag.Var(&options.zookeepers, "zk", "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181 )")
	flag.BoolVar(&options.repstats, "reportstats", true, "report container statistics")
	flag.StringVar(&options.statshost, "statshost", "127.0.0.1:8443", "host:port for container statistics")
	flag.IntVar(&options.statsperiod, "statsperiod", 60, "Period (seconds) for container statistics reporting")
	flag.StringVar(&options.mcusername, "mcusername", "scott", "Username for the Zenoss metric consumer")
	flag.StringVar(&options.mcpasswd, "mcpasswd", "tiger", "Password for the Zenoss metric consumer")
	options.mount = make(ListOpts, 0)
	flag.Var(&options.mount, "mount", "bind mount: dockerImage,hostPath,containerPath (e.g. -mount zenoss/zenoss5x:latest,$HOME/src/europa/src,/mnt/src) dockerImage can be '*'")
	flag.StringVar(&options.vfs, "vfs", "rsync", "file system for container volumes")
	flag.StringVar(&options.hostaliases, "hostaliases", "", "list of aliases for this host, e.g., localhost:goldmine:goldmine.net")

	flag.IntVar(&options.esStartupTimeout, "esStartupTimeout", getEnvVarInt("ES_STARTUP_TIMEOUT", 600), "time to wait on elasticsearch startup before bailing")

	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

func compareVersion(a, b []int) int {
	astr := ""
	for _, s := range a {
		astr += fmt.Sprintf("%12d", s)
	}
	bstr := ""
	for _, s := range b {
		bstr += fmt.Sprintf("%12d", s)
	}
	if astr > bstr {
		return -1
	}
	if astr < bstr {
		return 1
	}
	return 0
}

// Start the agent or master services on this host.
func startServer() {
	daemon := newDaemon()
	daemon.start()
}

// main entry point of the product
func main() {

	// parse the command line flags
	flag.Parse()
	ensureMinimumInt("ES_STARTUP_TIMEOUT", "esStartupTimeout", 30)

	// are we in server mode
	if (options.master || options.agent) && len(flag.Args()) == 0 {
		startServer()
	} else {
		// we are in command line mode
		if len(flag.Args()) == 0 {
			// no arguments were give, show help
			cli := ServicedCli{}
			cli.CmdHelp(flag.Args()...)
			flag.Usage()
		} else {
			if err := ParseCommands(flag.Args()...); err != nil {
				glog.Fatalf("%s", err)
			}
		}
	}
	glog.Flush()
}
