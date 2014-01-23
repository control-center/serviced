package shell

import (
	"github.com/gorilla/websocket"

    "bytes"
    "fmt"
    "io"
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

type Process interface {
    Stdin() io.Writer
    Stdout() io.Reader
    Stderr() io.Reader
    Resize(cols, rows *int) error
    Wait() error
    Kill(s *int) error
    Close()
}

type Request struct {
	Action string
    Argv []string
    Signal *int

    TermName string
    TermCwd string
    TermCols *int
    TermRows *int
    TermUid *int
    TermGid *int
    TermEnv map[string]string
}

type Response struct {
	Timestamp                     int64
	Stdin, Stdout, Stderr, Result string
}

type WebsocketShell struct {
	// The websocket connection
	ws *websocket.Conn

	// The shell connection
	process Process

	// Buffered channel of outbound messages
	send chan Response
}

func (wss *WebsocketShell) reader() {
	for {
		var req Request
		if err := wss.ws.ReadJSON(&req); err != nil {
			break
		}
		if req.Action == "" {
			wss.send <- Response{Timestamp: time.Now().Unix(), Result: "required field 'Action'"}
			continue
		}
        if len(req.Argv) == 0 {
            wss.send <- Response{Timestamp: time.Now().Unix(), Result: "required field 'Argv'"}
            continue
        }

        var (
            name, file, cwd string
            args []string
            env map[string]string
            cols, rows, uid, gid *int
            signal *int
        )
        name = req.TermName
        cwd = req.TermCwd
        env = req.TermEnv
        cols = req.TermCols
        rows = req.TermRows
        uid = req.TermUid
        gid = req.TermGid
        signal = req.Signal
        file = req.Argv[0]
        if len(req.Argv) > 1 {
            args = req.Argv[1:]
        }

		if wss.process == nil {
			switch req.Action {
			case FORK:
                if term, err := CreateTerminal(name, file, args, env, cwd, cols, rows, uid, gid); err != nil {
					// LOGME: fmt.Sprint(err)
					wss.send <- Response{Timestamp: time.Now().Unix(), Result: "unable to fork pty"}
				} else {
					wss.process = term
				}
			case OPEN:
				if term, err := OpenTerminal(cols, rows); err != nil {
					// LOGME: fmt.Sprint(err)
					wss.send <- Response{Timestamp: time.Now().Unix(), Result: "unable to open pty"}
				} else {
					wss.process = term
				}
			case EXEC:
				if cmd, err := CreateCommand(file, args); err != nil {
					// LOGME: fmt.Sprint(err)
					wss.send <- Response{Timestamp: time.Now().Unix(), Result: "unable to run exec"}
				} else {
					wss.process = cmd
				}
			default:
				// LOGME: no running processes
				wss.send <- Response{Timestamp: time.Now().Unix(), Result: "no running process"}
				continue
			}
			go wss.respond()
		} else {
			switch req.Action {
			case RESIZE:
				if err := wss.process.Resize(cols, rows); err != nil {
					// LOGME: fmt.Sprint(err)
				}
			case SIGNAL:
				if err := wss.process.Kill(signal); err != nil {
					// LOGME: fmt.Sprint(err)
				}
			case EXEC:
				if err := wss.tx(strings.Join(req.Argv, " ")); err != nil {
					// LOGME: message failed to send
				}
			default:
				// LOGME: invalid action, ignoring
			}
		}
	}
	// LOGME: closing websocket connection
	wss.ws.Close()
}

func (wss *WebsocketShell) writer() {
	for response := range wss.send {
		if err := wss.ws.WriteJSON(response); err != nil {
			break
		}
	}
	// LOGME: closing websocket connection
	wss.ws.Close()
}

func (wss *WebsocketShell) tx(input string) error {
	if _, err := wss.process.Stdin().Write([]byte(input)); err != nil {
		wss.send <- Response{Timestamp: time.Now().Unix(), Result: "message failed to send"}
		return err
	}
	wss.send <- Response{Timestamp: time.Now().Unix(), Stdin: input}
	// LOGME: >> {input}
	return nil
}

func (wss *WebsocketShell) respond() {
	var (
		eof                  bool
		stdoutMsg, stderrMsg chan byte
		stdoutErr, stderrErr chan error
		stdoutBuf, stderrBuf bytes.Buffer
	)
	stdoutMsg, stdoutErr = pipe(wss.process.Stdout())
	stderrMsg, stderrErr = pipe(wss.process.Stderr())
	defer func() {
		close(stdoutMsg)
		close(stdoutErr)
		close(stderrMsg)
		close(stderrErr)
		wss.process.Close()
		wss.process = nil
	}()

	for {
		select {
		case m := <-stdoutMsg:
			stdoutBuf.WriteByte(m)
			if m == '\n' {
				wss.send <- Response{Timestamp: time.Now().Unix(), Stdout: stdoutBuf.String()}
				// LOGME: stdoutBuf.String()
				stdoutBuf.Reset()
			}
		case e := <-stdoutErr:
			if e == io.EOF {
				if stdoutBuf.Len() > 0 {
					wss.send <- Response{Timestamp: time.Now().Unix(), Stdout: stdoutBuf.String()}
					// LOGME: stdoutBuf.String()
					stdoutBuf.Reset()
				}
				if eof {
					if err := wss.process.Wait(); err != nil {
						wss.send <- Response{Timestamp: time.Now().Unix(), Result: fmt.Sprint(err)}
						// LOGME: stdoutBuf.String()
					} else {
						wss.send <- Response{Timestamp: time.Now().Unix(), Result: "0"}
						// LOGME: received code 0
					}
					return
				}
				eof = true
			} else {
				wss.send <- Response{Timestamp: time.Now().Unix(), Result: "connection closed unexpectedly"}
				// LOGME: connection closed unexpectedly
				return
			}
		case m := <-stderrMsg:
			stderrBuf.WriteByte(m)
			if m == '\n' {
				wss.send <- Response{Timestamp: time.Now().Unix(), Stderr: stderrBuf.String()}
				// LOGME: stdoutBuf.String()
				stderrBuf.Reset()
			}
		case e := <-stderrErr:
			if e == io.EOF {
				if stderrBuf.Len() > 0 {
					wss.send <- Response{Timestamp: time.Now().Unix(), Stderr: stderrBuf.String()}
					// LOGME: stdoutBuf.String()
					stderrBuf.Reset()
				}
				if eof {
					if err := wss.process.Wait(); err != nil {
						wss.send <- Response{Timestamp: time.Now().Unix(), Result: fmt.Sprint(err)}
						// LOGME: stdoutBuf.String()
					} else {
						wss.send <- Response{Timestamp: time.Now().Unix(), Result: "0"}
						// LOGME: received code 0
					}
					return
				}
				eof = true
			} else {
				wss.send <- Response{Timestamp: time.Now().Unix(), Result: "connection closed unexpectedly"}
				// LOGME: connection closed unexpectedly
				return
			}
		case <-time.After(1 * time.Second):
			// Hanging process; dump whatever is on the pipes
			var (
				response Response
				submit   bool = false
			)
			if stdoutBuf.Len() > 0 {
				response.Stdout = stdoutBuf.String()
				// LOGME: stdoutBuf.String()
				stdoutBuf.Reset()
				submit = true
			}
			if stderrBuf.Len() > 0 {
				response.Stderr = stderrBuf.String()
				// LOGME: stderrBuf.String()
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
