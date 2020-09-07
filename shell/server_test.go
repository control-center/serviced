// Copyright 2019 The Serviced Authors.
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

// +build unit

package shell

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
)

type osMock struct {
	wd string
	env map[string]string
}

func (m osMock) Getwd() (string, error) {
	return m.wd, nil
}

func (m osMock) Getenv(key string) string {
	return m.env[key]
}

func makeOsMock(tz string) osMock {
	mock := osMock{
		wd: "/home/zenoss",
		env: map[string]string{
			"SERVICED_NOREGISTRY": "1",
		},
	}
	if len(tz) > 0 {
		mock.env["TZ"] = tz
	}
	return mock
}

func makeService() service.Service {
	return service.Service{
		ID: "abcd-1234",
		Name: "Abcd",
		ImageID: "imageAbcd",
		DisableShell: false,
	}
}

func makeProcessConfig(isTTY bool, saveAs string, command string) ProcessConfig {
	config := ProcessConfig{
		IsTTY: isTTY,
		Mount: []string{
			"/home/dir/source,/mnt/src",
			"/opt/zenoss",
		},
		LogToStderr: true,
		LogStash: struct {
			Enable bool
			SettleTime time.Duration
		}{
			Enable: true,
			SettleTime: 1000000,
		},
	}
	if len(saveAs) > 0 {
		config.SaveAs = saveAs
	}
	if len(command) > 0 {
		config.Command = command
	}
	return config
}

func makeExpectedResult(image string, o *osMock, s *service.Service, c *ProcessConfig) []string {
	volumeCurrentDir := fmt.Sprintf("%s:/mnt/pwd", o.wd)
	volumeResources := fmt.Sprintf(
		"%s:%s", utils.ResourcesDir(), utils.RESOURCES_CONTAINER_DIRECTORY,
	)
	servicedVersion := fmt.Sprintf("SERVICED_VERSION=%s ", servicedversion.Version)
	servicedNoRegistry := fmt.Sprintf("SERVICED_NOREGISTRY=%s", o.env["SERVICED_NOREGISTRY"])
	servicedServiceImage := fmt.Sprintf("SERVICED_SERVICE_IMAGE=%s", image)

	expected := []string{
		"run", "-u", "root", "-w", "/",
		"-v", "/opt/serviced/bin/:/serviced",
		"-v", volumeCurrentDir,
		"-v", volumeResources,
		"-v", "/home/dir/source:/mnt/src",
		"-v", "/opt/zenoss:/opt/zenoss",
	}

	if len(c.SaveAs) > 0 {
		expected = append(expected, "--name=imageName")
	} else {
		expected = append(expected, "--rm")
	}

	if c.IsTTY {
		expected = append(expected, "-i", "-t")
	}

	expected = append(
		expected,
		"-e", servicedVersion,
		"-e", servicedNoRegistry,
		"-e", "SERVICED_IS_SERVICE_SHELL=true",
		"-e", servicedServiceImage,
		"-e", "SERVICED_UI_PORT=443",
		"-e", "SERVICED_ZOOKEEPER_ACL_USER=",
		"-e", "SERVICED_ZOOKEEPER_ACL_PASSWD=",
	)

	val, ok := o.env["TZ"]
	if ok {
		expected = append(expected, "-e", fmt.Sprintf("TZ=%s", val))
	}

	expected = append(
		expected,
		image,
		"/serviced/serviced-controller",
		"--logtostderr=true",
		"--autorestart=false",
		"--disable-metric-forwarding",
		"--logstash=true",
		"--logstash-settle-time=1ms",
		"abcd-1234",
		"0",
	)

	if len(c.Command) > 0 {
		expected = append(expected, c.Command)
	} else {
		expected = append(expected, "su -")
	}

	return expected
}

func TestBuildDockerArgs(t *testing.T) {
	controller := "/opt/serviced/bin/serviced-controller"
	docker := "/usr/bin/docker"
	image := "baseImage"

	svc := makeService()

	cases := []struct{
		id string
		mock osMock
		cfg ProcessConfig
	}{
		{"Case0", makeOsMock(""), makeProcessConfig(true, "imageName", "bash")},
		{"Case1", makeOsMock(""), makeProcessConfig(true, "", "bash")},
		{"Case2", makeOsMock(""), makeProcessConfig(false, "", "bash")},
		{"Case3", makeOsMock(""), makeProcessConfig(false, "", "")},
		{"Case4", makeOsMock("TZ"), makeProcessConfig(false, "", "")},
		{"Case5", makeOsMock("TZ"), makeProcessConfig(true, "", "")},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			expected := makeExpectedResult(image, &tc.mock, &svc, &tc.cfg)
			actual := buildDockerArgs(tc.mock, &svc, &tc.cfg, controller, docker, image)

			assert := assert.New(t)
			assert.NotNil(actual)
			assert.Equal(expected, actual)
		})
	}
}
