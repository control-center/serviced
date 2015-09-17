// Copyright 2015 The Serviced Authors.
// Use of f source code is governed by a

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/serviceimage"

	"time"
)

// GetServiceImage finds a ServiceImage
func (f *Facade) GetServiceImage(ctx datastore.Context, imageID string) (*serviceimage.ServiceImage, error) {
	image := serviceimage.ServiceImage{}
	if err := f.imageStore.Get(ctx, serviceimage.ImageKey(imageID), &image); err != nil {
		return nil, err
	}

	return &image, nil
}

func (f *Facade) PushServiceImage(ctx datastore.Context, image *serviceimage.ServiceImage) error {
	if image == nil {
		return datastore.ErrNilEntity
	}

	if image.CreatedAt.IsZero() {
		image.Status = serviceimage.IMGCreated
		image.Error = ""
		image.CreatedAt = time.Now()
		image.DeployedAt = time.Time{}
		if err := f.imageStore.Put(ctx, image.Key(), image); err != nil {
			image.CreatedAt = time.Time{}
			image.Status = serviceimage.IMGFailed
			image.Error = err.Error()
			return err
		}
	}

	pushErr := f.registry.PushImage(image.ImageID)
	if pushErr == nil {
		image.Status = serviceimage.IMGDeployed
		image.Error = ""
		image.DeployedAt = time.Now()
	} else {
		image.Status = serviceimage.IMGFailed
		image.Error = pushErr.Error()
		image.DeployedAt = time.Time{}
	}

	putErr := f.imageStore.Put(ctx, image.Key(), image)
	if putErr != nil {
		image.Status = serviceimage.IMGFailed
		image.Error = putErr.Error()
		return putErr
	}
	return pushErr
}

