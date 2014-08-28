// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
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
