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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package node

import (
	"github.com/control-center/serviced/rpc/rpcutils"
)

// AgentClient is an interface that the serviced agent implements to provide
// information about the host it is running on.
type AgentClient struct {
	addr      string
	rpcClient rpcutils.Client
}

// Create a new AgentClient.
func NewAgentClient(addr string) (s *AgentClient, err error) {
	client, err := rpcutils.GetCachedClient(addr)
	if err != nil {
		return nil, err
	}
	s = new(AgentClient)
	s.addr = addr
	s.rpcClient = client
	return s, nil
}
