package shell

import (
	"github.com/gorilla/websocket"

	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

const (
	FORK   = "FORK"
	OPEN   = "OPEN"
	EXEC   = "EXEC"
	RESIZE = "RESIZE"
	SIGNAL = "SIGNAL"
)

type process interface {
	Stdin() io.Writer
	Stdout() io.Reader
	Stderr() io.Reader
	Resize(cols, rows *int) error
	Wait() error
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
			case OPEN:
				log.Printf("opening a terminal\n")
				if term, err := OpenTerminal(cols, rows); err != nil {
					// LOGME: fmt.Sprint(err)
					log.Println(err)
					wss.send <- response{Timestamp: time.Now().Unix(), Result: "unable to open pty"}
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
	if _, err := wss.proc.Stdin().Write([]byte(input)); err != nil {
		wss.send <- response{Timestamp: time.Now().Unix(), Result: "message failed to send"}
		return err
	}
	wss.send <- response{Timestamp: time.Now().Unix(), Stdin: input}
	// LOGME: >> {input}
	fmt.Printf(">> %s\n", input)
	return nil
}

func (wss *WebsocketShell) respond() {
	var (
		eof                  bool
		stdoutMsg, stderrMsg chan byte
		stdoutErr, stderrErr chan error
		stdoutBuf, stderrBuf bytes.Buffer
	)
	stdoutMsg, stdoutErr = pipe(wss.proc.Stdout())
	stderrMsg, stderrErr = pipe(wss.proc.Stderr())
	defer func() {
		close(stdoutMsg)
		close(stdoutErr)
		close(stderrMsg)
		close(stderrErr)
		wss.proc.Close()
		wss.proc = nil
	}()

	for {
		select {
		case m := <-stdoutMsg:
			stdoutBuf.WriteByte(m)
			if m == '\n' {
				wss.send <- response{Timestamp: time.Now().Unix(), Stdout: stdoutBuf.String()}
				// LOGME: stdoutBuf.String()
				fmt.Print(stdoutBuf.String())
				stdoutBuf.Reset()
			}
		case e := <-stdoutErr:
			if e == io.EOF {
				if stdoutBuf.Len() > 0 {
					wss.send <- response{Timestamp: time.Now().Unix(), Stdout: stdoutBuf.String()}
					// LOGME: stdoutBuf.String()
					fmt.Print(stdoutBuf.String())
					stdoutBuf.Reset()
				}
				if eof {
					if err := wss.proc.Wait(); err != nil {
						wss.send <- response{Timestamp: time.Now().Unix(), Result: fmt.Sprint(err)}
						// LOGME: err
						fmt.Printf(">> %s <<\n", err)
					} else {
						wss.send <- response{Timestamp: time.Now().Unix(), Result: "0"}
						// LOGME: received code 0
						fmt.Printf(">> received code 0 <<\n")
					}
					return
				}
				eof = true
			} else {
				wss.send <- response{Timestamp: time.Now().Unix(), Result: "connection closed unexpectedly"}
				// LOGME: connection closed unexpectedly
				log.Printf("connection closed unexpectedly: %s\n", e)
				return
			}
		case m := <-stderrMsg:
			stderrBuf.WriteByte(m)
			if m == '\n' {
				wss.send <- response{Timestamp: time.Now().Unix(), Stderr: stderrBuf.String()}
				// LOGME: stderrBuf.String()
				fmt.Print(stderrBuf.String())
				stderrBuf.Reset()
			}
		case e := <-stderrErr:
			if e == io.EOF {
				if stderrBuf.Len() > 0 {
					wss.send <- response{Timestamp: time.Now().Unix(), Stderr: stderrBuf.String()}
					// LOGME: stderrBuf.String()
					fmt.Print(stderrBuf.String())
					stderrBuf.Reset()
				}
				if eof {
					if err := wss.proc.Wait(); err != nil {
						wss.send <- response{Timestamp: time.Now().Unix(), Result: fmt.Sprint(err)}
						// LOGME: err
						fmt.Printf(">> %s <<\n", err)
					} else {
						wss.send <- response{Timestamp: time.Now().Unix(), Result: "0"}
						// LOGME: received code 0
						fmt.Printf(">> received code 0 <<\n")
					}
					return
				}
				eof = true
			} else {
				wss.send <- response{Timestamp: time.Now().Unix(), Result: "connection closed unexpectedly"}
				// LOGME: connection closed unexpectedly
				log.Println("connection closed unexpectedly")
				return
			}
		case <-time.After(1 * time.Second):
			// Hanging process; dump whatever is on the pipes
			var (
				response response
				submit   bool = false
			)
			if stdoutBuf.Len() > 0 {
				response.Stdout = stdoutBuf.String()
				// LOGME: stdoutBuf.String()
				fmt.Print(stdoutBuf.String())
				stdoutBuf.Reset()
				submit = true
			}
			if stderrBuf.Len() > 0 {
				response.Stderr = stderrBuf.String()
				// LOGME: stderrBuf.String()
				fmt.Print(stderrBuf.String())
				stderrBuf.Reset()
				submit = true
			}
			if submit {
				wss.send <- response
			}
		}
	}
}

func pipe(reader io.Reader) (chan byte, chan error) {
	bchan := make(chan byte, 1024)
	echan := make(chan error)
	go func() {
		if reader == nil {
			echan <- io.EOF
			return
		}
		for {
			buffer := make([]byte, 1)
			n, err := reader.Read(buffer)
			if n > 0 {
				bchan <- buffer[0]
			} else {
				echan <- err
				return
			}
		}
	}()
	return bchan, echan
}
