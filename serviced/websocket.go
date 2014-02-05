package main

import (
	"bufio"
	"errors"
	"github.com/gorilla/websocket"
	"os"

	"github.com/zenoss/serviced/dao"
	"net"
	"net/http"

	"fmt"
	"io"
	"syscall"
	"time"
)

const (
	FORK   = "FORK"
	EXEC   = "EXEC"
	SIGNAL = "SIGNAL"
)

// Client->Agent protocol
type request struct {
	Action    string
	ServiceId string
	Env       []string
	Cmd       string
	Signal    int
}

// Agent->Client protocol
type response struct {
	Stdin  string
	Stdout string
	Stderr string
	Result string
}

// Shitty log func for debugging
func WriteToFile(msg string) {
	f, _ := os.Create("/opt/zenoss/log/servicedproxy.log")
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	w.Write([]byte(msg + "\n"))
}

// Describes streams from an agent-executed process to a client
type ProcessStream interface {

	// Initiate client-side communication and create Process
	StreamClient(http.ResponseWriter, *http.Request, chan *dao.Process)

	// Initiate agent-side communication and kick off shell
	StreamAgent()

	// Wait for the process to end
	Wait()

	// Shut down resources
	Close()
}

type WebsocketProcessStream struct {
	client  *websocket.Conn
	agent   *websocket.Conn
	process *dao.Process
	addr    string
	exited  chan bool
}

type HttpProcessStream struct {
	client  *net.Conn
	agent   *websocket.Conn
	process *dao.Process
}

type WebsocketProcessHandler struct {
	addr string
}

type HTTPProcessHandler struct {
	addr string
}

// Implement http.Handler
func (h *WebsocketProcessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stream := &WebsocketProcessStream{addr: h.addr, exited: make(chan bool)}
	defer stream.Close()

	// Create a client and wait for the process packet
	pc := make(chan *dao.Process)
	go stream.StreamClient(w, r, pc)
	// Wait for the process to come from the client
	stream.process = <-pc

	// Now that we have the process, connect to the agent
	go stream.StreamAgent()

	// Wait for the process to die
	stream.Wait()
}

// Read the first packet from the client and deserialize to Process
func (s *WebsocketProcessStream) readProcessPacket() *dao.Process {
	var (
		req   request
		istty bool
	)
	if err := s.client.ReadJSON(&req); err != nil {
		return nil
	}
	switch req.Action {
	case FORK:
		istty = true
	case EXEC:
		istty = false
	default:
		return nil
	}
	return dao.NewProcess(req.ServiceId, req.Cmd, req.Env, istty)
}

func (s *WebsocketProcessStream) StreamClient(w http.ResponseWriter, r *http.Request, pc chan *dao.Process) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	s.client = ws
	s.process = s.readProcessPacket()
	pc <- s.process
	s.forwardToClient()
}

func (s *WebsocketProcessStream) StreamAgent() {
	// TODO: Proper ws scheme validation
	ws, _, _ := websocket.DefaultDialer.Dial("ws://"+s.addr, nil)
	s.agent = ws
	s.agent.WriteJSON(s.process)
	s.forwardFromAgent()
}

func (s *WebsocketProcessStream) Wait() {
	<-s.exited
	return
}

func (s *WebsocketProcessStream) Close() {
	// TODO: See if we need to do anything else. Close ws conns?
	if s.process != nil {
		s.process.Signal <- syscall.SIGKILL
	}
}

// Wire up the Process to the agent connection
func (s *WebsocketProcessStream) forwardFromAgent() {
	// Writer
	go func() {
		for {
			select {
			case m := <-s.process.Stdin:
				s.agent.WriteJSON(request{Action: EXEC, Cmd: m})
			case m := <-s.process.Signal:
				s.agent.WriteJSON(request{Action: SIGNAL, Signal: int(m)})
			}
		}
	}()

	// Reader
	for {
		var res response
		if err := s.agent.ReadJSON(&res); err == io.EOF {
			break
		} else if err != nil {
			// Bad read send message
		}

		if res.Stdout != "" {
			s.process.Stdout <- res.Stdout
		}

		if res.Stderr != "" {
			s.process.Stderr <- res.Stderr
		}

		if res.Result != "" {
			s.process.Error = errors.New(res.Result)
			s.process.Exited <- true
			s.exited <- true
			break
		}
	}
	s.agent.Close()
}

// Wire up the Process to the client connection
func (s *WebsocketProcessStream) forwardToClient() {
	defer s.client.Close()

	// Reader
	go func() {
		for {
			var req request
			if err := s.client.ReadJSON(&req); err == io.EOF {
				break
			} else if err != nil {
				// Bad read send message
			}

			if req.Cmd != "" {
				s.process.Stdin <- req.Cmd
			}

			if req.Signal != 0 {
				s.process.Signal <- syscall.Signal(req.Signal)
			}
		}
	}()

	// Writer
	for {
		select {
		case m := <-s.process.Stdout:
			s.client.WriteJSON(response{Stdout: m})
		case m := <-s.process.Stderr:
			s.client.WriteJSON(response{Stderr: m})
		case <-s.process.Exited:
			s.client.WriteJSON(response{Result: fmt.Sprint(s.process.Error)})
			break
		}
	}

}

// Agent-side websocket handler.
func ExecHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	defer ws.Close()

	WriteToFile("I gots a connection!!!!!")

	ws.WriteJSON(response{Stdout: "Stdout"})
	ws.WriteJSON(response{Stderr: "Stderr"})
	ws.WriteJSON(response{Result: "0"})

	defer ws.Close()
	ws.WriteJSON(response{Result: "0"})
}
