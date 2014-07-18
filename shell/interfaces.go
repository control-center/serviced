// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Shell package.
package shell

import (
	"github.com/googollee/go-socket.io"
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
	ServiceID string
	IsTTY     bool
	SaveAs    string
	Envv      []string
	Mount	  []string
	Command   string
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
