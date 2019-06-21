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

package isvcs

import (
	"github.com/Sirupsen/logrus"

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"	
)

var randomSource string

func init() {
	randomSource = "/dev/urandom"
}

// check if the given path is a directory
func isDir(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return stat.IsDir(), nil
	} else {
		if os.IsNotExist(err) {
			return false, nil
		}
	}
	return false, err
}

// generate a uuid
func uuid() string {
	f, _ := os.Open(randomSource)
	defer f.Close()
	b := make([]byte, 16)
	f.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Get the IP of the docker0 interface, which can be used to access the serviced API from inside the container.
// inspiration: the following is used for the same purpose during deploy/provision:
// ip addr show docker0 | grep inet | grep -v inet6 | awk '{print $2}' | awk -F / '{print $1}'
func getDockerIP() string {
	// Execute 'ip -4 -br addr show docker0'
	c1 := exec.Command("ip", "-4", "-br", "addr", "show", "docker0")
	var out bytes.Buffer
	c1.Stdout = &out
	err := c1.Run()
	if err != nil {
		log.WithField("command", c1).Infof("Error calling command: %s", err)
		return ""
	}
	outstr := out.String()
	// use a regex to extract the ip address from the result.
	// We're expecting something that looks like: ###.###.###.###/##
	// We use a capture group to exclude the trailing /##.
	re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)/\d+`)
	addr := re.FindStringSubmatch(outstr)
	if addr != nil && len(addr) > 1 {
		return addr[1]
	}
	log.WithFields(logrus.Fields{"match": addr, "output": outstr}).Info("Output was not as expected")
	return ""
}

type metric struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

func postDataToOpenTSDB(metrics []metric) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	url := "http://127.0.0.1:4242/api/put"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
