// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/domain/host"
)

// New creates an initialized  Facade instance
func New(hostStore host.HostStore) *Facade {
	return &Facade{hostStore}
}

// Facade is an entrypoint to available controlpane methods
type Facade struct {
	hostStore host.HostStore
}
