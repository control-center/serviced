// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package commons

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

const (
	doForce   = true
	dontForce = false
	snr       = "SERVICED_NOREGISTRY"
)

var noregistry bool

// Check for the SERVICED_NOREGISTRY environment variable on initialization.
// If it is set, to anything but the empty string, images won't be pushed or
// pulled from a local registry.
func init() {
	if os.Getenv(snr) != "" {
		noregistry = true
	}
}

// ListImages wraps client.ListImages, checking the registry and pulling
// any missing or out-dated images and/or tags first.
func ListImages(registry DockerRegistry, client *dockerclient.Client) (i []dockerclient.APIImages, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.ListImages error: %s", e)
		}
	}()

	remoteImages, err := registry.ListRemoteImageTags()
	madeChanges := false
	if err != nil {
		return []dockerclient.APIImages{}, err
	}
	images, err := client.ListImages(true)
	if err != nil {
		return images, err
	}
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

func pullImageFromRegistry(registry DockerRegistry, client *dockerclient.Client, name string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.pullImageFromRegistry error: %s", e)
		}
	}()

	// Dev short circuit for ZEN-11996
	if noregistry {
		return nil
	}

	repoName, tag, err := repoNameAndTag(name)
	if err != nil {
		return err
	}
	if tag == "" {
		tag = "latest"
	}
	opts := dockerclient.PullImageOptions{
		Repository: repoName,
		Tag:        tag,
		Registry:   registry.String(),
	}
	return client.PullImage(opts, auth)
}

func pushImageToRegistry(registry DockerRegistry, client *dockerclient.Client, name string, force bool) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.pushImageToRegistry name: %s, force: %t, error: %s", name, force, e)
		} else {
			glog.V(2).Infof("done pushing docker image %s to %s", name, registry)
		}
	}()

	// Dev short circuit for ZEN-11996
	if noregistry {
		return nil
	}

	glog.V(2).Infof("pushing docker image %s to %s...", name, registry)
	imageID, err := ParseImageID(name)
	if err != nil {
		return fmt.Errorf("invalid image name: %s, error: %s", name, err)
	}
	repoName := registry.String() + "/"
	if imageID.User != "" {
		repoName += imageID.User + "/"
	}
	repoName += imageID.Repo
	fullName := repoName
	if imageID.Tag != "" {
		fullName += ":" + imageID.Tag
	}

	hostAndPort := imageID.Host
	if imageID.Port != 0 {
		hostAndPort += ":" + fmt.Sprintf("%d", imageID.Port)
	}

	if hostAndPort != registry.String() {
		image, err := client.InspectImage(name)
		if err != nil {
			return err
		}
		if force == dontForce {
			// check to make sure that either the full (adjusted) name doesn't exist
			// locally, or if it does exist, that it points to the exact same image.
			foundImage, err := client.InspectImage(fullName)
			if err != nil && err != dockerclient.ErrNoSuchImage {
				return err
			}
			if err == nil && foundImage.ID != image.ID {
				err = fmt.Errorf("refusing to push image %s (%s) to %s because %s is a different image (%s) locally", name, image.ID, registry, repoName, foundImage.ID)
				return err
			}
		}
		// tag the image locally
		tagOpts := dockerclient.TagImageOptions{
			Repo:  repoName,
			Force: true,
			Tag:   imageID.Tag,
		}
		glog.V(2).Infof("tagging image %s: %+v", image.ID, tagOpts)
		if err = client.TagImage(image.ID, tagOpts); err != nil {
			return err
		}
	}
	opts := dockerclient.PushImageOptions{
		Name:     repoName,
		Tag:      imageID.Tag,
		Registry: registry.String(),
	}
	glog.Infof("pushing image: %+v", opts)
	return client.PushImage(opts, auth)
}

// syncImageFromRegistry gets the local docker image to match the registry.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
func syncImageFromRegistry(registry DockerRegistry, client *dockerclient.Client, name string) (i *dockerclient.Image, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.syncImageFromRegistry error: %s", e)
		}
	}()
	image, err := client.InspectImage(name)
	if err != nil && err != dockerclient.ErrNoSuchImage {
		return nil, err
	}

	// Dev short circuit for ZEN-11996
	if noregistry {
		return image, err
	}

	uuid, err := registry.GetRemoteRepoTag(name)
	if err != nil {
		if err != dockerclient.ErrNoSuchImage {
			return nil, err
		}
		// Well, let's push then...
		if err = pushImageToRegistry(registry, client, name, dontForce); err != nil {
			return nil, err
		}
		uuid, err = registry.GetRemoteRepoTag(name)
		if err != nil {
			return nil, err
		}
	}
	if image == nil || image.ID != uuid {
		if err = pullImageFromRegistry(registry, client, name); err != nil {
			return nil, fmt.Errorf("failed to pull image \"%s\" from registry (%s): %s", name, registry, err)
		}
		// Assert that it actually got pulled...
		image, err = client.InspectImage(name)
		if err != nil {
			return nil, err
		}
		if image == nil || image.ID != uuid {
			return nil, fmt.Errorf("pulled image \"%s\" from registry (%s), but image id (%s) does not match", name, registry, uuid)
		}
	}
	return image, err
}

