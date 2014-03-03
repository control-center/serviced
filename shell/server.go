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

	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dfs"
)

var empty interface{}

const (
	PROCESSKEY string = "process"
	MAXBUFFER  int    = 8192
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

func NewProcessExecutorServer(port string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Executor{port: port},
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
	// Client disconnected, so kill the process
	inst.Signal <- int(syscall.SIGKILL)
	inst.closeIncoming()
}

func (p *ProcessInstance) Close() {
	p.closeIncoming()
	p.closeOutgoing()
}

func (p *ProcessInstance) closeIncoming() {
	glog.V(0).Infof("Closing incoming channels")
	if _, ok := <-p.Stdin; ok {
		close(p.Stdin)
	}
	if _, ok := <-p.Signal; ok {
		close(p.Signal)
	}
}

func (p *ProcessInstance) closeOutgoing() {
	glog.V(0).Infof("Closing outgoing channels")
	if _, ok := <-p.Stdout; ok {
		close(p.Stdout)
	}
	if _, ok := <-p.Stderr; ok {
		close(p.Stderr)
	}
	if _, ok := <-p.Result; ok {
		close(p.Result)
	}
}

func (p *ProcessInstance) ReadRequest(ns *socketio.NameSpace) {
	ns.On("signal", func(n *socketio.NameSpace, signal int) {
		glog.V(4).Infof("received signal %d", signal)
		p.Signal <- signal
	})

	ns.On("stdin", func(n *socketio.NameSpace, stdin string) {
		glog.V(4).Infof("Received stdin: %s", stdin)
		p.Stdin <- stdin
	})

	glog.V(0).Info("Hooked up incoming events!")
}

func (p *ProcessInstance) WriteRequest(ns *socketio.NameSpace) {
	glog.V(0).Info("Hooking up input channels!")
	for p.Stdin != nil || p.Signal != nil {
		select {
		case m, ok := <-p.Stdin:
			if !ok {
				p.Stdin = nil
			} else {
				ns.Emit("stdin", m)
			}
		case s, ok := <-p.Signal:
			if !ok {
				p.Signal = nil
			} else {
				ns.Emit("signal", s)
			}
		}
	}
}

func (p *ProcessInstance) ReadResponse(ns *socketio.NameSpace) {
	ns.On("stdout", func(n *socketio.NameSpace, stdout string) {
		glog.V(4).Infof("Process received stdout: %s", stdout)
		p.Stdout <- stdout
	})

	ns.On("stderr", func(n *socketio.NameSpace, stderr string) {
		glog.V(4).Infof("Process received stderr: %s", stderr)
		p.Stderr <- stderr
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
				glog.Infof("Emitting stdout: %s", m)
				ns.Emit("stdout", m)
			}
		case m, ok := <-p.Stderr:
			if !ok {
				p.Stderr = nil
			} else {
				glog.Infof("Emitting stderr: %s", m)
				ns.Emit("stderr", m)
			}
		}
	}
	ns.Emit("result", <-p.Result)
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
		Stdin:  make(chan string),
		Stdout: make(chan string),
		Stderr: make(chan string),
		Signal: make(chan int),
		Result: make(chan Result),
	}

	client.On("disconnect", func(ns *socketio.NameSpace) {
		glog.Infof("Disconnected!")
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

func (e *Executor) Exec(cfg *ProcessConfig) *ProcessInstance {
	return StartDocker(cfg, e.port)
}

func (e *Executor) onDisconnect(ns *socketio.NameSpace) {
	inst := ns.Session.Values[PROCESSKEY].(*ProcessInstance)
	// Client disconnected, so kill the process
	inst.Signal <- int(syscall.SIGKILL)
	inst.closeIncoming()
	ns.Session.Values[PROCESSKEY] = nil
}

func StartDocker(cfg *ProcessConfig, port string) *ProcessInstance {
	// Acquire and release the lock to start a container from the latest image
	dfs.Lock.Lock()
	dfs.Lock.Unlock()

	var (
		runner  Runner
		service dao.Service
	)

	// Create a control plane client to look up the service
	cp, err := serviced.NewControlClient(port)
	if err != nil {
		glog.Fatalf("Could not create a control plane client %v", err)
	}
	glog.Infof("Connected to the control plane at %s", port)

	if err := cp.GetService(cfg.ServiceId, &service); err != nil {
		glog.Fatalf("Unable to find service %s", cfg.ServiceId)
	}

	// Bind mount on /serviced
	dir, bin, err := serviced.ExecPath()
	if err != nil {
		glog.Fatalf("Unable to find serviced binary: %v", err)
	}
	servicedVolume := fmt.Sprintf("%s:/serviced", dir)

	// Bind mount the pwd
	dir, err = os.Getwd()
	pwdVolume := fmt.Sprintf("%s:/mnt/pwd", dir)

	// Get the shell command
	shellcmd := cfg.Command
	if cfg.Command == "" {
		shellcmd = "su -"
	}

	// Get the serviced command
	svcdcmd := fmt.Sprintf("/serviced/%s", bin)

	// Get the proxy command
	proxycmd := []string{svcdcmd, "-logtostderr=false", "proxy", "-logstash=false", "-autorestart=false", service.Id, shellcmd}

	// Get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		glog.Fatalf("Docker not found: %v", err)
	}
	argv := []string{"run", "-v", servicedVolume, "-v", pwdVolume}
	argv = append(argv, cfg.Envv...)

	if cfg.SaveAs != "" {
		argv = append(argv, fmt.Sprintf("--name=%s", cfg.SaveAs))
	} else {
		argv = append(argv, "-rm")
	}

	if cfg.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	argv = append(argv, service.ImageId)
	argv = append(argv, proxycmd...)

	glog.Infof("%s %s", docker, argv)
	runner, err = CreateCommand(docker, argv)
	if err != nil {
		glog.Fatalf("Unable to run command {%s %s} %v", docker, argv, err)
	}

	// Wire it up
	inst := &ProcessInstance{
		Stdout: runner.StdoutPipe(),
		Stderr: runner.StderrPipe(),
		Stdin:  make(chan string),
		Signal: make(chan int),
		Result: make(chan Result),
	}

	go func() {
		go func() {
			glog.Infof("Reading from stdin/signal channels")
			for inst.Stdin != nil || inst.Signal != nil {
				select {
				case m, ok := <-inst.Stdin:
					if ok {
						runner.Write([]byte(m))
					} else {
						inst.Stdin = nil
					}
				case s, ok := <-inst.Signal:
					if ok {
						runner.Signal(syscall.Signal(s))
					} else {
						inst.Signal = nil
					}
				}
			}
		}()

		if err := runner.Reader(MAXBUFFER); err != nil {
			inst.Result <- Result{1, err, NORMAL}
		} else {
			inst.Result <- Result{0, err, NORMAL}
		}
	}()

	return inst
}
