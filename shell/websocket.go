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
			log.Printf("Error parsing JSON: %s\n", err)
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
				log.Printf("spawning a new terminal %s\n", file)
				if term, err := CreateTerminal(name, file, args, env, cwd, cols, rows, uid, gid); err != nil {
					// LOGME: fmt.Sprint(err)
					log.Printf("unable to fork pty: %s\n", err)
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "unable to fork pty"}
				} else {
					wss.proc = term
				}
			case EXEC:
				if len(req.Argv) == 0 {
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "missing required field 'Argv'"}
					continue
				}
				log.Printf("running exec: %s", strings.Join(req.Argv, " "))
				if cmd, err := CreateCommand(file, args); err != nil {
					// LOGME: fmt.Sprint(err)
					log.Println(err)
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "unable to run exec"}
				} else {
					wss.proc = cmd
				}
			default:
				// LOGME: no running processes
				log.Println("no running process")
				wss.send <- response{Timestamp: time.Now().Unix(), Result: "no running process"}
				continue
			}
			go wss.respond()
		} else {
			switch req.Action {
			case RESIZE:
				log.Println("resizing terminal: %s x %s", cols, rows)
				if err := wss.proc.Resize(cols, rows); err != nil {
					// LOGME: fmt.Sprint(err)
					log.Println(err)
				}
			case SIGNAL:
				log.Println("sending signal %d", *signal)
				if err := wss.proc.Kill(signal); err != nil {
					// LOGME: fmt.Sprint(err)
					log.Println(err)
				}
			case EXEC:
				log.Println("sending: %s", strings.Join(req.Argv, " "))
				if err := wss.tx(strings.Join(req.Argv, " ")); err != nil {
					// LOGME: message failed to send
					log.Println("message failed to send")
				}
			default:
				// LOGME: invalid action, ignoring
				log.Println("invalid action received")
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
	if _, err := wss.proc.Writer([]byte(input)); err != nil {
		wss.send <- response{Timestamp: time.Now().Unix(), Result: "message failed to send"}
		return err
	}
	wss.send <- response{Timestamp: time.Now().Unix(), Stdin: input}
	// LOGME: >> {input}
	fmt.Printf(">> %s\n", input)
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
			wss.send <- response{Timestamp: now, Stdout: string(m)}
		case m := <-stderr:
			wss.send <- response{Timestamp: now, Stderr: string(m)}
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
