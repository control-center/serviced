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

package shell

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/control-center/go-socket.io"
	"github.com/zenoss/glog"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
)

var empty interface{}

var ErrShellDisabled = errors.New("shell has been disabled for this service")

const (
	PROCESSKEY      string = "process"
	MAXBUFFER       int    = 8192
	DOCKER_ENDPOINT        = "unix:///var/run/docker.sock"
)

var webroot string

func init() {
	servicedHome := os.Getenv("SERVICED_HOME")
	if len(servicedHome) > 0 {
		webroot = servicedHome + "/share/shell/static"
	}
}

func staticRoot() string {
	if len(webroot) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		return path.Join(path.Dir(path.Dir(filename)), "shell", "static")
	}
	return webroot
}

func NewProcessForwarderServer(addr string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Forwarder{addr: addr},
	}
	server.sio.On("connect", server.onConnect)
	server.sio.On("disconnect", onForwarderDisconnect)
	// BUG: ZEN-10320
	// server.Handle("/", http.FileServer(http.Dir(staticRoot())))
	return server
}

func NewProcessExecutorServer(port, dockerRegistry, controllerBinary, uiport string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Executor{port: port, dockerRegistry: dockerRegistry, controllerBinary: controllerBinary, uiport: uiport},
	}
	server.sio.On("connect", server.onConnect)
	server.sio.On("disconnect", onExecutorDisconnect)
	server.sio.On("process", server.onProcess)
	// BUG: ZEN-10320
	// server.Handle("/", http.FileServer(http.Dir(staticRoot())))
	return server
}

func (p *ProcessServer) Handle(pattern string, handler http.Handler) error {
	p.sio.Handle(pattern, handler)
	return nil
}

func (p *ProcessServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.sio.ServeHTTP(w, r)
}

func (s *ProcessServer) onProcess(ns *socketio.NameSpace, cfg *ProcessConfig) {
	// Kick it off
	glog.Infof("Received process packet")
	proc := s.actor.Exec(cfg)
	ns.Session.Values[PROCESSKEY] = proc

	// Wire up output
	go proc.ReadRequest(ns)
	go proc.WriteResponse(ns)
}

func (s *ProcessServer) onConnect(ns *socketio.NameSpace) {
	glog.Infof("Waiting for process packet")
}

func onForwarderDisconnect(ns *socketio.NameSpace) {
	// ns.Session.Values[PROCESSKEY].(*ProcessInstance)
}

func onExecutorDisconnect(ns *socketio.NameSpace) {
	inst := ns.Session.Values[PROCESSKEY].(*ProcessInstance)
	inst.Disconnect()
}

func (p *ProcessInstance) Disconnect() {
	p.disconnected = true
	if p.Stdin != nil {
		close(p.Stdin)
		p.Stdin = nil
	}
}

func (p *ProcessInstance) Close() {
	p.closed = true
	if p.Stdout != nil {
		close(p.Stdout)
		p.Stdout = nil
	}
	if p.Stderr != nil {
		close(p.Stderr)
		p.Stderr = nil
	}
	// do not close Result channel
}

func (p *ProcessInstance) ReadRequest(ns *socketio.NameSpace) {
	ns.On("signal", func(n *socketio.NameSpace, signal int) {
		glog.V(4).Infof("received signal %d", signal)
		if p.disconnected {
			glog.Warning("disconnected; cannot send signal: %s", signal)
		}
	})

	ns.On("stdin", func(n *socketio.NameSpace, stdin string) {
		glog.V(4).Infof("Received stdin: %s", stdin)
		if p.disconnected {
			glog.Warning("disconnected; cannot send stdin: %s", stdin)
		} else {
			for _, b := range []byte(stdin) {
				p.Stdin <- b
			}
		}
	})

	glog.V(0).Info("Hooked up incoming events!")
}

func (p *ProcessInstance) WriteRequest(ns *socketio.NameSpace) {
	glog.V(0).Info("Hooking up input channels!")
	for p.Stdin != nil {
		select {
		case m, ok := <-p.Stdin:
			if !ok {
				p.Stdin = nil
			} else {
				ns.Emit("stdin", m)
			}
		}
	}
}

