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

func Connect(cp *serviced.LBClient, ws *websocket.Conn) *WebsocketShell {
	return &WebsocketShell{
		cp:   cp,
		ws:   ws,
		send: make(chan response),
	}
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
			var service dao.Service
			if err := wss.cp.getService(req.ServiceId, &service); err != nil {
				result := fmt.Sprintf("cannot access service %s: %v", req.ServiceId, err)
				wss.send <- response{Result: result}
			}

			switch req.Action {
			case FORK, EXEC:
				process := dao.NewProcess(req.Cmd, env, true)
				if err := service.Exec(process); err != nil {
					result := fmt.Sprintf("unable to start container: %v", err)
					wss.send <- response{Result: result}
				} else {
					wss.process = process
				}
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
		wss.process.Close()
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
