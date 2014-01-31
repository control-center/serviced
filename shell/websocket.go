package shell

import (
	"github.com/gorilla/websocket"

	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

const (
	FORK   = "FORK"
	EXEC   = "EXEC"
	RESIZE = "RESIZE"
	SIGNAL = "SIGNAL"
)

type process interface {
	Reader(size int)
	Writer(data []byte) (int, error)
	Stdout() chan string
	Stderr() chan string
	Exited() chan bool
	Error() error
	Resize(cols, rows *int) error
	Kill(s *int) error
	Close()
}

type request struct {
	Action string
	Argv   []string
	Signal *int

	TermName string
	TermCwd  string
	TermCols *int
	TermRows *int
	TermUid  *int
	TermGid  *int
	TermEnv  map[string]string
}

type response struct {
	Timestamp                     int64
	Stdin, Stdout, Stderr, Result string
}

type WebsocketShell struct {
	// The websocket connection
	ws *websocket.Conn

	// The shell connection
	proc process

	// Buffered channel of outbound messages
	send chan response
}

func Connect(ws *websocket.Conn) *WebsocketShell {
	return &WebsocketShell{
		ws:   ws,
		send: make(chan response),
	}
}

func (wss *WebsocketShell) Close() {
	close(wss.send)
}

func (wss *WebsocketShell) Reader() {
	for {
		var req request
		if err := wss.ws.ReadJSON(&req); err == io.EOF {
			break
		} else if err != nil {
			wss.send <- response{Timestamp: time.Now().Unix(), Result: "error parsing JSON"}
			continue
		}
		if req.Action == "" {
			wss.send <- response{Timestamp: time.Now().Unix(), Result: "required field 'Action'"}
			continue
		}

		var (
			name, file, cwd      string
			args                 []string
			env                  map[string]string
			cols, rows, uid, gid *int
			signal               *int
		)
		name = req.TermName
		cwd = req.TermCwd
		env = req.TermEnv
		cols = req.TermCols
		rows = req.TermRows
		uid = req.TermUid
		gid = req.TermGid
		signal = req.Signal

		if len(req.Argv) > 0 {
			file = req.Argv[0]
			if len(req.Argv) > 1 {
				args = req.Argv[1:]
			}
		}

		if wss.proc == nil {
			switch req.Action {
			case FORK:
				if term, err := CreateTerminal(name, file, args, env, cwd, cols, rows, uid, gid); err != nil {
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "unable to fork pty"}
				} else {
					wss.proc = term
				}
			case EXEC:
				if len(req.Argv) == 0 {
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "required field 'Argv'"}
					continue
				}
				if cmd, err := CreateCommand(file, args); err != nil {
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "unable to run exec"}
				} else {
					wss.proc = cmd
				}
			default:
				wss.send <- response{Timestamp: time.Now().Unix(), Result: "no running process"}
				continue
			}
			go wss.respond()
		} else {
			switch req.Action {
			case RESIZE:
				wss.proc.Resize(cols, rows)
			case SIGNAL:
				wss.proc.Kill(signal)
			case EXEC:
				wss.tx(strings.Join(req.Argv, " "))
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

func (wss *WebsocketShell) tx(input string) error {
	var r = response{Stdin: input}
	if _, err := wss.proc.Writer([]byte(input)); err != nil {
		r.Timestamp = time.Now().Unix()
		r.Result = "message failed to send"
		wss.send <- r
		return err
	}
	r.Timestamp = time.Now().Unix()
	wss.send <- r
	return nil
}

func (wss *WebsocketShell) respond() {
	go wss.proc.Reader(8192)

	stdout := wss.proc.Stdout()
	stderr := wss.proc.Stderr()
	done := wss.proc.Exited()

	defer func() {
		wss.proc.Close()
		wss.proc = nil
	}()

	for {
		now := time.Now().Unix()
		select {
		case m := <-stdout:
			wss.send <- response{Timestamp: now, Stdout: m}
		case m := <-stderr:
			wss.send <- response{Timestamp: now, Stderr: m}
		case <-done:
			if err := wss.proc.Error(); err != nil {
				wss.send <- response{Timestamp: now, Result: fmt.Sprint(err)}
			} else {
				wss.send <- response{Timestamp: now, Result: "0"}
			}
			return
		}
	}
}
