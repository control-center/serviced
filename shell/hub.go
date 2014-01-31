package shell

const (
	BUFFER_MAX int64 = 8192    // 8KB
)

type Reader interface {
    Reader(size_t int)
    Write(data []byte) (int,error)
    StdoutPipe() chan string
    StderrPipe() chan string
    ExitedPipe() chan bool
    Resize(cols, rows *int) error
    Error() error
    Signal(sig int) error
    Kill() error
    Close()
}

type session struct {
	id           string
	host         *wsshell
	participants map[*wsshell]bool
	proc         process
	buffer       bytes.Buffer

	promote chan *wsshell
}

func StartSession(conn *wsshell, proc process) (*session, error) {
	var id string

	// Generate the session id
	if info, err := json.Marshal(proc.info()); err != nil {
		return nil, err
	} else {
		h := md5.New()
		io.WriteString(h, info)
		id = fmt.Sprintf("%x", h.Sum(nil))
	}

	s := session{
		id:           id,
		host:         conn,
		participants: {conn: true},
		proc:         proc,
        promote: make(chan *wsshell, 16)
	}

	h.connect <- s
	go s.send()
	return &s, nil
}

func OpenSession(conn *wsshell, proc process) (*session, error) {
	// Lookup the session id
	s, ok := h.sessions[id]
	if !ok {
		return nil, errors.New("session not found")
	}

	// Send the buffer to the connection
    conn.stdout <- s.buffer.String()

	// Add the connection to the session
	s.participants[conn] = true

	// Promote to host if only user
    if len(s.participants) == 1 {
        s.promote <- conn
    }

	return s
}

func (s *session) send() {
	stdout := s.proc.stdoutChan()
	stderr := s.proc.stderrChan()
	exited := s.proc.exitedChan()

	go s.proc.reader(BUFFER_MAX)
	defer s.close()
	for {
		select {
		case m := <-stdout:
			size := s.buffer.Len() + len(m) - BUFFER_MAX
			if size > 0 {
                if b := s.buffer.Next(size); b[len(b) - 1] != '\n' {
                    s.buffer.ReadString('\n')
                }
			}
			if _, err := s.buffer.WriteString(m); err != nil {
				// WARN: failed to write to buffer
			}
			for c, _ := range s.participants {
				c.stdout <- m
			}
		case m := <-stderr:
			size := s.buffer.Len() + len(m) - BUFFER_MAX
			if size > 0 {
                if b := s.buffer.Next(size); b[len(b) - 1] != '\n' {
                    s.buffer.ReadString('\n')
                }
			}
			if _, err := s.buffer.WriteString(m); err != nil {
				// WARN: failed to write to buffer
			}
			for c, _ := range s.participants {
				c.stderr <- m
			}
		case <-exited:
			if err := s.proc.getError(); err != nil {
				c.result <- fmt.Sprint(err)
			} else {
				c.result <- "0"
			}
			return
		case <-time.After(15 * time.Minutes):
			if len(s.participants) == 0 {
				// INFO: idle session, closing
				s.proc.kill()
			}
		}
	}
}

func (s *session) write(conn *wsshell, data []byte) (n, error) {
	if conn != s.host {
		return 0, errors.New("permission error")
	}
	return s.proc.write(data)
}

func (s *session) resize(conn *wsshell, cols, rows *int) error {
	if conn != s.host {
		return errors.New("permission error")
	}
	return s.proc.resize(cols, rows)
}

func (s *session) signal(conn *wsshell, s int) error {
	if conn != s.host {
		return errors.New("permission error")
	}
	return s.proc.signal(s)
}

func (s *session) demote(conn *wsshell) {
    if conn == nil || conn != s.host {
        return
    }

    s.host = nil
    for c := range s.promote {
        if s.participants[c] {
            s.host = c
            c.host <- true
            return
        }
    }
}

func (s *session) disconnect(conn *wsshell) {
	delete(s.participants, conn)
    if len(s.participants) == 1 {
        for c,_ := range s.participants {
            s.promote <- c
        }
    }
    go s.demote(conn)
}

func (s *session) close() {
    close(s.promote)
	s.proc.close()
	h.disconnect <- s
}

type hub struct {
	connections map[*wsshell]bool
	register    chan *wsshell
	unregister  chan *wsshell

	sessions   map[string]*session
	connect    chan *session
	disconnect chan *session
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			if c.sid != nil {
				c.sid.disconnect(c)
			}
			delete(h.connections, c)
		case s := <-h.connect:
			h.sessions[s.id] = s
		case s := <-h.disconnect:
			delete(h, s.id)
		}
	}
}
