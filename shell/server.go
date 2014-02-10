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
	inst.signal <- int(syscall.SIGKILL)
	inst.closeIncoming()
}

func (p *ProcessInstance) Close() {
	p.closeIncoming()
	p.closeOutgoing()
}

func (p *ProcessInstance) closeIncoming() {
	glog.V(0).Infof("Closing incoming channels")
	if _, ok := <-p.stdin; ok {
		close(p.stdin)
	}
	if _, ok := <-p.signal; ok {
		close(p.signal)
	}
}

func (p *ProcessInstance) closeOutgoing() {
	glog.V(0).Infof("Closing outgoing channels")
	if _, ok := <-p.stdout; ok {
		close(p.stdout)
	}
	if _, ok := <-p.stderr; ok {
		close(p.stderr)
	}
	if _, ok := <-p.result; ok {
		close(p.result)
	}
}

func (p *ProcessInstance) ReadRequest(ns *socketio.NameSpace) {
	ns.On("signal", func(n *socketio.NameSpace, signal int) {
		p.signal <- signal
	})

	ns.On("stdin", func(n *socketio.NameSpace, stdin string) {
		glog.Infof("Received stdin: %s", stdin)
		p.stdin <- stdin
	})

	glog.V(0).Info("Hooked up incoming events!")
}

func (p *ProcessInstance) WriteRequest(ns *socketio.NameSpace) {
	glog.V(0).Info("Hooking up input channels!")
	for p.stdin != nil || p.signal != nil {
		select {
		case m, ok := <-p.stdin:
			if !ok {
				glog.V(0).Infof("Setting stdin to nil")
				p.stdin = nil
				continue
			} else {
				ns.Emit("stdin", m)
			}
		case m, ok := <-p.signal:
			if !ok {
				p.signal = nil
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
		p.stdout <- stdout
	})

	ns.On("stderr", func(n *socketio.NameSpace, stderr string) {
		glog.Infof("Process received stderr: %s", stderr)
		p.stderr <- stderr
	})

	ns.On("result", func(n *socketio.NameSpace, result Result) {
		glog.Infof("Process received stderr: %s", result)
		p.result <- result
	})
	glog.V(0).Info("Hooked up outgoing events!")

}

func (p *ProcessInstance) WriteResponse(ns *socketio.NameSpace) {
	glog.V(0).Info("Hooking up output channels!")
	for p.stdout != nil || p.stderr != nil || p.result != nil {
		select {
		case m, ok := <-p.stdout:
			if !ok {
				p.stdout = nil
				continue
			} else {
				glog.Infof("Emitting stdout: %s", m)
				ns.Emit("stdout", m)
			}
		case m, ok := <-p.stderr:
			if !ok {
				p.stderr = nil
				continue
			} else {
				glog.Infof("Emitting stderr: %s", m)
				ns.Emit("stderr", m)
			}
		case m, ok := <-p.result:
			if !ok {
				p.result = nil
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
		stdin:  make(chan string),
		stdout: make(chan string),
		stderr: make(chan string),
		signal: make(chan int),
		result: make(chan Result),
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
	var (
		runner   Runner
		service  *dao.Service
		services []*dao.Service
	)

	// Create a control plane client to look up the service
	controlplane, err := serviced.NewControlClient(e.port)
	if err != nil {
		glog.Fatalf("Could not create a control plane client %v", err)
	}
	glog.Infof("We got us a control plane client!")

	err = (*controlplane).GetServices(&empty, &services)
	for _, svc := range services {
		if svc.Id == cfg.ServiceId || svc.Name == cfg.ServiceId {
			service = svc
			break
		}
	}

	glog.Infof("Service found")

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
	shellCmd := cfg.Command
	if cfg.Command == "" {
		shellCmd = "su -"
	}

	// Get the proxy Command
	proxyCmd := []string{fmt.Sprintf("/serviced/%s", bin), "-logtostderr=false", "proxy", "-logstash=false", "-autorestart=false", service.Id, shellCmd}
	// Get the docker start command
	docker, err := exec.LookPath("docker")
	if err != nil {
		glog.Fatalf("Docker is not installed: %v", err)
	}
	argv := []string{"run", "-rm", "-v", servicedVolume, "-v", pwdVolume}
	argv = append(argv, cfg.Envv...)

	if cfg.IsTTY {
		argv = append(argv, "-i", "-t")
	}

	argv = append(argv, service.ImageId)
	argv = append(argv, proxyCmd...)

	runner, err = CreateCommand(docker, argv)

	if err != nil {
		glog.Fatalf("Unable to run command: %v", err)
	}
	// Wire it up

	inst := &ProcessInstance{
		stdout: runner.StdoutPipe(),
		stderr: runner.StderrPipe(),
		stdin:  make(chan string),
		signal: make(chan int),
		result: make(chan Result),
	}
	glog.Infof("Process instance! %s", inst)

	go e.send(inst, runner)
	return inst
}

func (e *Executor) send(p *ProcessInstance, r Runner) {
	go r.Reader(8192)
	go func() {
		glog.V(0).Infof("Beginning to read from stdin/signal channels")
		for {
			select {
			case m := <-p.stdin:
				glog.V(0).Infof("Read a byte")
				r.Write([]byte(m))
			case s := <-p.signal:
				r.Signal(syscall.Signal(s))
			}
		}
	}()

	<-r.ExitedPipe()
	glog.V(0).Infof("Exited reading stdin!")

	if e := r.Error(); e != nil {
		p.result <- Result{1, NORMAL}
	} else {
		p.result <- Result{0, NORMAL}
	}
}
