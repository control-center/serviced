// Copyright 2015 The Serviced Authors.
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
	"errors"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/govpool"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

var (
	ErrRemotePoolExists = errors.New("facade: remote pool exists")
	ErrGovPoolExists    = errors.New("facade: governed pool exists")
	ErrGovPoolNotExists = errors.New("facade: governed pool does not exist")
	ErrPoolHasServices  = errors.New("facade: resource pool has services")
)

// AddGovernedPool adds a governor to an existing resource pool
func (f *Facade) AddGovernedPool(ctx datastore.Context, poolID string, msg string) error {
	// decode the message
	var packet utils.PacketData
	if err := utils.DecodePacket(msg, &packet); err != nil {
		glog.Errorf("Could not decode packet %s: %s", msg, err)
		return err
	}

	// verify the pool exists
	if pool, err := f.GetResourcePool(ctx, poolID); err != nil {
		glog.Errorf("Could not look up resource pool %s: %s", poolID, err)
		return err
	} else if pool == nil {
		glog.Errorf("Pool %s does not exist", poolID)
		return ErrPoolNotExists
	}

	// verify the pool does not already have a governor
	if pool, err := f.GetGovernedPoolByPoolID(ctx, poolID); err != nil {
		glog.Errorf("Could not look up governor for %s: %s", poolID, err)
		return err
	} else if pool != nil {
		glog.Errorf("Pool %s already has a governor", poolID)
		return ErrGovPoolExists
	}

	// verify the pool does not have any services
	if _, svcs, err := f.GetServicesByPool(ctx, poolID, NoServiceFilter); err != nil {
		glog.Errorf("Could not look up services for %s: %s", poolID, err)
		return err
	} else if svccount := len(svcs); svccount > 0 {
		glog.Errorf("Found %d services associated with pool %s; cannot add governor", svccount, poolID)
		return ErrPoolHasServices
	}

	// create the governed pool and store the secret
	// TODO: this should be transactional
	store := govpool.NewStore()
	gpool := &govpool.GovernedPool{
		PoolID:        poolID,
		RemotePoolID:  packet.RemotePoolID,
		RemoteAddress: packet.RemoteAddress,
	}
	if err := store.Put(ctx, gpool); err != nil {
		glog.Errorf("Could not create governed pool: %s", err)
		return err
	} else if err := f.addPoolSecret(ctx, packet.RemotePoolID, packet.Secret); err != nil {
		defer f.RemoveGovernedPool(ctx, poolID)
		glog.Errorf("Could not create governed pool: %s", err)
		return err
	}
	return nil
}

// RemoveGovernedPool removes the governor from a resource pool
func (f *Facade) RemoveGovernedPool(ctx datastore.Context, poolID string) error {
	// verify this pool is a governor pool
	pool, err := f.GetGovernedPoolByPoolID(ctx, poolID)
	if err != nil {
		glog.Errorf("Could not look up governed pool for %s: %s", poolID, err)
		return err
	} else if pool == nil {
		return nil
	}

	// delete the pool and its secret
	// TODO: this should be transactional
	store := govpool.NewStore()
	if err := f.removePoolSecret(ctx, pool.RemotePoolID); err != nil {
		glog.Errorf("Error trying to delete pool secret for pool %s: %s", poolID, err)
		return err
	} else if err := store.Delete(ctx, pool.RemotePoolID); err != nil {
		glog.Errorf("Could not delete governed pool %s: %s", poolID, err)
		return err
	}
	return nil
}

// GetGovernedPool searches for a governed pool by its remote pool id
func (f *Facade) GetGovernedPool(ctx datastore.Context, remotePoolID string) (*govpool.GovernedPool, error) {
	store := govpool.NewStore()
	pool, err := store.Get(ctx, remotePoolID)
	if datastore.IsErrNoSuchEntity(err) {
		return nil, nil
	}
	return pool, err
}

// GetGovernedPoolByPoolID searches for a governed pool by its resource pool id
func (f *Facade) GetGovernedPoolByPoolID(ctx datastore.Context, poolID string) (*govpool.GovernedPool, error) {
	store := govpool.NewStore()
	return store.GetByPoolID(ctx, poolID)
}

// GetGovernedPools returns all governed pools
func (f *Facade) GetGovernedPools(ctx datastore.Context) ([]govpool.GovernedPool, error) {
	store := govpool.NewStore()
	return store.GetGovernedPools(ctx)
}

// addPoolSecret is a placeholder for writing the pool secret in a separate datastore
func (f *Facade) addPoolSecret(ctx datastore.Context, remotePoolID, secret string) error {
	// verify the remote pool id does not exist
	if secret, err := f.getPoolSecret(ctx, remotePoolID); err != nil {
		glog.Errorf("Could not look up secret for pool %s: %s", remotePoolID, err)
		return err
	} else if secret != "" {
		glog.Errorf("Remote pool %s exists", remotePoolID)
		return ErrRemotePoolExists
	}

	// TODO: add a pool secret given the remote pool id and the secret
	return nil
}

// removePoolSecret is a placeholder for removing the pools secret from its
// respective datastore
func (f *Facade) removePoolSecret(ctx datastore.Context, remotePoolID string) error {
	// TODO: delete a pool secret given a remote pool id
	return nil
}

// getPoolSecret is a placeholder for searching for a pool secret by its remote
// pool id
func (f *Facade) getPoolSecret(ctx datastore.Context, remotePoolID string) (string, error) {
	// TODO: get the pool secret given a remote pool id, return empty string if
	// secret not found.
	return "", nil
}