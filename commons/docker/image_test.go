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

// +build integration,!quick

package docker

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

const (
	bogusimage = "iam/bogus:zenoss"
	rawbase    = "busybox:latest"
	basetag    = "localhost:5000/testimageapi/busybox:latest"
	imptag     = "testimageapi/imported:latest"
	snaptag    = "localhost:5000/testimageapi/busybox:snapshot"
	victim     = "localhost:5000/testimageapi/busybox:victim"
	bareimg    = "localhost:5000/testimageapi/busyb"
)

func TestImageAPI(t *testing.T) {
	TestingT(t)
}

type ImageTestSuite struct {
	regowner bool
	regid    string
}

var _ = Suite(&ImageTestSuite{})

func (s *ImageTestSuite) SetUpSuite(c *C) {
	regid, ok := isRegistryRunning(c)
	if ok {
		s.regid = regid
	} else {
		cmd := []string{
			"docker", "run",
			"-d",
			"-p", "5000:5000",
			// Workaround from https://github.com/docker/docker-registry/issues/796
			"-e", "GUNICORN_OPTS=[--preload]",
			"registry",
		}
		regid, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			panic("can't start registry")
		}

		s.regid = strings.Trim(string(regid), "\n")
		s.regowner = true

		regup := make(chan struct{})
		timeout := make(chan struct{})

		go func(timeout, regup chan struct{}) {
		WaitForRegistryConnection:
			for {
				select {
				case <-timeout:
					panic("start registry timed out: can't connect")
				default:
					if _, err := net.Dial("tcp", ":5000"); err != nil {
						time.Sleep(1 * time.Second)
						continue WaitForRegistryConnection
					}
					break WaitForRegistryConnection
				}
			}

		WaitForRegistryPing:
			for {
				select {
				case <-timeout:
					panic("start registry timed out: can't ping it")
				default:
					_, err := http.Get("http://localhost:5000/v1/_ping")
					if err != nil {
						time.Sleep(1 * time.Second)
						continue WaitForRegistryPing
					}
					regup <- struct{}{}
					break WaitForRegistryPing
				}
			}
		}(timeout, regup)

		select {
		case <-time.After(60 * time.Second):
			timeout <- struct{}{}
		case <-regup:
			// Give the registry a couple of seconds to complete startup to make
			//     make sure that it's really ready. If the registry fails after
			// 	   it's initial startup, this delay should catch that case.
			time.Sleep(2 * time.Second)
			if _, ok := isRegistryRunning(c); !ok {
				panic("could not start registry")
			}
			break
		}
	}

	exportcmd := exec.Command("docker", "export", s.regid)
	stdout, err := exportcmd.StdoutPipe()
	if err != nil {
		panic(fmt.Errorf("can't create pipe for docker export: %v", err))
	}

	if err = exportcmd.Start(); err != nil {
		panic(fmt.Errorf("can't start docker export command: %v", err))
	}

	f, err := os.Create("/tmp/regexp.tar")
	if err != nil {
		panic(fmt.Errorf("can't create /tmp/regexp.tar: %v", err))
	}

	io.Copy(f, stdout)

	if err = exportcmd.Wait(); err != nil {
		panic(fmt.Errorf("waiting for docker export to finish failed: %v", err))
	}
}

func (s *ImageTestSuite) TearDownSuite(c *C) {
	var cmd []string

	if s.regowner {
		cmd = []string{"docker", "kill", s.regid}
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			panic("can't kill the registry")
		}
	}

	cmd = []string{"docker", "rm", s.regid}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", basetag}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", imptag}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", snaptag}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", rawbase}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", bareimg}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"rm", "/tmp/regexp.tar"}
	exec.Command(cmd[0], cmd[1:]...).Run()
}

func (s *ImageTestSuite) TestFindImage(c *C) {
	_, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v", rawbase, err)
	}
}

func (s *ImageTestSuite) TestFindNonexistentImage(c *C) {
	_, err := FindImage(bogusimage, false)
	if err == nil {
		c.Fatalf("should not be able to find %s", bogusimage)
	}
}

