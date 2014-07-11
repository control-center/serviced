// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package docker

import (
	"fmt"

	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

const (
	doForce   = true
	dontForce = false
)

// ListImages wraps client.ListImages, checking the registry and pulling
// any missing or out-dated images and/or tags first.
func ListImages(registry DockerRegistry, client *dockerclient.Client) ([]dockerclient.APIImages, error) {
	remoteImages, err := registry.ListRemoteImageTags()
	if err != nil {
		return []dockerclient.APIImages{}, err
	}

	images, err := client.ListImages(true)
	if err != nil {
		return images, err
	}

	var madeChanges bool

	for _, image := range images {
		remoteImageTags, found := remoteImages[image.ID]
		if !found {
			continue
		}
		for _, repoTag := range image.RepoTags {
			if len(remoteImageTags) == 0 {
				break
			}
			repoName, tag, err := repoNameAndTag(repoTag)
			if err != nil {
				return []dockerclient.APIImages{}, err
			}
			if tag == "" {
				tag = "latest"
			}
			repoTag = fmt.Sprintf("%s:%s", repoName, tag)
			remoteImageTags = removeString(remoteImageTags, repoTag)
		}
		for _, repoTag := range remoteImageTags {
			repoName, tag, err := repoNameAndTag(repoTag)
			opts := dockerclient.TagImageOptions{
				Repo:  fmt.Sprintf("%s/%s", registry, repoName),
				Force: true,
				Tag:   tag,
			}
			if err = client.TagImage(image.ID, opts); err != nil {
				return []dockerclient.APIImages{}, err
			}
			madeChanges = true
		}
		delete(remoteImages, image.ID)
	}

	for imageID, remoteImageTags := range remoteImages {
		if len(remoteImageTags) == 0 {
			continue
		}
		repoTag, remoteImageTags := remoteImageTags[0], remoteImageTags[1:]
		opts := dockerclient.PullImageOptions{
			Repository: repoTag,
			Registry:   registry.String(),
		}
		if err = client.PullImage(opts, auth); err != nil {
			return []dockerclient.APIImages{}, err
		}
		madeChanges = true
		for _, repoTag := range remoteImageTags {
			repoName, tag, err := repoNameAndTag(repoTag)
			opts := dockerclient.TagImageOptions{
				Repo:  fmt.Sprintf("%s/%s", registry, repoName),
				Force: true,
				Tag:   tag,
			}
			if err = client.TagImage(imageID, opts); err != nil {
				return []dockerclient.APIImages{}, err
			}
		}
	}

	if madeChanges {
		return client.ListImages(true)
	}

	return images, nil
}

// CreateContainer wraps client.CreateContainer.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
// Finally, calls client.CreateContainer.
func CreateContainer(registry DockerRegistry, client *dockerclient.Client, opts dockerclient.CreateContainerOptions) (*dockerclient.Container, error) {
	//TODO: Change this to just use Ward's docker stuff.
	if _, err := syncImageFromRegistry(registry, client, opts.Config.Image); err != nil {
		if err != dockerclient.ErrNoSuchImage {
			return nil, err
		}
	}
	return client.CreateContainer(opts)
}

// InspectImage wraps client.InspectImage.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
// Finally, calls client.InspectImage.
func InspectImage(registry DockerRegistry, client *dockerclient.Client, name string) (*dockerclient.Image, error) {
	return syncImageFromRegistry(registry, client, name)
}

// TagImage wraps client.TagImage.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
// Tags the image locally, and in the registry.
func TagImage(registry DockerRegistry, client *dockerclient.Client, name string, opts dockerclient.TagImageOptions) error {
	image, err := syncImageFromRegistry(registry, client, name)
	if err != nil {
		return err
	}

	glog.V(2).Infof("Tagging %s %+v", name, opts)
	if err = client.TagImage(name, opts); err != nil {
		return err
	}

	// Dev short circuit for ZEN-11996
	if !useRegistry {
		return nil
	}

	repoTag := opts.Repo
	if opts.Tag != "" {
		repoTag = repoTag + ":" + opts.Tag
	}

	pushImageToRegistry(registry, client, repoTag, doForce)

	return registry.TagRemoteImage(image.ID, repoTag)
}

// RemoveImage wraps client.RemoveImage, removing the tag from the registry
// too, if necessary. TODO: perhaps also purge images that have no more tags?
func RemoveImage(registry DockerRegistry, client *dockerclient.Client, name string) error {
	localRemoval := client.RemoveImage(name)
	if localRemoval != nil && localRemoval != dockerclient.ErrNoSuchImage {
		return localRemoval
	}

	remoteRemoval := registry.RemoveRemoteImageTag(name)
	if remoteRemoval != nil && remoteRemoval != dockerclient.ErrNoSuchImage {
		return remoteRemoval
	}

	return localRemoval
}

// CommitContainer wraps client.CommitContainer, pushing the image and tag to
// the registry afterwards.
func CommitContainer(registry DockerRegistry, client *dockerclient.Client, opts dockerclient.CommitContainerOptions) (*dockerclient.Image, error) {
	if opts.Repository == "" {
		return nil, fmt.Errorf("must provide repo name when commiting container, so it can be pushed to the registry")
	}

	image, err := client.CommitContainer(opts)
	if err != nil {
		return nil, err
	}

	repoName := opts.Repository
	if opts.Tag != "" {
		repoName += ":" + opts.Tag
	}

	if err = pushImageToRegistry(registry, client, repoName, doForce); err != nil {
		return nil, err
	}

	return image, nil
}

// ImportImage wraps client.ImportImage, pushing the image and tag to the
// registry afterwards.
func ImportImage(registry DockerRegistry, client *dockerclient.Client, opts dockerclient.ImportImageOptions) error {
	if opts.Repository == "" {
		return fmt.Errorf("must provide repo name when importing image, so it can be pushed to the registry")
	}

	if err := client.ImportImage(opts); err != nil {
		return err
	}

	repoName := opts.Repository

	if opts.Tag != "" {
		repoName += ":" + opts.Tag
	}

	if err := pushImageToRegistry(registry, client, repoName, doForce); err != nil {
		return err
	}

	return nil
}
