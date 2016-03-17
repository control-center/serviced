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

package facade

import (
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/zenoss/glog"
)

const (
	userLockTimeout = time.Second
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
