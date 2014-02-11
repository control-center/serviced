package shell

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/googollee/go-socket.io"
	"github.com/zenoss/glog"

	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
)

var empty interface{}

const (
	PROCESSKEY string = "process"
	MAXBUFFER  int    = 8192
)

func NewProcessForwarderServer(addr string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Forwarder{addr: addr},
	}
	server.sio.On("connect", server.onConnect)
	server.sio.On("disconnect", onForwarderDisconnect)
	return server
}

func NewProcessExecutorServer(port string) *ProcessServer {
	server := &ProcessServer{
		sio:   socketio.NewSocketIOServer(&socketio.Config{}),
		actor: &Executor{port: port},
	}
	server.sio.On("connect", server.onConnect)
	server.sio.On("disconnect", onExecutorDisconnect)
	return server
}

func (p *ProcessServer) Handle(pattern string, handler http.Handler) error {
	p.sio.Handle(pattern, handler)
	return nil
}

func (p *ProcessServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.sio.ServeHTTP(w, r)
}

func (s *ProcessServer) onConnect(ns *socketio.NameSpace) {
	glog.Infof("===========================================")
	glog.Infof("===========================================")
	glog.Infof("===========================================")
	glog.Infof("Received connection")
	glog.Infof("===========================================")
	glog.Infof("===========================================")
	glog.Infof("===========================================")

	ns.On("process", func(ns *socketio.NameSpace, cfg *ProcessConfig) {
		// Kick it off
		glog.Infof("Received process packet")
		proc := s.actor.Exec(cfg)
		ns.Session.Values[PROCESSKEY] = proc

		// Wire up output
		go proc.ReadRequest(ns)
		go proc.WriteResponse(ns)
	})
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
		p.Signal <- signal
	})

	ns.On("stdin", func(n *socketio.NameSpace, stdin string) {
		glog.Infof("Received stdin: %s", stdin)
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
				glog.V(0).Infof("Setting stdin to nil")
				p.Stdin = nil
				continue
			} else {
				ns.Emit("stdin", m)
			}
		case m, ok := <-p.Signal:
			if !ok {
				p.Signal = nil
				continue
			} else {
				ns.Emit("signal", m)
			}
		}
	}
}

func (p *ProcessInstance) ReadResponse(ns *socketio.NameSpace) {
	ns.On("stdout", func(n *socketio.NameSpace, stdout string) {
		glog.Infof("Process received stdout: %s", stdout)
		p.Stdout <- stdout
	})

	ns.On("stderr", func(n *socketio.NameSpace, stderr string) {
		glog.Infof("Process received stderr: %s", stderr)
		p.Stderr <- stderr
	})

	ns.On("result", func(n *socketio.NameSpace, result Result) {
		glog.Infof("Process received stderr: %s", result)
		p.Result <- result
	})
	glog.V(0).Info("Hooked up outgoing events!")

}

func (p *ProcessInstance) WriteResponse(ns *socketio.NameSpace) {
	glog.V(0).Info("Hooking up output channels!")
	for p.Stdout != nil || p.Stderr != nil || p.Result != nil {
		select {
		case m, ok := <-p.Stdout:
			if !ok {
				p.Stdout = nil
				continue
			} else {
				glog.Infof("Emitting stdout: %s", m)
				ns.Emit("stdout", m)
			}
		case m, ok := <-p.Stderr:
			if !ok {
				p.Stderr = nil
				continue
			} else {
				glog.Infof("Emitting stderr: %s", m)
				ns.Emit("stderr", m)
			}
		case m, ok := <-p.Result:
			if !ok {
				p.Result = nil
				continue
			} else {
				glog.Infof("Emitting result: %s", m)
				ns.Emit("result", m)
			}
		}
	}
}

func (f *Forwarder) Exec(cfg *ProcessConfig) *ProcessInstance {

	// Dial the remote ProcessServer
	host := strings.Split(f.addr, ":")[0]
	client, err := socketio.Dial(fmt.Sprintf("http://%s:50000/", host))

	if err != nil {
		glog.Fatalf("Unable to contact remote process server: %v", err)
	}

	client.On("connect", func(ns *socketio.NameSpace) {
		ns.Emit("process", cfg)
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

func (e *Executor) Exec(cfg *ProcessConfig) *ProcessInstance {
	return StartDocker(cfg, e.port)
}

func StartDocker(cfg *ProcessConfig, port string) *ProcessInstance {
	var (
		runner   Runner
		service  *dao.Service
		services []*dao.Service
	)

	// Create a control plane client to look up the service
	cp, err := serviced.NewControlClient(port)
	if err != nil {
		glog.Fatalf("Could not create a control plane client %v", err)
	}
	glog.Infof("Connected to the control plane at port :%s", port)

	err = (*cp).GetServices(&empty, &services)
	for _, svc := range services {
		if svc.Id == cfg.ServiceId || svc.Name == cfg.ServiceId {
			service = svc
			break
		}
	}

	glog.Infof("Found service %s", cfg.ServiceId)

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
	argv := []string{"run", "-rm", "-v", servicedVolume, "-v", pwdVolume}
	argv = append(argv, cfg.Envv...)

	if cfg.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	argv = append(argv, service.ImageId)
	argv = append(argv, proxycmd...)

	runner, err = CreateCommand(docker, argv)
	if err != nil {
		glog.Fatalf("Unable to run command: %v", err)
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
