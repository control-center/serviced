// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package master

import (
	"github.com/control-center/serviced/domain/host"

	"errors"
)

// GetHost gets the host
func (s *Server) GetHost(hostID string, reply *host.Host) error {
	response, err := s.f.GetHost(s.context(), hostID)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("hosts_server.go host not found")
	}
	*reply = *response
	return nil
}

// GetHosts returns all Hosts
func (s *Server) GetHosts(empty struct{}, hostReply *[]*host.Host) error {
	hosts, err := s.f.GetHosts(s.context())
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

// GetActiveHosts returns all active host ids
func (s *Server) GetActiveHosts(empty struct{}, hostReply *[]string) error {
	hosts, err := s.f.GetActiveHosts(s.context())
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

// AddHost adds the host
func (s *Server) AddHost(host host.Host, _ *struct{}) error {
	return s.f.AddHost(s.context(), &host)
}

// UpdateHost updates the host
func (s *Server) UpdateHost(host host.Host, _ *struct{}) error {
	return s.f.UpdateHost(s.context(), &host)
}

// RemoveHost removes the host
func (s *Server) RemoveHost(hostID string, _ *struct{}) error {
	return s.f.RemoveHost(s.context(), hostID)
}

// FindHostsInPool  Returns all Hosts in a pool
func (s *Server) FindHostsInPool(poolID string, hostReply *[]*host.Host) error {
	hosts, err := s.f.FindHostsInPool(s.context(), poolID)
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}
