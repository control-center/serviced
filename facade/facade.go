// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
)

// New creates an initialized Facade instance
func New() *Facade {
	return &Facade{
		host.NewStore(),
		pool.NewStore(),
	}
}

// Facade is an entrypoint to available controlplane methods
type Facade struct {
	hostStore *host.HostStore
	poolStore *pool.Store
}
