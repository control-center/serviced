// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elastic

import (
	check "gopkg.in/check.v1"

	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	esVersion = "0.90.0"
)

// ElasticTest for running tests that need elasticsearch. Type is to be used a a gocheck Suite. When writing a test,
// embed ElasticTest to create a test suite that will automatically start and stop elasticsearch. See gocheck
// documentation for more infomration about writing gocheck tests.
type ElasticTest struct {
	driver ElasticDriver
	server *testCluster
	//InitTimeout in seconds to wait for elastic to start
	InitTimeout time.Duration
	//Index for initializing driver, must be set
	Index string
	//Port for elasticsearch server
	Port uint16
	//Mappings are elastic mappings to initialize
	Mappings map[string]string
}

//setDefaults sets up sane defaults for what it can. Fatal if required values not set.
func (et *ElasticTest) setDefaults(c *check.C) {
	if et.Index == "" {
		c.Fatal("index required to run ElasticTest")
	}
	if et.InitTimeout == 0 {
		et.InitTimeout = 60
	}
	if et.Port == 0 {
		et.Port = 9202
	}
}

//SetUpSuite Run once when the suite starts running.
func (et *ElasticTest) SetUpSuite(c *check.C) {
	log.Print("ElasticTest SetUpSuite called")
	et.setDefaults(c)
	driver := new("localhost", et.Port, et.Index)
	et.driver = driver

	//is elastic already running?
	if !driver.isUp() {
		//download elastic jar if needed
		dir := ensureElasticJar()
		//start elastic
		tc, err := newTestCluster(dir, et.Port)
		if err != nil {
			tc.Stop()
			c.Fatalf("error in SetUpSuite: %v", err)
		}
		et.server = tc
	}

	for name, path := range et.Mappings {
		driver.AddMappingFile(name, path)
	}

	err := driver.Initialize(time.Second * et.InitTimeout)
	if err != nil {
		c.Fatalf("error in SetUpSuite: %v", err)
	}

}

//TearDownSuite Run once after all tests or benchmarks have finished running.
func (et *ElasticTest) TearDownSuite(c *check.C) {
	log.Print("ElasticTest TearDownSuite called")

	et.stop()
}

//Driver returns the initialized driver
func (et *ElasticTest) Driver() ElasticDriver {
	return et.driver
}

func (et *ElasticTest) stop() error {
	if et.server != nil {
		et.server.Stop()
	}
	return nil
}

type testCluster struct {
	tmpDir     string
	cmd        *exec.Cmd
	clientPort int
	shutdown   bool
}

func (tc *testCluster) Stop() error {
	tc.shutdown = true
	if tc.cmd != nil && tc.cmd.Process != nil {
		log.Print("Stop called, killing elastic search")
		return tc.cmd.Process.Kill()
	}
	return nil
}

func newTestCluster(elasticDir string, port uint16) (*testCluster, error) {

	tc := &testCluster{}
	tc.shutdown = false

	command := []string{elasticDir + "/bin/elasticsearch", "-f", fmt.Sprintf("-Des.http.port=%v", port)}
	cmd := exec.Command(command[0], command[1:]...)
	tc.cmd = cmd
	go func() {
		log.Printf("Starting elastic on port %v....: %v\n", port, command)
		out, err := cmd.CombinedOutput()
		if err != nil && !tc.shutdown {
			log.Printf("%s :%s\n", out, err) // do stuff
		}
	}()
	return tc, nil
}

func ensureElasticJar() string {
	_, err := exec.LookPath("java")
	if err != nil {
		log.Fatal("Can't find java in path")
	}

	gz := fmt.Sprintf("elasticsearch-%s.tar.gz", esVersion)
	path := fmt.Sprintf("/tmp/elasticsearch-%s", esVersion)

	log.Printf("checking %s exists", path)
	if _, err = os.Stat(path); err != nil {
		url := fmt.Sprintf("https://download.elasticsearch.org/elasticsearch/elasticsearch/%s", gz)
		commands := [][]string{
			{"curl", "-O", url},
			{"mv", gz, "/tmp/."},
			{"tar", "-xvzf", "/tmp/" + gz, "-C", "/tmp/."},
		}

		for _, cmd := range commands {
			log.Printf("exec: %v", cmd)
			err = exec.Command(cmd[0], cmd[1:]...).Run()
			if err != nil {
				log.Fatalf("could not execute %v: %v", strings.Join(cmd, " "), err)
			}
		}
	}
	return path
}
