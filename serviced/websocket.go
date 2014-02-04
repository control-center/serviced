package main

import (
	"bufio"
	"errors"
	"github.com/gorilla/websocket"
	"os"

	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"net"
	"net/http"

	"fmt"
	"io"
	"log"
	"syscall"
	"time"
)

const (
	FORK   = "FORK"
	EXEC   = "EXEC"
	SIGNAL = "SIGNAL"
)

type request struct {
	Action    string
	ServiceId string
	Env       []string
	Cmd       string
	Signal    int
}

type response struct {
	Stdin  string
	Stdout string
	Stderr string
	Result string
}

type Stream interface {
	ClientHandler(w http.ResponseWriter, r *http.Request)
	AgentHandler(w http.ResponseWriter, r *http.Request)
	StreamClient()
	StreamAgent()
}

type WebsocketStream struct {
	client  *websocket.Conn
	agent   *websocket.Conn
	process *dao.Process
}

func (s *WebsocketStream) ClientHandler(w http.ResponseWriter, r *http.Request) {
}

func (s *WebsocketStream) AgentHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024) // TODO: Make buffer size configurable?
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	}

	s.agent = ws
	s.StreamAgent()
}

func (s *WebsocketStream) StreamAgent() {
	// Writer
	go func() {
		for {
			select {
			case m := <-proc.Stdin:
				s.agent.WriteJSON(request{Action: EXEC, Cmd: m})
			case s := <-proc.Signal:
				s.agent.WriteJSON(request{Action: SIGNAL, Signal: int(s)})
			}
		}
	}()

	// Reader
	for {
		var res response
		if err := s.agent.ReadJSON(&response); err == io.EOF {
			break
		} else if err != nil {
			// Bad read send message
		}

		if res.Stdout != "" {
			proc.Stdout <- res.Stdout
		}

		if res.Stderr != "" {
			proc.Stderr <- res.Stderr
		}

		if res.Result != "" {
			proc.Error = errors.New(res.Result)
			proc.Exited <- true
			break
		}
	}
	s.agent.Close()
}

func (s *WebsocketStream) StreamClient() {
}

type HttpStream struct {
	client  *http.Conn
	agent   *websocket.Conn
	process *dao.Process
}

type WebsocketShell struct {
	// The control plane client
	cp *serviced.LBClient

	// The websocket connection
	ws *websocket.Conn

	// The shell connection
	process *dao.Process

	// Buffered channel of outbound messages
	send chan response
}

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

<<<<<<< HEAD
	defer ws.Close()
	ws.WriteJSON(response{Result: "0"})
=======
	for {
		time.Sleep(10)
	}

>>>>>>> 91ad16b349e388e27cb54d917bc4848c3b40f1ff
}

func StreamProcToWebsocket(proc *dao.Process, ws *websocket.Conn) {
	// Websocket in (request) to proc in
	go func() {
		for {
			var req request
			if err := ws.ReadJSON(&req); err == io.EOF {
				break
			} else if err != nil {
				ws.WriteJSON(response{Result: "error parsing JSON"})
				continue
			}
			if req.Action == "" {
				ws.WriteJSON(response{Result: "required field 'Action'"})
				continue
			}
			switch req.Action {
			case SIGNAL:
				proc.Signal <- syscall.Signal(req.Signal)
			case EXEC:
				proc.Stdin <- req.Cmd
			}
		}
	}()
	// Proc out to websocket out
	for {
		select {
		case m := <-proc.Stdout:
			ws.WriteJSON(response{Stdout: m})
		case m := <-proc.Stderr:
			ws.WriteJSON(response{Stderr: m})
		case <-proc.Exited:
			if proc.Error != nil {
				ws.WriteJSON(response{Result: fmt.Sprint(proc.Error)})
			} else {
				ws.WriteJSON(response{Result: "0"})
			}
			return
		}
	}
}

func StreamWebsocketToProc(proc *dao.Process, ws *websocket.Conn) {

	// Websocket out to proc out
	go func() {
		for {
			var resp response
			if err := ws.ReadJSON(&resp); err == io.EOF {
				break
			} else if err != nil {
				ws.WriteJSON(response{Result: "error parsing JSON"})
				continue
			}
			switch {
			case len(resp.Stdout) > 0:
				proc.Stdout <- resp.Stdout
			case len(resp.Stderr) > 0:
				proc.Stderr <- resp.Stderr
			case len(resp.Result) > 0:
				proc.Error = errors.New(resp.Result)
				proc.Exited <- true
			default:
				WriteToFile("SHIT GOT BROKE " + fmt.Sprint(resp))
			}
		}
	}()

	// Proc in to websocket in
	for {
		select {
		case m := <-proc.Stdin:
			ws.WriteJSON(request{Cmd: m, Action: EXEC}) // We never send FORK in this direction, trust me
		case m := <-proc.Signal:
			ws.WriteJSON(request{Signal: int(m), Action: SIGNAL})
		}
	}
}

