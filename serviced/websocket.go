package main

import (
	"github.com/gorilla/websocket"

	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"net"
	"net/http"

	"fmt"
	"io"
	"log"
	"syscall"
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
	if ws != nil {
		defer ws.Close()
	}
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}

	for {
		_, msg, _ := ws.ReadMessage()
		if len(msg) > 0 {
			fmt.Println(msg)
		}
	}
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
	var req request
	if err := clientConn.ReadJSON(&req); err != nil {
		return nil
	}

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

	// Next, have Proxy connect to the Agent and tell it to start the Shell
	addr = "ws://" + addr
	agentConn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		return nil
	}

	// The Proxy-Agent connection has at this point been upgraded to a
	// websocket. Have that websocket dump output into our local Process
	// instance.
	go StreamWebsocketToProc(process, clientConn)
	go StreamProcToWebsocket(process, agentConn)

	// Now hook our local Process instance up to the client websocket so the
	// client is receiving output from the agent, proxied by us
	agentConn.WriteJSON(process)
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
