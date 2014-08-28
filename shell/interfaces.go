// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Shell package.
package shell

import (
	"github.com/googollee/go-socket.io"

	"time"
)

// Describes whether a process terminated normally or abnormally
type Termination int

const (
	NORMAL   Termination = iota // Process terminated normally
	ABNORMAL                    // Process terminated abnormally
)

type ProcessServer struct {
	sio   *socketio.SocketIOServer
	actor ProcessActor
}

type ProcessConfig struct {
	ServiceID   string
	IsTTY       bool
	SaveAs      string
	Envv        []string
	Mount       []string
	Command     string
	LogToStderr bool // log the command output for stderr
	LogStash    struct {
		Enable        bool          //enable log stash
		SettleTime    time.Duration //how long to wait for log stash to flush logs before exiting, ex. 1s
		IdleFlushTime time.Duration //interval log stash flushes its buffer, ex 1ms
	}
}

type Result struct {
	ExitCode    int
	Error       string
	Termination Termination
}

type ProcessInstance struct {
	disconnected bool
	closed       bool

	Stdin  chan byte
	Stdout chan byte
	Stderr chan byte
	Result chan Result
}

type ProcessActor interface {
	Exec(*ProcessConfig) *ProcessInstance
	onDisconnect(*socketio.NameSpace)
}

type Forwarder struct {
	addr string
}

type Executor struct {
	port           string
	dockerRegistry string
}
