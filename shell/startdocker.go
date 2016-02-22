// Copyright 2016 The Serviced Authors.
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

package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

var ErrShellDisabled = errors.New("shell has been disabled for this service")

type ProcessConfig struct {
	ServiceID   string
	IsTTY       bool
	SaveAs      string
	Envv        []string
	Mount       []string
	Command     string
	LogToStderr bool // log the command output for stderr
	LogStash    struct {
		Enable        bool          //enable log stash
		SettleTime    time.Duration //how long to wait for log stash to flush logs before exiting, ex. 1s
		IdleFlushTime time.Duration //interval log stash flushes its buffer, ex 1ms
	}
}

func StartDocker(cfg *ProcessConfig, dockerRegistry, port, controller string, uiport string) (*exec.Cmd, error) {
	var svc service.Service

	// Create a control center client to look up the service
	cp, err := node.NewControlClient(port)
	if err != nil {
		glog.Errorf("could not create a control center client %v", err)
		return nil, err
	}
	glog.Infof("Connected to the control center at port %s", port)

	if err := cp.GetService(cfg.ServiceID, &svc); err != nil {
		glog.Errorf("unable to find service %s", cfg.ServiceID)
		return nil, err
	}
	if svc.DisableShell {
		glog.Errorf("Could not start shell for service %s (%s): %s", svc.Name, svc.ID, ErrShellDisabled)
		return nil, ErrShellDisabled
	}
	// make sure docker image is present
	imageID, err := commons.ParseImageID(svc.ImageID)
	if err != nil {
		glog.Errorf("Could not parse image %s: %s", svc.ImageID, err)
		return nil, err
	}
	image := (&registry.Image{
		Library: imageID.User,
		Repo:    imageID.Repo,
		Tag:     imageID.Tag,
	}).String()
	image = dockerRegistry + "/" + image
	glog.Infof("Getting image %s", image)
	if _, err = docker.FindImage(image, false); err != nil {
		if docker.IsImageNotFound(err) {
			if err := docker.PullImage(image); err != nil {
				glog.Errorf("unable to pull image %s: %s", image, err)
				return nil, err
			}
		} else {
			glog.Errorf("unable to inspect image %s: %s", image, err)
			return nil, err
		}
	}

	dir, binary := filepath.Split(controller)
	servicedVolume := fmt.Sprintf("%s:/serviced", dir)

	// bind mount the pwd
	dir, err = os.Getwd()
	pwdVolume := fmt.Sprintf("%s:/mnt/pwd", dir)

	// get the shell command
	shellcmd := cfg.Command
	if cfg.Command == "" {
		shellcmd = "su -"
	}

	// get the serviced command
	svcdcmd := fmt.Sprintf("/serviced/%s", binary)

	// get the proxy command
	proxycmd := []string{
		svcdcmd,
		fmt.Sprintf("--logtostderr=%t", cfg.LogToStderr),
		"--autorestart=false",
		"--disable-metric-forwarding",
		fmt.Sprintf("--logstash=%t", cfg.LogStash.Enable),
		fmt.Sprintf("--logstash-idle-flush-time=%s", cfg.LogStash.IdleFlushTime),
		fmt.Sprintf("--logstash-settle-time=%s", cfg.LogStash.SettleTime),
		svc.ID,
		"0",
		shellcmd,
	}

	// get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		glog.Errorf("Docker not found: %v", err)
		return nil, err
	}
	argv := []string{"run", "-v", servicedVolume, "-v", pwdVolume, "-v", utils.ResourcesDir() + ":" + "/usr/local/serviced/resources", "-u", "root", "-w", "/"}
	for _, mount := range cfg.Mount {
		hostPath, containerPath, err := parseMountArg(mount)
		if err != nil {
			return nil, err
		}
		argv = append(argv, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	argv = append(argv, cfg.Envv...)

	if cfg.SaveAs != "" {
		argv = append(argv, fmt.Sprintf("--name=%s", cfg.SaveAs))
	} else {
		argv = append(argv, "--rm")
	}

	if cfg.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	// set the systemuser and password
	unused := 0
	systemUser := user.User{}
	err = cp.GetSystemUser(unused, &systemUser)
	if err != nil {
		glog.Errorf("Unable to get system user account for client %s", err)
	}
	argv = append(argv, "-e", fmt.Sprintf("CONTROLPLANE_SYSTEM_USER=%s ", systemUser.Name))
	argv = append(argv, "-e", fmt.Sprintf("CONTROLPLANE_SYSTEM_PASSWORD=%s ", systemUser.Password))
	argv = append(argv, "-e", fmt.Sprintf("SERVICED_NOREGISTRY=%s", os.Getenv("SERVICED_NOREGISTRY")))
	argv = append(argv, "-e", fmt.Sprintf("SERVICED_IS_SERVICE_SHELL=true"))
	argv = append(argv, "-e", fmt.Sprintf("SERVICED_SERVICE_IMAGE=%s", image))
	argv = append(argv, "-e", fmt.Sprintf("SERVICED_UI_PORT=%s", strings.Split(uiport, ":")[1]))

	argv = append(argv, image)
	argv = append(argv, proxycmd...)

	// wait for the DFS to be ready in order to start container on the latest image
	glog.Infof("Acquiring image from the dfs...")
	if err := cp.ReadyDFS(svc.ID, new(int)); err != nil {
		glog.Errorf("Could not ready dfs: %s", err)
		return nil, err
	}
	glog.Infof("Acquired!  Starting shell")

	glog.V(1).Infof("command: docker %+v", argv)
	return exec.Command(docker, argv...), nil
}

func parseMountArg(arg string) (hostPath, containerPath string, err error) {
	splitMount := strings.Split(arg, ",")
	hostPath = splitMount[0]
	if len(splitMount) > 1 {
		containerPath = splitMount[1]
	} else {
		containerPath = hostPath
	}
	return

}
