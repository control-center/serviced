package shell

const (
    BUFFER_MAX int64 = 2097152 // 2MB
)

type session struct {
	host         *wsshell
	participants map[*wsshell]bool
	proc         process
	buffer       bytes.Buffer
}

func StartSession(connection *wsshell, shell process) (*session, error) {
    s := session {
        host: connection,
        participants: {connection:true},
        proc: shell,
    }
    go s.send()
    return &s, nil
}

func OpenSession(id string, connection *wsshell) (*session, error) {
    // lookup the session id
    s := GetSession(id)
    if s == nil {
        return nil, errors.New("session not found")
    }

    // send the buffer to the connection
    b := s.buffer.Bytes()
    if len(b) > proc.max() {
        connection.stdout <- string(b[len(b)-proc.max():])
    else {
        connection.stdout <- string(b)
    }

    // add the connection to the session
    s.participants[connection] = true

    // set host if not set
    if s.host == nil {
        s.host = connection
    }

    h.connect <- s

    return s, nil
}

func (s *session) send() {
    stdout := s.proc.stdout()
    stderr := s.proc.stderr()
    done := s.proc.exited()

    go s.proc.reader()
    defer s.disconnect()
    for {
        select {
        case m := <-stdout:
            size := s.buffer.Len() + len(m) - BUFFER_MAX
            if size > 0 {
                s.buffer.Next(size)
            }
            if _,err := s.buffer.WriteString(m); err != nil {
                // LOGWARN: failed to write to buffer
            }
            for c,_ := range(s.participants) {
                c.stdout <- m
            }
        case m := <-stderr:
            size := s.buffer.Len() + len(m) - BUFFER_MAX
            if size > 0 {
                s.buffer.Next(size)
            }
            if _,err := s.buffer.WriteString(m); err != nil {
                // LOGWARN: failed to write to buffer
            }
            for c,_ := range(s.participants) {
                c.stderr <- m
            }
        case <-done:
            if err := s.proc.Error(); err != nil {
                c.result <- fmt.Sprintf(err)
            } else {
                c.result <- "0"
            }
            return
        case <-time.After(15 * time.Minutes):
            if len(s.participants) == 0 {
                // LOGINFO: idle session, closing
                s.proc.kill(nil)
            }
        }
    }
}

func (s *session) disconnect() {
    close(s.proc)
    h.disconnect <- s
}

type hub struct {
	connections map[*wsshell]bool
	register    chan *wsshell
	unregister  chan *wsshell

	sessions    map[string]*session
	connect     chan *session
	disconnect  chan *session
}

func (h *hub) run() {
    for {
        select {
        case c := <-h.register:
        case c := <-h.unregister:
        case s := <-h.connect:
        case s := <-h.disconnect:
        }
    }



func GetSession(id string) *session {
    if s,ok := h.sessions[id]; ok {
        return s
    }
    return nil
}
