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

package master

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/facade"

	"errors"
)

var (
	ErrRequestExpired    = errors.New("Authentication request expired")
	ErrRequestFromFuture = errors.New("Authentication request has future timestamp")
)

// GetHost gets the host
func (s *Server) GetHost(hostID string, reply *host.Host) error {
	response, err := s.f.GetHost(s.context(), hostID)
	if err != nil {
		return err
	}
	if response == nil {
		return facade.ErrHostDoesNotExist
	}
	*reply = *response
	return nil
}

// GetHosts returns all Hosts
func (s *Server) GetHosts(empty struct{}, hostReply *[]host.Host) error {
	hosts, err := s.f.GetHosts(s.context())
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

// GetActiveHosts returns all active host ids
func (s *Server) GetActiveHostIDs(empty struct{}, hostReply *[]string) error {
	hosts, err := s.f.GetActiveHostIDs(s.context())
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

// AddHost adds the host
func (s *Server) AddHost(host host.Host, hostReply *[]byte) error {
	privateKey, err := s.f.AddHost(s.context(), &host)
	if err != nil {
		return err
	}
	*hostReply = privateKey
	return nil
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
func (s *Server) FindHostsInPool(poolID string, hostReply *[]host.Host) error {
	hosts, err := s.f.FindHostsInPool(s.context(), poolID)
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

type HostAuthenticationRequest struct {
	HostID    string
	Timestamp int64
	Signature []byte
}

type HostAuthenticationResponse struct {
	Token   string
	Expires int64
}

func (req HostAuthenticationRequest) toMessage() []byte {
	return []byte(fmt.Sprintf("%s:%d", req.HostID, req.Timestamp))
}

func (req HostAuthenticationRequest) valid(publicKeyPEM []byte) error {
	verifier, err := auth.RSAVerifierFromPEM(publicKeyPEM)
	if err != nil {
		return err
	}
	if err := verifier.Verify(req.toMessage(), req.Signature); err != nil {
		return err
	}
	logger := plog.WithField("hostid", req.HostID)
	timeDiff := time.Now().UTC().Unix() - req.Timestamp
	if timeDiff > int64(auth.ClockDriftDelta/time.Second) {
		logger.WithField("clockdriftsec", timeDiff).Error("Delegate time behind master, re-sync clocks")
		return ErrRequestExpired
	} else if -1*timeDiff > int64(auth.ClockDriftDelta/time.Second) {
		logger.WithField("clockdriftsec", timeDiff).Error("Delegate time ahead of master, re-sync clocks")
		return ErrRequestFromFuture
	}

	return nil
}

func (s *Server) AuthenticateHost(req HostAuthenticationRequest, resp *HostAuthenticationResponse) error {
	keypem, err := s.f.GetHostKey(s.context(), req.HostID)
	if err != nil {
		s.f.RemoveHostExpiration(s.context(), req.HostID)
		return err
	}
	if err := req.valid(keypem); err != nil {
		s.f.RemoveHostExpiration(s.context(), req.HostID)
		return err
	}

	host, err := s.f.GetHost(s.context(), req.HostID)
	if err != nil {
		return err
	}
	if host == nil {
		return facade.ErrHostDoesNotExist
	}
	p, err := s.f.GetResourcePool(s.context(), host.PoolID)
	if err != nil {
		return err
	}
	if p == nil {
		return facade.ErrPoolNotExists
	}
	adminAccess := p.Permissions&pool.AdminAccess != 0
	dfsAccess := p.Permissions&pool.DFSAccess != 0
	signed, expires, err := auth.CreateJWTIdentity(host.ID, host.PoolID, adminAccess, dfsAccess, keypem, s.expiration)
	if err != nil {
		s.f.RemoveHostExpiration(s.context(), host.ID)
		return err
	}
	s.f.SetHostExpiration(s.context(), host.ID, expires)
	*resp = HostAuthenticationResponse{signed, expires}
	return nil
}

// Return host's public key
func (s *Server) GetHostPublicKey(hostID string, key *[]byte) error {
	publicKey, err := s.f.GetHostKey(s.context(), hostID)
	*key = publicKey
	return err
}

// Reset and return host's private key
func (s *Server) ResetHostKey(hostID string, key *[]byte) error {
	publicKey, err := s.f.ResetHostKey(s.context(), hostID)
	*key = publicKey
	return err
}

// Given a list of hostsID return if they are authenticated
func (s *Server) HostsAuthenticated(hostIDs []string, res *map[string]bool) error {
	authenticatedHosts := make(map[string]bool)
	if hostIDs == nil || len(hostIDs) == 0 {
		*res = authenticatedHosts
		return nil
	}
	for _, hid := range hostIDs {
		isAuth, _ := s.f.HostIsAuthenticated(s.context(), hid)
		authenticatedHosts[hid] = isAuth
	}
	*res = authenticatedHosts
	return nil
}
