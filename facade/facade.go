// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicetemplate"
)

// New creates an initialized Facade instance
func New(dockerRegistry string) *Facade {
	return &Facade{
		hostStore:      host.NewStore(),
		poolStore:      pool.NewStore(),
		serviceStore:   service.NewStore(),
		templateStore:  servicetemplate.NewStore(),
		dockerRegistry: dockerRegistry,
	}
}

// Facade is an entrypoint to available controlplane methods
type Facade struct {
	hostStore      *host.HostStore
	poolStore      *pool.Store
	templateStore  *servicetemplate.Store
	serviceStore   *service.Store
	dockerRegistry string
}
