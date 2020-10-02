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
	"github.com/control-center/serviced/config"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-center/go-socket.io"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/logging"
	worker "github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	log "github.com/Sirupsen/logrus"
)

var (
	empty interface{}

	// ErrShellDisabled - Shell disabled error message
	ErrShellDisabled = errors.New("shell has been disabled for this service")

	plog = logging.PackageLogger()
)

const (
	PROCESSKEY string = "process"
)

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

// NewProcessExecutorServer - Create and return a processServer instance
func NewProcessExecutorServer(masterAddress, agentAddress, dockerRegistry, controllerBinary string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Executor{masterAddress: masterAddress, agentAddress: agentAddress, dockerRegistry: dockerRegistry, controllerBinary: controllerBinary},
	}
	server.sio.On("connect", server.onConnect)
	server.sio.On("disconnect", onExecutorDisconnect)
	server.sio.On("process", server.onProcess)
	// BUG: ZEN-10320
	// server.Handle("/", http.FileServer(http.Dir(staticRoot())))
	return server
}

func (s *ProcessServer) Handle(pattern string, handler http.Handler) error {
	s.sio.Handle(pattern, handler)
	return nil
}

func (s *ProcessServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.sio.ServeHTTP(w, r)
}

func (s *ProcessServer) onProcess(ns *socketio.NameSpace, cfg *ProcessConfig) {
	// Kick it off
	plog.Info("Received process packet")
	proc := s.actor.Exec(cfg)
	ns.Session.Values[PROCESSKEY] = proc

	// Wire up output
	go proc.ReadRequest(ns)
	go proc.WriteResponse(ns)
}

func (s *ProcessServer) onConnect(ns *socketio.NameSpace) {
	plog.Info("Waiting for process packet")
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
		if p.disconnected {
			plog.WithField("signal", signal).Warning("disconnected; cannot send signal")
		}
	})

	ns.On("stdin", func(n *socketio.NameSpace, stdin string) {
		if p.disconnected {
			plog.WithField("value", stdin).Warning("disconnected; cannot send string from stdin")
		} else {
			for _, b := range []byte(stdin) {
				p.Stdin <- b
			}
		}
	})

	plog.Debug("Hooked up incoming events")
}

func (p *ProcessInstance) WriteRequest(ns *socketio.NameSpace) {
	plog.Debug("Hooking up input channels!")
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
		if p.closed {
			plog.WithField("value", stdout).Warning("connection closed; cannot send string to stdout")
		} else {
			for _, b := range []byte(stdout) {
				p.Stdout <- b
			}
		}
	})

	ns.On("stderr", func(n *socketio.NameSpace, stderr string) {
		if p.closed {
			plog.WithField("value", stderr).Warning("connection closed; cannot send string to stderr")
		} else {
			for _, b := range []byte(stderr) {
				p.Stderr <- b
			}
		}
	})

	ns.On("result", func(n *socketio.NameSpace, result Result) {
		plog.WithField("result", result).Debug("Process received result")
		p.Result <- result
	})
	plog.Debug("Hooked up outgoing events")
}

