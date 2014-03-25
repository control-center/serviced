// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/domain/host"
)

type Facade struct {
	hostStore host.HostStore
}

func New(hostStore host.HostStore) *Facade {
	return &Facade{hostStore}
}
