// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Shell package.
package shell

import (
	"github.com/control-center/go-socket.io"

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
	port             string
	dockerRegistry   string
	controllerBinary string
	uiport           string
}
