// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"fmt"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/registry"
	"github.com/zenoss/glog"
)

// GetRegistryImage returns information about an image that is stored in the
// docker registry index.
// e.g. GetRegistryImage(ctx, "library/reponame:tagname")
func (f *Facade) GetRegistryImage(ctx datastore.Context, image string) (*registry.Image, error) {
	rImage, err := f.registryStore.Get(ctx, image)
	if err != nil {
		return nil, err
	}
	return rImage, nil
}

// SetRegistryImage creates/updates an image in the docker registry index.
func (f *Facade) SetRegistryImage(ctx datastore.Context, rImage *registry.Image) error {
	if err := f.registryStore.Put(ctx, rImage); err != nil {
		return err
	}
	err := f.zzk.SetRegistryImage(rImage)
	if err != nil {
		return err
	}
	svcs, err := f.GetServicesByImage(ctx, rImage.String())
	if err != nil {
		return fmt.Errorf("error getting services: %s", err)
	}
	for _, svc := range svcs {
		if svc.ID == "" {
			continue
		}
		states, err := f.GetServiceStates(ctx, svc.ID)
		if err != nil {
			return fmt.Errorf("unable to retrieve service states for %s: %s", svc.ID, err)
		}
		for _, state := range states {
			if state.ImageUUID != rImage.UUID {
				state.InSync = false
				glog.V(1).Infof("Updating InSync for service %s", state.ID)
				if err = f.zzk.UpdateServiceState(svc.PoolID, &state); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// DeleteRegistryImage removes an image from the docker registry index.
// e.g. DeleteRegistryImage(ctx, "library/reponame:tagname")
func (f *Facade) DeleteRegistryImage(ctx datastore.Context, image string) error {
	if err := f.registryStore.Delete(ctx, image); err != nil {
		return err
	}
	if err := f.zzk.DeleteRegistryImage(registry.Key(image).ID()); err != nil {
		return err
	}
	return nil
}

// GetRegistryImages returns all the image that are in the docker registry
// index.
func (f *Facade) GetRegistryImages(ctx datastore.Context) ([]registry.Image, error) {
	rImages, err := f.registryStore.GetImages(ctx)
	if err != nil {
		return nil, err
	}
	return rImages, nil
}

// SearchRegistryLibrary searches the docker registry index for images at a
// particular library and tag.
// e.g. library/reponame:tagname => SearchRegistryLibrary("library", "tagname")
func (f *Facade) SearchRegistryLibraryByTag(ctx datastore.Context, library, tagname string) ([]registry.Image, error) {
	rImages, err := f.registryStore.SearchLibraryByTag(ctx, library, tagname)
	if err != nil {
		return nil, err
	}
	return rImages, nil
}

// SyncRegistryImages makes sure images on es are in sync with zk.  If force is
// enabled, all images are reset.
func (f *Facade) SyncRegistryImages(ctx datastore.Context, force bool) error {
	if err := f.DFSLock(ctx).LockWithTimeout("sync registry images", userLockTimeout); err != nil {
		glog.Warningf("Cannot sync registry images: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	// get all the images that are currently in the index
	rImages, err := f.GetRegistryImages(ctx)
	if err != nil {
		return err
	}
	// we aren't going to try to sync deletes because that can get too messy;
	// only adds and updates
	for _, rImage := range rImages {
		img, err := f.zzk.GetRegistryImage(rImage.ID())
		if err != client.ErrNoNode && err != nil {
			return err
		}
		// only update the images where the uuid has changed and from the
		// upstream only, to make sure we don't override any changes that
		// occur out of band from the sync.  If force is set, then it is okay
		// to blanket reset everything.
		if force || img == nil || img.UUID != rImage.UUID {
			if err := f.SetRegistryImage(ctx, &rImage); err != nil {
				return err
			}
		}
	}
	return nil
}