func (p *ProcessInstance) ReadResponse(ns *socketio.NameSpace) {
	ns.On("stdout", func(n *socketio.NameSpace, stdout string) {
		glog.V(4).Infof("Process received stdout: %s", stdout)
		if p.closed {
			glog.Warning("connection closed; cannot write stdout: %s", stdout)
		} else {
			for _, b := range []byte(stdout) {
				p.Stdout <- b
			}
		}
	})

	ns.On("stderr", func(n *socketio.NameSpace, stderr string) {
		glog.V(4).Infof("Process received stderr: %s", stderr)
		if p.closed {
			glog.Warning("connection closed; cannot write stderr: %s", stderr)
		} else {
			for _, b := range []byte(stderr) {
				p.Stderr <- b
			}
		}
	})

	ns.On("result", func(n *socketio.NameSpace, result Result) {
		glog.V(0).Infof("Process received result: %s", result)
		p.Result <- result
	})
	glog.V(0).Info("Hooked up outgoing events!")
}

func (p *ProcessInstance) WriteResponse(ns *socketio.NameSpace) {
	glog.V(0).Info("Hooking up output channels!")

	for p.Stdout != nil || p.Stderr != nil {
		select {
		case m, ok := <-p.Stdout:
			if !ok {
				p.Stdout = nil
			} else {
				glog.V(2).Infof("Emitting stdout: %3s %c", m, m)
				ns.Emit("stdout", m)
			}
		case m, ok := <-p.Stderr:
			if !ok {
				p.Stderr = nil
			} else {
				glog.V(2).Infof("Emitting stderr: %3s %c", m, m)
				ns.Emit("stderr", m)
			}
		}
	}
	ns.Emit("result", <-p.Result)
	p.Disconnect()
}

func (f *Forwarder) Exec(cfg *ProcessConfig) *ProcessInstance {
	// TODO: make me more extensible
	urlAddr, err := url.Parse(f.addr)
	if err != nil {
		glog.Fatalf("Not a valid path: %s (%v)", f.addr, err)
	}

	host := fmt.Sprintf("http://%s:50000/", strings.Split(urlAddr.Host, ":")[0])

	// Dial the remote ProcessServer
	client, err := socketio.Dial(host)

	if err != nil {
		glog.Fatalf("Unable to contact remote process server: %v", err)
	}

	client.On("connect", func(ns *socketio.NameSpace) {
		if ns.Session.Values[PROCESSKEY] == nil {
			ns.Emit("process", cfg)
		} else {
			glog.Fatalf("Trying to connect to a stale process!")
		}
	})

	ns := client.Of("")
	proc := &ProcessInstance{
		Stdin:  make(chan byte, 1024),
		Stdout: make(chan byte, 1024),
		Stderr: make(chan byte, 1024),
		Result: make(chan Result),
	}

	client.On("disconnect", func(ns *socketio.NameSpace) {
		glog.Infof("Disconnected!")
		proc.Disconnect()
		proc.Close()
	})

	go proc.ReadResponse(ns)
	go proc.WriteRequest(ns)

	go client.Run()

	return proc
}

func (f *Forwarder) onDisconnect(ns *socketio.NameSpace) {
	ns.Session.Values[PROCESSKEY] = nil
}

func (e *Executor) Exec(cfg *ProcessConfig) (p *ProcessInstance) {
	p = &ProcessInstance{
		Stdin:  make(chan byte, 1024),
		Stdout: make(chan byte, 1024),
		Stderr: make(chan byte, 1024),
		Result: make(chan Result, 2),
	}

	cmd, err := StartDocker(cfg, e.dockerRegistry, e.port, e.controllerBinary, e.uiport)
	if err != nil {
		p.Result <- Result{0, err.Error(), ABNORMAL}
		return
	}

	cmd.Stdin = ShellReader{p.Stdin}
	cmd.Stdout = ShellWriter{p.Stdout}
	cmd.Stderr = ShellWriter{p.Stderr}

	go func() {
		defer p.Close()
		err := cmd.Run()
		if exitcode, ok := utils.GetExitStatus(err); !ok {
			p.Result <- Result{exitcode, err.Error(), ABNORMAL}
		} else if exitcode == 0 {
			p.Result <- Result{exitcode, "", NORMAL}
		} else {
			p.Result <- Result{exitcode, err.Error(), NORMAL}
		}
	}()

	return
}

func (e *Executor) onDisconnect(ns *socketio.NameSpace) {
	inst := ns.Session.Values[PROCESSKEY].(*ProcessInstance)
	inst.Disconnect()
	ns.Session.Values[PROCESSKEY] = nil
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
