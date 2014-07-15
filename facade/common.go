// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
)

type eventContext map[string]interface{}

func newEventCtx() eventContext {
	return make(map[string]interface{})
}

func (f *Facade) beforeEvent(event beforeEvent, eventCtx eventContext, entity interface{}) error {
	//TODO: register and deal with handlers
	return nil
}
func (f *Facade) afterEvent(event afterEvent, eventCtx eventContext, entity interface{}, err error) error {
	//TODO: register and deal with handlers
	return nil
}

type beforeEvent string
type afterEvent string

// delete common code for removing an entity and publishes events
func (f *Facade) delete(ctx datastore.Context, ds datastore.EntityStore, key datastore.Key, be beforeEvent, ae afterEvent) error {
	glog.V(2).Infof("Facade.delete: %s:%s", key.Kind(), key.ID())
	ec := newEventCtx()
	err := f.beforeEvent(be, ec, key.ID())
	if err == nil {
		err = ds.Delete(ctx, key)
	}
	defer f.afterEvent(ae, ec, key.ID(), err)
	return err
}