func (p *ProcessInstance) WriteResponse(ns *socketio.NameSpace) {
	plog.Debug("Hooking up output channels")

	for p.Stdout != nil || p.Stderr != nil {
		select {
		case m, ok := <-p.Stdout:
			if !ok {
				p.Stdout = nil
			} else {
				ns.Emit("stdout", m)
			}
		case m, ok := <-p.Stderr:
			if !ok {
				p.Stderr = nil
			} else {
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
		plog.WithError(err).WithField("url", f.addr).Fatal("Not a valid URL")
	}

	host := fmt.Sprintf("http://%s:50000/", strings.Split(urlAddr.Host, ":")[0])

	// Dial the remote ProcessServer
	client, err := socketio.Dial(host)

	if err != nil {
		plog.WithError(err).WithField("host", host).Fatal("Unable to contact remote process server")
	}

	client.On("connect", func(ns *socketio.NameSpace) {
		if ns.Session.Values[PROCESSKEY] == nil {
			ns.Emit("process", cfg)
		} else {
			plog.WithError(err).Fatal("Trying to connect to a stale process!")
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
		plog.Info("Disconnected")
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

	//Control center is always available in a container on port 443 regardless of the ui port
	cmd, err := StartDocker(cfg, e.masterAddress, e.agentAddress, e.dockerRegistry, e.controllerBinary)
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

// StartDocker - Start a docker container
func StartDocker(cfg *ProcessConfig, masterAddress, workerAddress, dockerRegistry, controller string) (*exec.Cmd, error) {
	logger := plog.WithFields(log.Fields{
		"masteraddress":   masterAddress,
		"delegateaddress": workerAddress,
		"serviceid":       cfg.ServiceID,
	})

	// look up the service on the master
	masterClient, err := master.NewClient(masterAddress)
	if err != nil {
		logger.WithError(err).Error("Could not connect to the master server")
		return nil, err
	}
	defer masterClient.Close()
	logger.Info("Connected to master")

	svc, err := masterClient.GetService(cfg.ServiceID)
	if err != nil {
		logger.WithError(err).Error("Could not get services")
		return nil, err
	}

	logger = logger.WithField("servicename", svc.Name)

	// can we create a shell for this service?
	if svc.DisableShell {
		logger.Warning("Shell commands are disabled for service")
		return nil, ErrShellDisabled
	} else if svc.ImageID == "" {
		logger.Warning("No image is configured for service")
		return nil, ErrShellDisabled
	}

	// lets make sure the image exists locally and is the latest
	workerClient, err := worker.NewClient(workerAddress)
	if err != nil {
		logger.WithError(err).Error("Could not connect to the delegate server")
		return nil, err
	}
	defer workerClient.Close()

	logger = logger.WithField("imageid", svc.ImageID)

	logger.Info("Connected to delegate; pulling image")
	image, err := workerClient.PullImage(dockerRegistry, svc.ImageID, time.Minute)
	if err != nil {
		logger.WithError(err).Error("Could not pull image")
		return nil, err
	}
	logger.Info("Pulled image, setting up shell")

	// get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		logger.WithError(err).Error("Docker not found")
		return nil, err
	}

	argv := buildDockerArgs(osWrapImpl{}, svc, cfg, controller, docker, image)
	logger.Debugf("command: docker %+v", argv)
	logger.Info("Shell initialized for service, starting")
	return exec.Command(docker, argv...), nil
}

// Wrap the use of os.Getwd and os.Getenv so that the output of those
// functions can be controlled for testing purposes.
type osWrap interface {
	Getwd() (string, error)
	Getenv(key string) string
}

type osWrapImpl struct {
}

func (osWrapImpl) Getwd() (string, error) {
	return os.Getwd()
}

func (osWrapImpl) Getenv(key string) string {
	return os.Getenv(key)
}

func buildDockerArgs(wrap osWrap, svc *service.Service, cfg *ProcessConfig, controller string, docker string, image string) []string {
	argv := []string{"run", "-u", "root", "-w", "/"}

	dir, binary := filepath.Split(controller)
	servicedVolume := fmt.Sprintf("%s:/serviced", dir)

	// bind mount the current directory
	dir, _ = wrap.Getwd()
	pwdVolume := fmt.Sprintf("%s:/mnt/pwd", dir)

	resourceBinding := fmt.Sprintf("%s:%s", utils.ResourcesDir(), utils.RESOURCES_CONTAINER_DIRECTORY)

	// add the mount volume arguments
	argv = append(
		argv,
		"-v", servicedVolume,
		"-v", pwdVolume,
		"-v", resourceBinding,
	)
	for _, mount := range cfg.Mount {
		hostPath, containerPath, _ := parseMountArg(mount)
		if len(hostPath) == 0 {
			continue
		}
		argv = append(argv, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	if cfg.SaveAs != "" {
		argv = append(argv, fmt.Sprintf("--name=%s", cfg.SaveAs))
	} else {
		argv = append(argv, "--rm")
	}

	if cfg.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	// Add the environment variables
	argv = append(
		argv,
		"-e", fmt.Sprintf("SERVICED_VERSION=%s ", servicedversion.Version),
		"-e", fmt.Sprintf("SERVICED_NOREGISTRY=%s", wrap.Getenv("SERVICED_NOREGISTRY")),
		"-e", "SERVICED_IS_SERVICE_SHELL=true",
		"-e", fmt.Sprintf("SERVICED_SERVICE_IMAGE=%s", image),
		//The SERVICED_UI_PORT environment variable is deprecated and services
		//should always use port 443 to contact serviced from inside a container
		"-e", "SERVICED_UI_PORT=443",
		"-e", fmt.Sprintf("SERVICED_ZOOKEEPER_ACL_USER=%s", config.GetOptions().ZkAclUser),
		"-e", fmt.Sprintf("SERVICED_ZOOKEEPER_ACL_PASSWD=%s", config.GetOptions().ZkAclPasswd),
	)
	tz := wrap.Getenv("TZ")
	if len(tz) > 0 {
		argv = append(argv, "-e", fmt.Sprintf("TZ=%s", tz))
	}

	argv = append(argv, image)

	// get the shell command
	shellcmd := cfg.Command
	if cfg.Command == "" {
		shellcmd = "su -"
	}

	// get the proxy command
	argv = append(
		argv,
		fmt.Sprintf("/serviced/%s", binary),
		fmt.Sprintf("--logtostderr=%t", cfg.LogToStderr),
		"--autorestart=false",
		"--disable-metric-forwarding",
		fmt.Sprintf("--logstash=%t", cfg.LogStash.Enable),
		fmt.Sprintf("--logstash-settle-time=%s", cfg.LogStash.SettleTime),
		svc.ID,
		"0",
		shellcmd,
	)

	return argv
}