// CreateContainer wraps client.CreateContainer.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
// Finally, calls client.CreateContainer.
func CreateContainer(registry DockerRegistry, client *dockerclient.Client, opts dockerclient.CreateContainerOptions) (c *dockerclient.Container, e error) {
	//TODO: Change this to just use Ward's docker stuff.
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.CreateContainer error: %s", e)
		}
	}()
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
func InspectImage(registry DockerRegistry, client *dockerclient.Client, name string) (i *dockerclient.Image, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.InspectImage error: %s", e)
		}
	}()
	return syncImageFromRegistry(registry, client, name)
}

// TagImage wraps client.TagImage.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
// Tags the image locally, and in the registry.
func TagImage(registry DockerRegistry, client *dockerclient.Client, name string, opts dockerclient.TagImageOptions) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.TagImage error: %s", e)
		}
	}()
	image, err := syncImageFromRegistry(registry, client, name)
	if err != nil {
		return err
	}

	glog.V(2).Infof("Tagging %s %+v", name, opts)
	if err = client.TagImage(name, opts); err != nil {
		return err
	}

	// Dev short circuit for ZEN-11996
	if noregistry {
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
func RemoveImage(registry DockerRegistry, client *dockerclient.Client, name string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.RemoveImage error: %s", e)
		}
	}()
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
func CommitContainer(registry DockerRegistry, client *dockerclient.Client, opts dockerclient.CommitContainerOptions) (i *dockerclient.Image, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.CommitContainer error: %s", e)
		}
	}()
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
func ImportImage(registry DockerRegistry, client *dockerclient.Client, opts dockerclient.ImportImageOptions) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.ImportImage error: %s", e)
		}
	}()
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

// DockerRegistry holds docker images, organized with repositories and tags.
type DockerRegistry interface {
	String() string
	ListRemoteRepos() ([]string, error)
	ListRemoteRepoTags(repo string) (map[string]string, error)
	GetRemoteRepoTag(repoTag string) (string, error)
	ListRemoteImageTags() (map[string][]string, error)
	TagRemoteImage(id, repoTag string) error
	RemoveRemoteImageTag(repoTag string) error
}

type dockerRegistry struct {
	hostAndPort string
}

// NewDockerRegistry creates a new DockerRegistry.
// hostAndPort must have either a . or a : in it. Example: "localhost:5000".
func NewDockerRegistry(hostAndPort string) (DockerRegistry, error) {
	result := dockerRegistry{
		hostAndPort: hostAndPort,
	}
	return &result, nil
}

// String returns the host and port of the registry.
func (r *dockerRegistry) String() string {
	return r.hostAndPort
}

// ListRemoteRepos returns a list of all repos in the registry.
func (r *dockerRegistry) ListRemoteRepos() (list []string, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.ListRemoteRepos error: %s", e)
		}
	}()

	var response searchResponse
	if err := r.get("/v1/search", &response); err != nil {
		return []string{}, err
	}
	result := []string{}
	for _, entry := range response.results {
		result = append(result, entry.name)
	}
	return result, nil
}

// ListRemoteRepoTags returns a map from tag to image ID (UUID) for all tags
// in the registry associated with the given docker repo.
func (r *dockerRegistry) ListRemoteRepoTags(repo string) (t map[string]string, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.ListRemoteRepoTags error: %s", e)
		}
	}()

	repoName, _, err := repoNameAndTag(repo)
	if err != nil {
		return map[string]string{}, err
	}
	path := "/v1/repositories/" + repoName + "/tags"
	var response map[string]string
	if err = r.get(path, &response); err != nil {
		return map[string]string{}, err
	}
	return response, nil
}

// GetRemoteRepoTag returns the UUID of the given image.
func (r *dockerRegistry) GetRemoteRepoTag(repoTag string) (t string, e error) {
	// FIXME: this method is never used out side of this file and so shouldn't be exported.
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.GetRemoteRepoTag error: %s", e)
		}
	}()

	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		return "", err
	}
	if tag == "" {
		tag = "latest"
	}
	path := "/v1/repositories/" + repoName + "/tags/" + tag
	var response string
	if err = r.get(path, &response); err != nil {
		return "", err
	}
	return response, nil
}

