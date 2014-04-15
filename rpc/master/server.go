// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/facade"
)

func NewServer() *Server {
	return &Server{facade.New()}
}

// Server is the RPC type for the master(s)
type Server struct {
	f *facade.Facade
}

func (s *Server) context() datastore.Context {
	//here in case we ever need to create a per request context
	return datastore.Get()

}
