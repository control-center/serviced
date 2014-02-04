package main

import (
	"github.com/gorilla/websocket"

	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"

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

func StreamProcToWebsocket(proc *dao.Process, ws *websocket.Conn) {
	// Websocket in (request) to proc in
	go func() {
		var req request
		if err := ws.ReadJSON(&req); err == io.EOF {
			break
		} else if err != nil {
			ws.send <- response{Result: "error parsing JSON"}
			continue
		}
		if req.Action == "" {
			ws.send <- response{Result: "required field 'Action'"}
			continue
		}
		switch req.Action {
		case SIGNAL:
			process.Signal <- syscall.Signal(req.Signal)
		case EXEC:
			process.Stdin <- req.Cmd
		}
	}()
	// Proc out to websocket out
	for {
		select {
		case m := <-proc.Stdout:
			ws.send <- response{Stdout: m}
		case m := <-proc.Stderr:
			ws.send <- response{Stderr: m}
		case <-proc.Exited:
			if proc.Error != nil {
				ws.send <- response{Result: fmt.Sprint(proc.Error)}
			} else {
				ws.send <- response{Result: "0"}
			}
			return
		}
	}
}

func StreamWebsocketToProc(proc *dao.Process, ws *websocket.Conn) {

	// Websocket out to proc out
	go func() {
		var resp response
		if err := ws.ReadJSON(&resp); err == io.EOF {
			break
		} else if err != nil {
			wss.send <- response{Result: "error parsing JSON"}
			continue
		}
		switch req.Action {
		case SIGNAL:
			wss.process.Signal <- syscall.Signal(signal)
		case EXEC:
			wss.process.Stdin <- req.Cmd
		}
	}()

	// Proc in to websocket in
	for {
		select {
		case m := <-proc.Stdin:
			ws.send <- req{Cmd: m, Action: EXEC} // We never send FORK in this direction, trust me
		case m := <-proc.Signal:
			ws.send <- response{Signal: m, Action: SIGNAL}
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

func ConnectExecRequestToServicedAgent(cp *serviced.LBClient, clientConn *websocket.Conn) {
	// Client <--ws--> Proxy <--rpc/ws--> Agent <--os--> Shell
	// This code executes in Proxy, creating the two connections on either
	// side and hooking the streams together, more or less

	// First, read the first packet from the Client which contains the process information

	// Next, have Proxy connect to the Agent and tell it to start the Shell
	execReq := &serviced.ExecRequest{}
	agentConn := cp.ExecAsService(&execReq, nil)

	// The Proxy-Agent connection has at this point been upgraded to a
	// websocket. Have that websocket dump output into our local Process
	// instance.

	// Now hook our local Process instance up to the client websocket so the
	// client is receiving output from the agent, proxied by us
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

		var (
			serviceId, cmd string
			env            []string
			signal         int
		)
		serviceId = req.ServiceId
		cmd = req.Cmd
		env = req.Env
		signal = req.Signal

		if wss.process == nil {

			switch req.Action {
			case FORK, EXEC:
				process := dao.NewProcess(req.Cmd, env, true)
				wss.process = process

				wss.cp.ExecAsService(&ExecRequest{
					Process:   process,
					ServiceId: serviceId,
				}, nil)

				//		if err := service.Exec(process); err != nil {
				//			result := fmt.Sprintf("unable to start container: %v", err)
				//			wss.send <- response{Result: result}
				//		} else {
				//			wss.process = process
				//		}
			default:
				wss.send <- response{Result: "no running process"}
				continue
			}
			go wss.respond()
		} else {
			switch req.Action {
			case SIGNAL:
				wss.process.Signal <- syscall.Signal(signal)
			case EXEC:
				wss.process.Stdin <- req.Cmd
			}
		}
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
