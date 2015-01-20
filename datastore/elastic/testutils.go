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

package elastic

import (
	gocheck "gopkg.in/check.v1"

	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	esVersion = "0.90.13"
)

// ElasticTest for running tests that need elasticsearch. Type is to be used a a gocheck Suite. When writing a test,
// embed ElasticTest to create a test suite that will automatically start and stop elasticsearch. See gocheck
// documentation for more infomration about writing gocheck tests.
type ElasticTest struct {
	driver *elasticDriver
	server *testCluster
	//InitTimeout in seconds to wait for elastic to start
	InitTimeout time.Duration
	//Index for initializing driver, must be set
	Index string
	//Port for elasticsearch server
	Port uint16
	//Mappings are elastic mappings to initialize
	Mappings []Mapping
	//MappingsFile path to a file that contains multiple mappings
	MappingsFile string
}

//setDefaults sets up sane defaults for what it can. Fatal if required values not set.
func (et *ElasticTest) setDefaults(c *gocheck.C) {
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
func (et *ElasticTest) SetUpSuite(c *gocheck.C) {
	log.Printf("ElasticTest SetUpSuite called.\n")
	et.setDefaults(c)
	driver := newDriver("localhost", et.Port, et.Index)
	et.driver = driver

	existingServer := true
	//is elastic already running?
	if !driver.isUp() {
		//Seeding because mkdir uses rand and it was returning the same directory
		rand.Seed(time.Now().Unix())
		//Create unique tmp dir that will be deleted when suite ends.
		tmpDir := c.MkDir()
		//download elastic jar if needed
		elasticDir := ensureElasticJar(tmpDir)
		//start elastic
		tc, err := newTestCluster(elasticDir, et.Port)
		if err != nil {
			tc.Stop()
			c.Fatalf("error in SetUpSuite: %v", err)
		}
		et.server = tc
		existingServer = false
	}

	for _, mapping := range et.Mappings {
		driver.AddMapping(mapping)
	}

	if et.MappingsFile != "" {
		if err := driver.AddMappingsFile(et.MappingsFile); err != nil {
			c.Fatalf("error in SetUpSuite: %v", err)
		}
	}
	err := driver.Initialize(time.Second * et.InitTimeout)
	if err != nil {
		c.Fatalf("error in SetUpSuite: %v", err)
	}
	if !existingServer {
		//give it some time if we started it
		time.Sleep(time.Second * 1)
		driver.isUp()
	}
}

//TearDownSuite Run once after all tests or benchmarks have finished running.
func (et *ElasticTest) TearDownSuite(c *gocheck.C) {
	log.Print("ElasticTest TearDownSuite called")

	et.stop()
}

func (et *ElasticTest) SetUpTest(c *gocheck.C) {
	log.Print("ElasticTest SetUpTest called")
	err := et.driver.deleteIndex()
	c.Assert(err, gocheck.IsNil)
	err = et.driver.Initialize(time.Second * et.InitTimeout)
	c.Assert(err, gocheck.IsNil)
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

	command := []string{
		elasticDir + "/bin/elasticsearch",
		"-f",
		fmt.Sprintf("-Des.http.port=%v", port),
	}

	conf := fmt.Sprintf("cluster.name: %v", rand.Int())
	err := ioutil.WriteFile(elasticDir+"/config/elasticsearch.yml", []byte(conf), 0644)
	if err != nil {
		return nil, err
	}

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

func ensureElasticJar(runDir string) string {
	_, err := exec.LookPath("java")
	if err != nil {
		log.Fatal("Can't find java in path")
	}
	gz := fmt.Sprintf("elasticsearch-%s.tar.gz", esVersion)
	gzPath := fmt.Sprintf("/tmp/%s", gz)

	path := fmt.Sprintf("%s/elasticsearch-%s", runDir, esVersion)

	commands := [][]string{}

	log.Printf("checking tar %s exists", gzPath)
	if _, err = os.Stat(gzPath); err != nil {
		url := fmt.Sprintf("https://download.elasticsearch.org/elasticsearch/elasticsearch/%s", gz)
		commands = append(commands, []string{"curl", "-O", url})
		commands = append(commands, []string{"mv", gz, gzPath})
	}
	commands = append(commands, []string{"tar", "-xvzf", gzPath, "-C", runDir + "/."})

	for _, cmd := range commands {
		log.Printf("exec: %v", cmd)
		err = exec.Command(cmd[0], cmd[1:]...).Run()
		if err != nil {
			log.Fatalf("could not execute %v: %v", strings.Join(cmd, " "), err)
		}
	}
	return path
}