func WriteToFile(msg string) {
	f, _ := os.Create("/opt/zenoss/log/servicedproxy.log")
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	w.Write([]byte(msg + "\n"))
}

func Connect(cp *serviced.LBClient, ws *websocket.Conn) *WebsocketShell {
	return &WebsocketShell{
		cp:   cp,
		ws:   ws,
		send: make(chan response),
	}
}

func ProxyCommandOverWS(addr string, clientConn *websocket.Conn) (proc *dao.Process) {
	// Client <--ws--> Proxy <--ws--> Agent <--os--> Shell
	// This code executes in Proxy, creating the two connections on either
	// side and hooking the streams together, more or less

	// First, read the first packet from the Client which contains the process information
	WriteToFile("Beginning of func")
	var req request
	if err := clientConn.ReadJSON(&req); err != nil {
		return nil
	}
	WriteToFile("I got a request")

	var istty bool
	switch req.Action {
	case FORK:
		istty = true
	case EXEC:
		istty = false
	default:
		return nil
	}
	process := dao.NewProcess(req.ServiceId, req.Cmd, req.Env, istty)
	WriteToFile("Made a process")

	// Next, have Proxy connect to the Agent and tell it to start the Shell
	addr = "ws://" + addr
	WriteToFile("About to dial " + addr)
	agentConn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		return nil
	}
	WriteToFile("Dialed!")

	// The Proxy-Agent connection has at this point been upgraded to a
	// websocket. Have that websocket dump output into our local Process
	// instance.
	WriteToFile("Streaming like a mofo")

	// Now hook our local Process instance up to the client websocket so the
	// client is receiving output from the agent, proxied by us
	WriteToFile("About to write some json")
	e := agentConn.WriteJSON(process)
	if e != nil {
		WriteToFile(fmt.Sprint(e))
	}
	go StreamWebsocketToProc(process, clientConn)
	go StreamProcToWebsocket(process, agentConn)

	return process
}

func ProxyCommandOverHTTP(addr string, clientConn *net.Conn) {
}

func (wss *WebsocketShell) Close() {
	close(wss.send)
}

func (wss *WebsocketShell) Reader() {
	defer func() {
		if wss.process != nil {
			wss.process.Signal <- syscall.SIGKILL
		}
	}()

	for {
		var req request
		if err := wss.ws.ReadJSON(&req); err == io.EOF {
			break
		} else if err != nil {
			wss.send <- response{Result: "error parsing JSON"}
			continue
		}
		if req.Action == "" {
			wss.send <- response{Result: "required field 'Action'"}
			continue
		}

		//var (
		//	serviceId, cmd string
		//	env            []string
		//	signal         int
		//)
		//serviceId = req.ServiceId
		//cmd = req.Cmd
		//env = req.Env
		//signal = req.Signal

		//if wss.process == nil {

		//	switch req.Action {
		//	case FORK, EXEC:
		//		process := dao.NewProcess(req.Cmd, env, true)
		//		wss.process = process

		//		//wss.cp.ExecAsService(&ExecRequest{
		//		//	Process:   process,
		//		//	ServiceId: serviceId,
		//		//}, nil)

		//		//		if err := service.Exec(process); err != nil {
		//		//			result := fmt.Sprintf("unable to start container: %v", err)
		//		//			wss.send <- response{Result: result}
		//		//		} else {
		//		//			wss.process = process
		//		//		}
		//	default:
		//		wss.send <- response{Result: "no running process"}
		//		continue
		//	}
		//	go wss.respond()
		//} else {
		//	switch req.Action {
		//	case SIGNAL:
		//		wss.process.Signal <- syscall.Signal(signal)
		//	case EXEC:
		//		wss.process.Stdin <- req.Cmd
		//	}
		//}
	}
	wss.ws.Close()
}

func (wss *WebsocketShell) Writer() {
	for response := range wss.send {
		if err := wss.ws.WriteJSON(response); err != nil {
			break
		}
	}
	// LOGME: closing websocket connection
	log.Println("Closing websocket connection")
	wss.ws.Close()
}

func (wss *WebsocketShell) respond() {

	defer func() {
		wss.process = nil
	}()

	for {
		select {
		case m := <-wss.process.Stdout:
			wss.send <- response{Stdout: m}
		case m := <-wss.process.Stderr:
			wss.send <- response{Stderr: m}
		case <-wss.process.Exited:
			if wss.process.Error != nil {
				wss.send <- response{Result: fmt.Sprint(wss.process.Error)}
			} else {
				wss.send <- response{Result: "0"}
			}
			return
		}
	}
}
