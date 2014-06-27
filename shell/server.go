package shell

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"syscall"

	"github.com/googollee/go-socket.io"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"

	"github.com/zenoss/serviced/commons/docker"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/user"
	"github.com/zenoss/serviced/node"
)

var empty interface{}

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

func NewProcessExecutorServer(port, dockerRegistry string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Executor{port: port, dockerRegistry: dockerRegistry},
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

	registry, err := docker.NewDockerRegistry(e.dockerRegistry)
	if err != nil {
		p.Result <- Result{0, err.Error(), ABNORMAL}
		return
	}
	dockerClient, err := dockerclient.NewClient(DOCKER_ENDPOINT)
	if err != nil {
		p.Result <- Result{0, err.Error(), ABNORMAL}
		return
	}
	cmd, err := StartDocker(registry, dockerClient, cfg, e.port)
	if err != nil {
		p.Result <- Result{0, err.Error(), ABNORMAL}
		return
	}

	cmd.Stdin = ShellReader{p.Stdin}
	cmd.Stdout = ShellWriter{p.Stdout}
	cmd.Stderr = ShellWriter{p.Stderr}

	go func() {
		defer p.Close()

		if err := cmd.Run(); err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					p.Result <- Result{status.ExitStatus(), err.Error(), NORMAL}
					return
				}
			}
			p.Result <- Result{0, err.Error(), ABNORMAL}
		} else {
			p.Result <- Result{0, "", NORMAL}
		}
	}()

	return
}

func (e *Executor) onDisconnect(ns *socketio.NameSpace) {
	inst := ns.Session.Values[PROCESSKEY].(*ProcessInstance)
	inst.Disconnect()
	ns.Session.Values[PROCESSKEY] = nil
}

func StartDocker(registry *docker.DockerRegistry, dockerClient *dockerclient.Client, cfg *ProcessConfig, port string) (*exec.Cmd, error) {
	var svc service.Service

	// Create a control plane client to look up the service
	cp, err := node.NewControlClient(port)
	if err != nil {
		glog.Errorf("could not create a control plane client %v", err)
		return nil, err
	}
	glog.Infof("Connected to the control plane at port %s", port)

	if err := cp.GetService(cfg.ServiceID, &svc); err != nil {
		glog.Errorf("unable to find service %s", cfg.ServiceID)
		return nil, err
	}

	// make sure docker image is present
	if _, err = docker.InspectImage(*registry, dockerClient, svc.ImageID); err != nil {
		glog.Errorf("unable to inspect image %s: %s", svc.ImageID, err)
		return nil, err
	}

	// bind mount on /serviced
	dir, bin, err := node.ExecPath()
	if err != nil {
		glog.Errorf("serviced not found: %s", err)
		return nil, err
	}
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
	svcdcmd := fmt.Sprintf("/serviced/%s", bin)

	// get the proxy command
	proxycmd := []string{
		svcdcmd,
		"--logtostderr=false",
		"service",
		"proxy",
		"--autorestart=false",
		"--logstash=false",
		svc.Id,
		"0",
		shellcmd,
	}

	// get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		glog.Errorf("Docker not found: %v", err)
		return nil, err
	}
	argv := []string{"run", "-v", servicedVolume, "-v", pwdVolume}
	argv = append(argv, cfg.Envv...)

	if cfg.SaveAs != "" {
		argv = append(argv, fmt.Sprintf("--name=%s", cfg.SaveAs))
	} else {
		argv = append(argv, "--rm")
	}

	if cfg.IsTTY {
		argv = append(argv, "-i", "-t")
	} else {
		argv = append(argv, "-t")
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

	argv = append(argv, svc.ImageID)
	argv = append(argv, proxycmd...)

	// wait for the DFS to be ready in order to start container on the latest image
	glog.Infof("Acquiring image from the dfs...")
	cp.ReadyDFS(false, nil)
	glog.Infof("Acquired!  Starting shell")

	glog.V(1).Infof("command: docker %+v", argv)
	return exec.Command(docker, argv...), nil
}
