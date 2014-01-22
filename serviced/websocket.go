// websocket.go

const (
	FORK 	= "FORK"
	OPEN	= "OPEN"
	EXEC	= "EXEC"
	RESIZE	= "RESIZE"
	SIGNAL	= "SIGNAL"
)

type Process interface {
	Stdin	*io.Writer
	Stdout	*io.Reader
	Stderr	*io.Reader
}
func (*Process) Wait() error
func (*Process) Kill(signal int) error
func (*Process) Close()
func (*Process) Resize(cols, rows int) error

type Request struct {
	Action string
	Info TerminalData
	File string
	Args []string
	Input string
}

type TerminalData struct {
	Name, Cwd string
	Env map[string]string
	Cols, Rows int
	Uid, Gid int
}

type Response struct {
	Timestamp int64
	Stdin, Stdout, Stderr string
}

type WebsocketShell struct {
	// The websocket connection
	ws *websocket.Conn
	
	// The shell connection
	process *Process
	
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
			wss.send <- Response{Timestamp:time.Now().Unix(), Result: "required field 'Action'"}
			continue
		}
		
		t := req.Info
		
		if wss.process == nil {
			switch req.Action {
			case FORK:
				if term, err := CreateTerminal(req.File, t.Name, t.Cwd, req.Argv, t.Envv, t.Cols, t.Rows, t.Uid, t.Gid); err != nil {
					// LOGME: err.String()
					wss.send <- Response{Timestamp:time.Now().Unix(), Result: "unable to fork pty"}
				} else {
					wss.process = term
				}
			case OPEN:
				if term, err := OpenTerminal(t.Cols, t.Rows); err != nil {
					// LOGME: err.String()
					wss.send <- Response{Timestamp:time.Now().Unix(), Result: "unable to open pty"}
				} else {
					wss.process = term
				}
			case EXEC:
				if cmd, err := CreateCommand(req.File, req.Argv...); err != nil {
					// LOGME: err.String()
					wss.send <- Response{Timestamp: time.Now().Unix(), Result: "unable to run exec"}
				} else {
					wss.process = cmd
				}
			default:
				// LOGME: no running processes
				wss.send <- Response{TimeStamp:time.Now().Unix(), Result: "no running process"}
				continue
			go wss.respond()
		} else {
			switch req.Action {
			case RESIZE:
				if err := wss.process.Resize(t.Cols, t.Rows); err != nil {
					// LOGME: err.String()
				}
			case SIGNAL:
				if err := wss.process.Kill(req.Signal); err != nil {
					// LOGME: err.String()
				}
			case EXEC:
				if err := wss.send(req.Argv[0]); err != nil {
					// LOGME: message failed to send
				}
			default:
				// LOGME: invalid action, ignoring
			}
		}
					
	}
	// LOGME: closing websocket connection
	c.ws.Close()
}

func (wss *WebsocketShell) writer() {
	for response := range wss.send {
		if err := wss.ws.WriteJSON(response); err != nil {
			break
		}
	}
	// LOGME: closing websocket connection
	c.ws.Close()
}

func (wss *WebsocketShell) send(input string) error {
	if _, err := wss.process.Stdin.Write([]byte(input)); err != nil {
		wss.send <- Response{Timestamp:time.Now().Unix(), Result: "message failed to send"}
		return err
	}
	wss.send <- Response{Timestamp:time.Now().Unix(), Timestamp:time.Now().Unix(), Stdin:input}
	// LOGME: >> {input}
	return nil
}

func (wss *WebsocketShell) respond() {
	var (
		eof bool,
		stdoutMsg, stderrMsg chan byte,
		stdoutErr, stderrErr chan err,
		stdoutBuf, stderrBuf bytes.Buffer,
	)
		
	stdoutMsg, stdoutErr = pipe(wss.process.Stdout)
	stderrMsg, stderrErr = pipe(wss.process.Stderr)
	
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
		case m := <- stdoutMsg:
			stdoutBuf.WriteByte(m)
			if m == '\n' {
				wss.send <- Response{Timestamp:time.Now().Unix(), Stdout: stdoutBuf.String()}
				// LOGME: stdoutBuf.String()
				stdoutBuf.Reset()
			}
		case e := <- stdoutErr:
			if e == io.EOF {
				if stdoutBuf.Len() > 0 {
					wss.send <- Response{Timestamp:time.Now().Unix(), Stdout: stdoutBuf.String()}
					// LOGME: stdoutBuf.String()
					stdoutBuf.Reset()
				}
				if eof {
					if err := wss.process.Wait(); err != nil {
						wss.send <- Response{Timestamp:time.Now().Unix(), Result: err.String()}
						// LOGME: stdoutBuf.String()
					} else {
						wss.send <- Response{Timestamp:time.Now().Unix(), Result: 0}
						// LOGME: received code 0
					}
					return
				}
				eof = true
			} else {
				wss.send <- Response{Timestamp:time.Now().Unix(), Result: "connection closed unexpectedly"}
				// LOGME: connection closed unexpectedly
				return
			}
		case m := <- stderrMsg:
			stderrBuf.WriteByte(m)
			if m == '\n' {
				wss.send <- Response{Timestamp:time.Now().Unix(), Stderr: stderrBuf.String()}
				// LOGME: stdoutBuf.String()
				stderrBuf.Reset()
			}
		case e:= <- stderrErr:
			if e == io.EOF {
				if stderrBuf.Len() > 0 {
					wss.send <- Response{Timestamp:time.Now().Unix(), Stderr: stderrErr.String()}
					// LOGME: stdoutBuf.String()
					stderrBuf.Reset()
				}
				if eof {
					if err := wss.process.Wait(); err != nil {
						wss.send <- Response{Timestamp:time.Now().Unix(), Result: err.String()}
						// LOGME: stdoutBuf.String()
					} else {
						wss.send <- Response{Timestamp:time.Now().Unix(), Result: 0}
						// LOGME: received code 0
					}
					return
				}
				eof = true
			} else {
				wss.send <- Response{Timestamp:time.Now().Unix(), Result: "connection closed unexpectedly"}
				// LOGME: connection closed unexpectedly
				return
			}
		case <-time.After(1 * time.Second):
			// Hanging process; dump whatever is on the pipes
			var (
				response Response,
				submit bool = false
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

func pipe(reader *io.Reader) (chan byte, chan error) {
	bchan := make(chan byte, 1024)
	echan := make(chan err)
	
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