// ListRemoteImageTags returns a map from image ID (UUID) to a list of image
// names, where each name is like "namespace/repository:tag".
func (r *dockerRegistry) ListRemoteImageTags() (t map[string][]string, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.ListRemoteImageTags error: %s", e)
		}
	}()

	result := make(map[string][]string)

	repos, err := r.ListRemoteRepos()
	if err != nil {
		return result, err
	}
	for _, repo := range repos {
		repoTags, err := r.ListRemoteRepoTags(repo)
		if err != nil {
			return result, err
		}
		for tag, imageID := range repoTags {
			tags, found := result[imageID]
			if !found {
				tags = []string{}
			}
			result[imageID] = append(tags, fmt.Sprintf("%s:%s", repo, tag))
		}
	}
	return result, nil
}

// TagRemoteImage tags the given image in the registry with the given tag.
// imageID: the UUID of the image
// repoTag: like "namespace/repository:tag" or "repository:tag"
func (r *dockerRegistry) TagRemoteImage(imageID, repoTag string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.TagRemoteImage error: %s", e)
		}
	}()

	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		return err
	}
	if tag == "" {
		tag = "latest"
	}
	path := "/v1/repositories/" + repoName + "/tags/" + tag
	return r.put(path, imageID)
}

// RemoveRemoteImageTag removes the given tag from the registry.
// It is not considered an error if the tag already does not exist.
func (r *dockerRegistry) RemoveRemoteImageTag(repoTag string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.RemoveRemoteImageTag error: %s", e)
		}
	}()

	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		return err
	}
	if tag == "" {
		return fmt.Errorf("no tag was specified for removal: %s", repoTag)
	}
	path := "/v1/repositories/" + repoName + "/tags/" + tag
	err = r.delete(path)
	if err == dockerclient.ErrNoSuchImage {
		return nil
	}
	return err
}

var auth = dockerclient.AuthConfiguration{}

func (r *dockerRegistry) url(path string) string {
	result := r.hostAndPort
	if len(result) < 4 || result[:4] != "http" {
		result = "http://" + result
	}
	return strings.TrimRight(result, "/") + "/" + strings.TrimLeft(path, "/")
}

func (r *dockerRegistry) get(path string, v interface{}) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.get error: %s", e)
		}
	}()
	var (
		resp *http.Response
		err  error
	)
	req := r.url(path)
	attempts := 0
	for {
		attempts++
		resp, err = http.Get(req)
		if err == nil {
			break
		}
		if attempts < 3 {
			continue
		}
		return fmt.Errorf("failed to get response. attempts=%d, req=%s, err=%s", attempts, req, err)
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 404 {
		return dockerclient.ErrNoSuchImage
	}
	if resp.StatusCode > 304 {
		return fmt.Errorf("got error response. req=%s, code=%d, body=%s", req, resp.StatusCode, bytes)
	}
	if err = json.Unmarshal(bytes, v); err != nil {
		return fmt.Errorf("could not parse response. req=%s, body=%s", req, bytes)
	}

	return nil
}

func (r *dockerRegistry) put(path string, data interface{}) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.put error: %s", e)
		}
	}()
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", r.url(path), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return dockerclient.ErrNoSuchImage
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if res.StatusCode > 304 {
		return fmt.Errorf("status: %d, error: %s", res.StatusCode, bodyBytes)
	}
	return nil
}

func (r *dockerRegistry) delete(path string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.delete error: %s", e)
		}
	}()
	req, err := http.NewRequest("DELETE", r.url(path), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return dockerclient.ErrNoSuchImage
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if res.StatusCode > 304 {
		return fmt.Errorf("status: %d, error: %s", res.StatusCode, bodyBytes)
	}
	return nil
}

// repoNameAndTag parses the image name, removing any host, port,
// and tag, and returns a string like "(user|'library')/repo" for the
// repo name, and another string containing just the tag (or "" if none).
func repoNameAndTag(imageName string) (string, string, error) {
	s := make([]string, 0, 5)
	imageID, err := ParseImageID(imageName)
	if err != nil {
		return "", "", fmt.Errorf("invalid image name: %s, error: %s", imageName, err)
	}
	if imageID.User == "" {
		s = append(s, "library/")
	} else {
		s = append(s, imageID.User, "/")
	}
	s = append(s, imageID.Repo)
	return strings.Join(s, ""), imageID.Tag, nil
}

type searchResponse struct {
	results []searchResponseEntry
}

type searchResponseEntry struct {
	description string
	name        string
}

func removeString(a []string, s string) []string {
	result := make([]string, 0, len(a))
	for _, e := range a {
		if e != s {
			result = append(result, e)
		}
	}
	return result
}
