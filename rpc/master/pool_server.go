// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package master

import (
	"github.com/zenoss/serviced/facade"

	"errors"
)

// GetPoolIPs gets all ips available to a Pool
func (s *Server) GetPoolIPs(poolID string, reply *facade.PoolIPs) error {
	response, err := s.f.GetPoolIPs(s.context(), poolID)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("host not found")
	}
	*reply = *response
	return nil
}