func (s *ImageTestSuite) TestTagImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v ", rawbase, err)
	}

	ti, err := img.Tag(basetag, true)
	if err != nil {
		c.Fatalf("can't tag %s as %s: %v", rawbase, basetag, err)
	}

	_, err = FindImage(ti.ID.String(), false)
	if err != nil {
		c.Fatalf("can't find %s: %v", ti.ID, err)
	}
}

func (s *ImageTestSuite) TestInspectImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v ", rawbase, err)
	}

	imgInfo, err := InspectImage(img.UUID)
	if err != nil {
		c.Fatalf("can't inspect %s: %v ", rawbase, err)
	}
	// Minimal validation of image
	if imgInfo.ID == "" {
		c.Fatalf("Invalid docker image structure %+v", imgInfo)
	}
}

func (s *ImageTestSuite) TestDoubleTagImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v", rawbase, err)
	}

	bt, err := img.Tag(basetag, true)
	if err != nil {
		c.Fatalf("can't tag %s as %s: %v", rawbase, basetag, err)
	}

	st, err := img.Tag(snaptag, true)
	if err != nil {
		c.Fatalf("can't tag %s as %s: %v", bt.ID.String(), snaptag, err)
	}

	_, err = FindImage(st.ID.String(), false)
	if err != nil {
		c.Fatalf("can't find %s: %v", snaptag, err)
	}
}

func (s *ImageTestSuite) TestBareTag(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v", rawbase, err)
	}

	bi, err := img.Tag(bareimg, true)
	if err != nil {
		c.Fatalf("can't tag %s as %s: %v", rawbase, bareimg, err)
	}

	_, err = FindImage(bi.ID.String(), false)
	if err != nil {
		c.Fatalf("can't find %s: %v", bi.ID, err)
	}
}

func (s *ImageTestSuite) TestDeleteImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v", rawbase, err)
	}

	ti, err := img.Tag(victim, true)
	if err != nil {
		c.Fatalf("can't tag %s as %s: %v", rawbase, victim, err)
	}

	if err = ti.Delete(); err != nil {
		c.Fatalf("can't delete %s: %v", ti.ID.String(), err)
	}

	img, err = FindImage(ti.ID.String(), false)
	if img != nil {
		c.Fatal("should not have found: ", ti.ID.String())
	}
}

/*
func (s *ImageTestSuite) TestFindThruLocalRepository(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Fatalf("can't find %s: %v", rawbase, err)
	}

	ti, err := img.Tag(basetag)
	if err != nil {
		c.Fatalf("can't tag %s as %s: %v", rawbase, basetag, err)
	}

	if err = ti.Delete(); err != nil {
		c.Fatalf("can't delete %s: %v", ti.ID.String(), err)
	}

	_, err = FindImage(basetag, false)
	if err == nil {
		c.Fatalf("%s should not be in the repo", basetag)
	}

	_, err = FindImage(basetag, true)
	if err != nil {
		c.Fatalf("can't find %s in local registry: %v", basetag, err)
	}
}
*/

func (s *ImageTestSuite) TestImportImage(c *C) {
	_, err := FindImage(imptag, false)
	if err == nil {
		c.Fatalf("%s should not be in the repo", imptag)
	}

	if err = ImportImage(imptag, "/tmp/regexp.tar"); err != nil {
		c.Fatalf("can't import %s: %v", imptag, err)
	}

	_, err = FindImage(imptag, false)
	if err != nil {
		c.Fatalf("can't find imported image (%s): %v", imptag, err)
	}
}

func isRegistryRunning(c *C) (string, bool) {
	var regid string
	var result bool

	dockerps := exec.Command("docker", "ps")
	stdout, err := dockerps.StdoutPipe()
	if err != nil {
		c.Fatal("can't redirect docker ps stdout: ", err)
	}

	if err = dockerps.Start(); err != nil {
		c.Fatal("can't start docker ps command: ", err)
	}

	re := regexp.MustCompile("registry")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		l := scanner.Text()
		if len(re.FindString(strings.ToLower(l))) > 0 {
			fs := strings.Fields(l)
			regid = fs[0]
			result = true
			goto WaitForCompletion
		}
	}

WaitForCompletion:
	if err = dockerps.Wait(); err != nil {
		c.Fatal("waiting for docker ps completion failed: ", err)
	}
	return regid, result
}
