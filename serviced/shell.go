package main

import (
    "github.com/gorilla/websocket"

    "bytes"
    "fmt"
    "io"
    "net/http"
    "os/exec"
    "syscall"
    "time"
)

type ShellRequest struct {
    Command string
    Signal  int
}

type ShellResponse struct {
    Stdin, Stdout, Stderr, Result string
}

type ShellPipe struct {
    stream      io.ReadCloser
    Message     chan byte
    Error       chan error
}

func LoadShellPipe(stream io.ReadCloser) ShellPipe {
    p := ShellPipe {
        stream:     stream,
        Message:    make(chan byte),
        Error:      make(chan error),
    }
    go p.run()
    return p
}

func (p *ShellPipe) run() {
    for {
        buffer := make([]byte, 1)
        n, err := p.stream.Read(buffer)

        if n > 0 {
            p.Message <- buffer[0]
        } else {
            p.Error <- err
            return
        }
    }
}

func (p *ShellPipe) Close() {
    close(p.Message)
}

type Shell struct {
    Done        bool
    cmd         *exec.Cmd
    stdin       io.WriteCloser
    stdout      ShellPipe
    stderr      ShellPipe
    signal      chan int
    response    chan ShellResponse
}

func Connect(command string, send chan ShellResponse) (*Shell, error) {
    cmd := exec.Command("bash", "-c", command)

    // Initialize stdin pipe
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }

    // Initialize stdout pipe
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }

    // Initialize stderr pipe
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return  nil, err
    }

    // Start the shell
    if err := cmd.Start(); err != nil {
        return nil, err
    }

    s := Shell {
        Done:       false,
        cmd:        cmd,
        stdin:      stdin,
        stdout:     LoadShellPipe(stdout),
        stderr:     LoadShellPipe(stderr),
        signal:     make(chan int),
        response:   send,
    }
    go s.recv()

    return &s, nil
}

func (s *Shell) Send(input string) error {
    if _, err := s.stdin.Write([]byte(input)); err != nil {
        return err
    }
    s.response <- ShellResponse{Stdin: input}
    return nil
}

func (s *Shell) recv() {
    var eof bool
    var stdoutBuffer, stderrBuffer bytes.Buffer

    defer func() {
        s.stdout.Close()
        s.stderr.Close()
        s.Done = true
    }()

    for {
        select {
        case m := <-s.stdout.Message:
            stdoutBuffer.WriteByte(m)
            if m == '\n' {
                s.response <- ShellResponse{Stdout: stdoutBuffer.String()}
                stdoutBuffer.Reset()
            }
        case e := <-s.stdout.Error:
            if e == io.EOF {
                if stdoutBuffer.Len() > 0 {
                    s.response <- ShellResponse{Stdout: stdoutBuffer.String()}
                    stdoutBuffer.Reset()
                }
                if eof {
                    if err := s.cmd.Wait(); err != nil {
                        s.response <- ShellResponse{Result: fmt.Sprintf("%s", err)}
                    } else {
                        s.response <- ShellResponse{Result: "0"}
                    }
                    return
                }
                eof = true
            } else {
                return
            }
        case m := <-s.stderr.Message:
            stderrBuffer.WriteByte(m)
            if m == '\n' {
                s.response <- ShellResponse{Stderr: stderrBuffer.String()}
                stderrBuffer.Reset()
            }
        case e := <-s.stderr.Error:
            if e == io.EOF {
                if stderrBuffer.Len() > 0 {
                    s.response <- ShellResponse {Stderr: stderrBuffer.String()}
                    stderrBuffer.Reset()
                }
                if eof {
                    if err := s.cmd.Wait(); err != nil {
                        s.response <- ShellResponse {Result: fmt.Sprintf("%s", err)}
                    } else {
                        s.response <- ShellResponse {Result: "0"}
                    }
                    return
                }
                eof = true
            } else {
                return
            }
        case <-time.After(1 * time.Second):
            // Hanging process, dump whatever is on the pipes
            var response ShellResponse
            submit := false
            if stdoutBuffer.Len() > 0 {
                response.Stdout = stdoutBuffer.String()
                stdoutBuffer.Reset()
                submit = true
            }
            if stderrBuffer.Len() > 0 {
                response.Stderr = stderrBuffer.String()
                stderrBuffer.Reset()
                submit = true
            }
            if submit {
                s.response <- response
            }
        case sig := <-s.signal:
            signal := syscall.Signal(sig)
            if err := s.cmd.Process.Signal(signal); err != nil {
                return
            }
        }
    }
}

type connection struct {
    // The websocket connection.
    ws      *websocket.Conn

    // The shell connection.
    cmd     *Shell

    // Buffered channel of outbound messages
    send    chan ShellResponse
}

func (c *connection) reader() {
    for {
        var req ShellRequest
        if err := c.ws.ReadJSON(&req); err != nil {
            break
        }

        if c.cmd == nil || c.cmd.Done {
            if cmd, err := Connect(req.Command, c.send); err != nil {
                break
            } else {
                c.cmd = cmd
            }
        } else {
            if req.Signal > 0 {
                c.cmd.signal <- req.Signal
            } else if err := c.cmd.Send(req.Command); err != nil {
                break
            }
        }
    }
    c.ws.Close()
}

func (c *connection) writer() {
    for response := range c.send {
        if err := c.ws.WriteJSON(response); err != nil {
            break
        }
    }
    c.ws.Close()
}

type hub struct {
    // Registered connections.
    connections map[*connection] bool

    // Register requests from the connections.
    register chan *connection

    // Unregister requests from the connections.
    unregister chan *connection
}

func (h *hub) run() {
    for {
        select {
        case c:= <-h.register:
            h.connections[c] = true
        case c:= <-h.unregister:
            delete(h.connections, c)
            close(c.send)
        }
    }
}

var h = hub {
    register:       make(chan *connection),
    unregister:     make(chan *connection),
    connections:    make(map[*connection]bool),
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
    if _, ok := err.(websocket.HandshakeError); ok {
        http.Error(w, "Not a websocket handshake", 400)
        return
    } else if err != nil {
        return
    }
    c := &connection {
        send: make(chan ShellResponse),
        ws: ws,
    }
    h.register <- c
    defer func() {h.unregister <- c}()
    go c.writer()
    c.reader()
}
