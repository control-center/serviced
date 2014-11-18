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

package web

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/zenoss/glog"
)

func newAccessLogger(varPath string, shutdown <-chan interface{}) (*log.Logger, error) {
	accessLogFile, err := os.OpenFile("/var/log/serviced.access.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	err = initRotation(varPath)
	if err != nil {
		return nil, err
	}
	rotate(varPath)
	go rotateLoop(varPath, shutdown)
	return log.New(accessLogFile, "", log.LstdFlags), nil
}

func initRotation(varPath string) error {
	conf := []byte(`compress
                    /var/log/serviced.access.log {
                        rotate 5
                        weekly
                        copytruncate
                    }`)
	confPath := path.Join(varPath, "logrotate.conf")
	return ioutil.WriteFile(confPath, conf, 0666)
}

func rotate(varPath string) {
	confPath := path.Join(varPath, "logrotate.conf")
	cmd := exec.Command("sh", "-c", fmt.Sprintf("logrotate %s", confPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("error executing logrotate: %v\n    %s", err, output)
	}
}

func rotateLoop(varPath string, shutdown <-chan interface{}) {
	for {
		select {
		case <-shutdown:
			return
		case <-time.After(time.Hour * 24):
			rotate(varPath)
		}
	}
}
