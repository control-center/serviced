// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package master

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
)

// NewServer creates a new serviced master rpc server
func NewServer(f *facade.Facade) *Server {
	return &Server{f}
}

// Server is the RPC type for the master(s)
type Server struct {
	f *facade.Facade
}

func (s *Server) context() datastore.Context {
	//here in case we ever need to create a per request context
	return datastore.Get()

}
